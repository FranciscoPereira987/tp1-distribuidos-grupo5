package lib

import (
	"errors"
	"net"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/connection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/sirupsen/logrus"
)

type Listener struct {
	data    net.Listener
	results net.Listener
}

func NewListener(dataAt string, resultsAt string) (*Listener, error) {
	data, err := net.Listen("tcp", "0.0.0.0:"+dataAt)
	if err != nil {
		return nil, err
	}
	results, err := net.Listen("tcp", "0.0.0.0:"+resultsAt)
	if err != nil {
		return nil, err
	}
	return &Listener{
		data:    data,
		results: results,
	}, nil

}

func getDataMessages() protocol.Data {
	multi := protocol.NewMultiData()
	flightType := typing.NewData(0, "")
	flightData := protocol.NewDataMessage(flightType)
	endData := protocol.NewDataMessage(typing.FinData())
	multi.Register(flightData, endData)
	return multi
}

func (l *Listener) Close() error {
	return errors.Join(l.data.Close(), l.results.Close())

}

func (l *Listener) Accept() (*protocol.Protocol, *protocol.Protocol, error) {
	dataConn, err := connection.FromListener(l.data)
	if err != nil {
		return nil, nil, err
	}
	resultsConn, err := connection.FromListener(l.results)
	if err != nil {
		dataConn.Close()
		return nil, nil, err
	}
	dataProt := protocol.NewProtocol(dataConn)
	if err := dataProt.Accept(); err != nil {
		dataConn.Close()
		resultsConn.Close()
		return nil, nil, err
	}
	logrus.Info("Accepted both connections")

	resultsProt := protocol.NewProtocol(resultsConn)
	if err := resultsProt.Accept(); err != nil {
		dataConn.Close()
		resultsConn.Close()
		return nil, nil, err
	}

	logrus.Info("accepted succesfuly")
	return dataProt, resultsProt, nil
}
