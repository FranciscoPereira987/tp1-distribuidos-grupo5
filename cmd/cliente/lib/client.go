package lib

import (
	"errors"
	"io"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/conection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/reader"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
	"github.com/sirupsen/logrus"
)

type ClientConfig struct {
	ResultsDir string
	//Query1File string
	Query2File string
	//Query3File string
	//Query4File string

	dataFile   string
	coordsFile string

	ServerData    conection.Conn
	ServerResults conection.Conn
}

type Client struct {
	config       ClientConfig
	writer       *utils.ResultWriter
	coordsReader reader.Reader
	dataReader   reader.Reader
	dataConn     *protocol.Protocol
	resultsConn  *protocol.Protocol

	resultsEnd     chan bool
	dataSendingEnd chan bool
}

func getFiles(config ClientConfig) []string {
	files := []string{config.Query2File}
	return files
}

func getResultTypes() []typing.Type {
	types := []typing.Type{&distance.AirportDataType{}}
	return types
}

func TypesAsNumbered() []utils.Numbered {
	types := getResultTypes()
	var numbered []utils.Numbered
	for _, value := range types {
		numbered = append(numbered, value)
	}
	return numbered
}

func NewClient(config ClientConfig) (*Client, error) {
	writer, err := utils.NewResultWriter(config.ResultsDir, getFiles(config), TypesAsNumbered())

	if err != nil {
		return nil, err
	}

	dataReader, err := reader.NewDataReader(config.dataFile)
	if err != nil {
		return nil, err
	}
	coordsReader, err := reader.NewCoordinatesReader(config.coordsFile)
	if err != nil {
		return nil, err
	}
	dataConn := protocol.NewProtocol(config.ServerData)
	if err := dataConn.Connect(); err != nil {
		return nil, err
	}
	resultsConn := protocol.NewProtocol(config.ServerResults)
	if err := resultsConn.Connect(); err != nil {
		return nil, err
	}
	return &Client{
		config:         config,
		writer:         writer,
		dataReader:     dataReader,
		coordsReader:   coordsReader,
		dataConn:       dataConn,
		resultsConn:    resultsConn,
		resultsEnd:     make(chan bool),
		dataSendingEnd: make(chan bool),
	}, nil
}

func getClientMultiData() protocol.Data {
	data := protocol.NewMultiData()
	data.Register(protocol.NewDataMessage(&distance.AirportDataType{}))
	return data
}

func (client *Client) runResults() {
	data := getClientMultiData()
	for {
		if err := client.resultsConn.Recover(data); err != nil {
			if err.Error() == errors.New("connection closed").Error() {
				break
			}
			logrus.Errorf("Error while recieving results: %s", err)
			break
		}
		client.writer.WriteInto(data.Type(), data.AsRecord())

	}
	client.resultsConn.Close()
	logrus.Info("Results listener exiting succesfuly")
	client.resultsEnd <- true
}

func (client *Client) runData() {
	for {
		data, err := client.coordsReader.ReadData()
		if err != nil {
			if err.Error() == io.EOF.Error() {
				logrus.Info("action: coordinate's data | result: success")
				break
			}
			logrus.Errorf("Error while reading coordinate data: %s", err)
			break
		}
		if err := client.dataConn.Send(data); err != nil {
			logrus.Errorf("error sending coordinates data: %s", err)
			break
		}
	}

	client.coordsReader.Close() //Try to get this error
	for {
		data, err := client.dataReader.ReadData()
		if err != nil {
			if err.Error() == io.EOF.Error() {
				logrus.Infof("action: data sending | result: success")
				break
			}
			logrus.Errorf("action: data sending | result: failed | error: %s", err)
			break
		}
		if err := client.dataConn.Send(data); err != nil {
			logrus.Errorf("action: data sending | result: failed | error: %s", err)
			break
		}
	}
	client.dataConn.Close()
	logrus.Infof("action: data sending | result: finished")
	client.dataSendingEnd <- true
}

func (client *Client) waiter() error {
	defer close(client.dataSendingEnd)
	defer close(client.resultsEnd)
	<-client.dataSendingEnd
	<-client.resultsEnd
	return nil
}

func (client *Client) Run() error {
	go client.runResults()
	go client.runData()
	return client.waiter()
}
