package lib

import (
	"net"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/reader"
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
	flightType, _ := reader.NewFlightDataType("", "", "", 0, 0, 0, "")
	flightData := protocol.NewDataMessage(flightType)
	endData := protocol.NewDataMessage(&reader.DataFin{})
	multi.Register(flightData, endData)
	return multi
}

func (l *Listener) Accept() (*protocol.Protocol, *protocol.Protocol, error) {
	dataConn, err := l.data.Accept()
	if err != nil {
		return nil, nil, err
	}
	resultsConn, err := l.results.Accept()
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
	logrus.Info("accepted succesfuly")

	resultsProt := protocol.NewProtocol(resultsConn)
	if err := resultsProt.Accept(); err != nil {
		dataConn.Close()
		resultsConn.Close()
		return nil, nil, err
	}

	return dataProt, resultsProt, nil
}
