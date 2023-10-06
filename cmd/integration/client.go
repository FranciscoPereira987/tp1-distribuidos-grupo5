package main

import (
	"net"

	client "github.com/franciscopereira987/tp1-distribuidos/cmd/cliente/lib"
	distanceFilter "github.com/franciscopereira987/tp1-distribuidos/cmd/filtroDistancia/lib"
	"github.com/franciscopereira987/tp1-distribuidos/cmd/interface/lib"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/conection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/dummies"
	"github.com/sirupsen/logrus"
)

/*
	This is a test command to test the system without having to run
	the docker files.
	It runs with a default configuration each of the instances
	and does not replicates them (it also does not use sockets)
*/

func getClientConfig(ports []string) client.ClientConfig {
	data, _ := conection.NewSocketConnection("127.0.0.1:" + ports[0])
	results, _ := conection.NewSocketConnection("127.0.0.1:" + ports[1])

	return client.ClientConfig{
		ResultsDir:    "./test/",
		Query2File:    "query_2.csv",
		DataFile:      "./data/test.csv",
		CoordsFile:    "./data/airports-codepublic.csv",
		ServerData:    data,
		ServerResults: results,
	}
}

func getDistanceConfig(ports []string) distanceFilter.WorkerConfig {
	data, _ := dummies.NewDummyProtocol("127.0.0.1:" + ports[0])
	result, _ := dummies.NewDummyProtocol("127.0.0.1:" + ports[1])
	return distanceFilter.WorkerConfig{
		DataConn:   data,
		ResultConn: result,
		Times:      4,
	}
}

func runInterface(parserPorts []string, listeningPorts []string) {
	channel := make(chan *protocol.Protocol)
	dataConn, _ := dummies.NewDummyProtocol("127.0.0.1:" + parserPorts[0])
	resultsConn, _ := dummies.NewDummyProtocol("127.0.0.1:" + parserPorts[1])
	dataProto := protocol.NewProtocol(dataConn)
	dataProto.Connect()
	resultsProto := protocol.NewProtocol(resultsConn)
	resultsProto.Connect()
	configParser := lib.ParserConfig{
		Query2:        dataProto,
		ListeningPort: listeningPorts[0],
		ResultsPort:   listeningPorts[1],

		ResultsChan: channel,
	}
	configAgg := lib.AgregatorConfig{
		AgregatorQueue: resultsProto,
	}
	parser, _ := lib.NewParser(configParser)
	aggregator := lib.NewAgregator(configAgg)
	go parser.Run()
	go aggregator.Run()
}

func makeMiddleMan(portsParser []string, portsWorker []string) {
	parserData, _ := net.Listen("tcp", "127.0.0.1:"+portsParser[0])
	parserResults, _ := net.Listen("tcp", "127.0.0.1:"+portsParser[1])
	workerData, _ := net.Listen("tcp", "127.0.0.1:"+portsWorker[0])
	workerResults, _ := net.Listen("tcp", "127.0.0.1:"+portsWorker[1])

	dataChan := make(chan byte)
	resultsChan := make(chan byte)

	go func() {
		conn, _ := parserData.Accept()
		bytes := make([]byte, 1)
		for {
			_, err := conn.Read(bytes)
			if err != nil {
				break
			}
			dataChan <- bytes[0]
		}
		conn.Close()
		parserData.Close()
	}()

	go func() {
		conn, _ := workerData.Accept()
		bytes := make([]byte, 1)

		for {
			value := <-dataChan
			bytes[0] = value
			if _, err := conn.Write(bytes); err != nil {
				break
			}
		}
		conn.Close()
		workerData.Close()
	}()
	go func() {
		conn, _ := workerResults.Accept()
		bytes := make([]byte, 1)
		for {
			_, err := conn.Read(bytes)
			if err != nil {
				break
			}
			resultsChan <- bytes[0]
		}
		conn.Close()
		workerResults.Close()
	}()
	go func() {
		conn, _ := parserResults.Accept()
		bytes := make([]byte, 1)
		for {
			value := <-resultsChan
			bytes[0] = value
			if _, err := conn.Write(bytes); err != nil {
				break
			}
		}
		conn.Close()
		parserResults.Close()
	}()

}

func main() {
	parserPorts := []string{"6666", "6667"}
	workerPorts := []string{"7777", "7778"}
	listeningPorts := []string{"10000", "10001"}
	makeMiddleMan(parserPorts, workerPorts)
	runInterface(parserPorts, listeningPorts)
	clientConfig := getClientConfig(listeningPorts)

	distanceConfig := getDistanceConfig(workerPorts)
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
