package main

import (
	client "github.com/franciscopereira987/tp1-distribuidos/cmd/cliente/lib"
	distanceFilter "github.com/franciscopereira987/tp1-distribuidos/cmd/filtroDistancia/lib"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/conection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/dummies"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/reader"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/sirupsen/logrus"
)

/*
	This is a test command to test the system without having to run
	the docker files.
	It runs with a default configuration each of the instances
	and does not replicates them (it also does not use sockets)
*/

func getClientConfig(data conection.Conn, results conection.Conn) client.ClientConfig {
	return client.ClientConfig{
		ResultsDir:    "./test/",
		Query2File:    "query_2.csv",
		DataFile:      "./data/test.csv",
		CoordsFile:    "./data/airports-codepublic.csv",
		ServerData:    data,
		ServerResults: results,
	}
}

func getDistanceConfig(data conection.Conn, result conection.Conn) distanceFilter.WorkerConfig {
	return distanceFilter.WorkerConfig{
		DataConn:   data,
		ResultConn: result,
		Times:      4,
	}
}

func getPipe() (conection.Conn, conection.Conn) {
	return dummies.NewDummyConnector(), dummies.NewDummyConnector()
}

func runMiddleMan(clientData conection.Conn, clientResult conection.Conn,
	distanceData conection.Conn, distanceResult conection.Conn) {
	clientForward := protocol.NewProtocol(clientData)
	if err := clientForward.Accept(); err != nil {
		logrus.Fatalf("failed accepting client: %s", err)
	}
	clientCallback := protocol.NewProtocol(clientResult)
	if err := clientCallback.Accept(); err != nil {
		logrus.Fatalf("failed accepting client: %s", err)
	}

	dataForward := protocol.NewProtocol(distanceData)
	if err := dataForward.Accept(); err != nil {
		logrus.Fatalf("failed accepting datafilter: %s", err)
	}
	dataCallback := protocol.NewProtocol(distanceResult)
	if err := dataCallback.Accept(); err != nil {
		logrus.Fatalf("failed accepting datafiltes: %s", err)
	}

	go func() {
		coordinates := protocol.NewDataMessage(&distance.CoordWrapper{
			Name: &typing.StrType{},
			Lat:  &typing.FloatType{},
			Long: &typing.FloatType{},
		})
		coordEnd := protocol.NewDataMessage(&distance.CoordFin{})
		multi := protocol.NewMultiData()
		data, _ := reader.NewFlightDataType(
			"", "", "", 1, 0, 0, "")
		multi.Register(coordinates, coordEnd, protocol.NewDataMessage(data))
		for {
			if err := clientForward.Recover(multi); err != nil {
				logrus.Infof("breaking off because of error: %s, at: %s", err, multi.AsRecord())
				if err.Error() == "connection closed" {
					dataForward.Send(coordEnd)
				}
				break
			}
			if value, ok := multi.Type().(*reader.FlightDataType); ok {
				dataForward.Send(protocol.NewDataMessage(value.IntoDistanceData()))
			} else {
				dataForward.Send(multi)
			}
		}
	}()

	airport, _ := distance.NewAirportData("", "", "", 0)

	result := protocol.NewDataMessage(airport)
	for {
		if err := dataCallback.Recover(result); err != nil {
			logrus.Errorf("Breaking off because of error: %s", err)
			clientCallback.Close()
			break
		}
		//logrus.Info("sending result to client")
		clientCallback.Send(result)
	}
}

func main() {
	clientData, clientResult := getPipe()
	distanceData, distanceResult := getPipe()

	clientConfig := getClientConfig(clientData, clientResult)
	distanceConfig := getDistanceConfig(distanceData, distanceResult)
	go runMiddleMan(clientData, clientResult, distanceData, distanceResult)
	client, err := client.NewClient(clientConfig)
	if err != nil {
		logrus.Fatalf("failed at client: %s", err)
	}

	distance, err := distanceFilter.NewWorker(distanceConfig)
	if err != nil {
		logrus.Fatalf("failed at distance filter: %s", err)
	}
	go distance.Run()

	client.Run()
}
