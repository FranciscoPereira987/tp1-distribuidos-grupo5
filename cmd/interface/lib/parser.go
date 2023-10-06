package lib

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/reader"
	"github.com/sirupsen/logrus"
)

type ParserConfig struct {
	//Query1 *protocol.Protocol
	Query2 *protocol.Protocol
	//Query3 *protocol.Protocol
	//Query4 *protocol.Protocol

	ListeningPort string
	ResultsPort   string

	ResultsChan chan<- *protocol.Protocol
}

/*
Parser is in charge of retrieving client data and distribute it
to the diferent workers
*/
type Parser struct {
	config   ParserConfig
	listener *Listener
	processed uint32
}

func NewParser(config ParserConfig) (*Parser, error) {
	listener, err := NewListener(config.ListeningPort, config.ResultsPort)
	if err != nil {
		return nil, err
	}
	return &Parser{
		config:   config,
		listener: listener,
		processed: 0,
	}, nil
}

func (parser *Parser) Run() error {
	logrus.Info("action: waiting conection | result: in progress")
	data, results, err := parser.listener.Accept()
	if err != nil {
		logrus.Errorf("action: waiting conection | result: failed | reason: %s", err)
		data.Shutdown()
		results.Shutdown()
		return err
	}
	
	parser.config.ResultsChan <- results
	
	message := getDataMessages()
	for {

		if err := data.Recover(message); err != nil {
			
			if err.Error() == "connection closed" {
				logrus.Info("client finished sending its data")
				parser.config.Query2.Close()
				break
			}
			continue
		}
		if value, ok := message.Type().(*reader.FlightDataType); ok {
			parser.processed++
			parser.config.Query2.Send(protocol.NewDataMessage(value.IntoDistanceData()))
		} else {
			parser.config.Query2.Send(message)
		}
	}
	return nil
}
