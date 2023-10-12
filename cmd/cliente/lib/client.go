package lib

import (
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/connection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/reader"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
	"github.com/sirupsen/logrus"
)

type ClientConfig struct {
	ResultsDir string
	Query1File string
	Query2File string
	Query3File string
	Query4File string

	DataFile   string
	CoordsFile string

	ServerData    *connection.Conn
	ServerResults *connection.Conn
}

type Client struct {
	config       ClientConfig
	writer       *utils.ResultWriter
	coordsReader reader.Reader
	dataReader   reader.Reader
	dataConn     *protocol.Protocol
	resultsConn  *protocol.Protocol


	notifyClose	   chan bool
	resultsEnd     chan bool
	dataSendingEnd chan bool
}

func getFiles(config ClientConfig) []string {
	files := []string{config.Query1File, config.Query2File, config.Query3File, config.Query4File}
	return files
}

func getResultTypes() []typing.Type {
	types := []typing.Type{typing.NewResultQ1(), typing.NewResultQ2(), typing.NewResultQ3(), typing.NewResultQ4()}
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
	dataReader, err := reader.NewDataReader(config.DataFile)
	if err != nil {
		return nil, err
	}
	coordsReader, err := reader.NewCoordinatesReader(config.CoordsFile)
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
		notifyClose: make(chan bool),
		resultsEnd:     make(chan bool),
		dataSendingEnd: make(chan bool),
	}, nil
}

func getClientMultiData() protocol.Data {
	data := protocol.NewMultiData()
	types := getResultTypes()
	for _, value := range types {
		data.Register(protocol.NewDataMessage(value))
	}
	return data
}

func (client *Client) runResults() {
	data := getClientMultiData()
	for {
		select {
		case <- client.notifyClose:
			return
		default:
		}
		if err := client.resultsConn.Recover(data); err != nil {
			if err == protocol.ErrConnectionClosed {
				break
			}
			logrus.Errorf("Error while recieving results: %s", err)
			break
		}
		logrus.Infof("client recovered result: %s", data.AsRecord())
		if err := client.writer.WriteInto(data.Type(), data.AsRecord()); err != nil {
			logrus.Errorf("Error writting to file: %s", err)
		}

	}
	client.resultsConn.Close()
	logrus.Info("Results listener exiting succesfuly")
	client.resultsEnd <- true
}

func (client *Client) runData() {
	for {
		select {
		case <- client.notifyClose:
			return
		default:
		}
		data, err := client.coordsReader.ReadData()

		if err != nil {
			if err == io.EOF {
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

	client.coordsReader.Close()
	for {
		select {
		case <- client.notifyClose:
			return
		default:
		}
		data, err := client.dataReader.ReadData()

		if err != nil {
			if err == io.EOF {
				logrus.Infof("action: data sending | result: success")
				break
			}
			logrus.Errorf("action: data sending | result: failed | error: %s", err)
			continue
		}
		if err := client.dataConn.Send(data); err != nil {
			logrus.Errorf("action: data sending | result: failed | error: %s", err)
			break
		}
	}
	client.dataConn.Send(protocol.NewDataMessage(reader.FinData()))
	client.dataConn.Close()
	logrus.Infof("action: data sending | result: finished")
	client.dataSendingEnd <- true
}

func (client *Client) finished() chan bool {
	finished := make(chan bool)
	
	go func() {
		data := <- client.dataSendingEnd
		result := <- client.resultsEnd
		finished <- result || data
	}()
	
	return finished
}

func (client *Client) waiter() error {
	defer close(client.dataSendingEnd)
	defer close(client.resultsEnd)
	defer func() {
		client.dataConn.Close()
		client.dataConn.Shutdown()
	}()
	defer func() {
		client.resultsConn.Close()
		client.resultsConn.Shutdown()
	}()
	defer client.coordsReader.Close()
	defer client.dataReader.Close()
	defer client.writer.Close()
	finished := client.finished()
	sig := make(chan os.Signal, 1)
	defer close(sig)
	defer close(finished)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	
	select {
	case <- finished:
	case <- sig:
		logrus.Info("action: shutting_down | result: recieved signal")
		client.notifyClose <- true
		client.notifyClose <- true
	}
	return nil
}

func (client *Client) Run() error {
	defer client.writer.Close()

	go client.runResults()
	go client.runData()
	return client.waiter()
}
