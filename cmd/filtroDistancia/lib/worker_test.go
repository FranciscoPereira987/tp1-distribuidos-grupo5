package lib_test

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/filtroDistancia/lib"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/dummies"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/reader"
)

func runClient(sender *protocol.Protocol) {
	if err := sender.Accept(); err != nil {
		return
	}
	valueR, err := reader.NewCoordinatesReader("/home/francisco/Documents/Distribuidos/tp1/data/airports-codepublic.csv")
	if err != nil {
		wd, _ := os.Getwd()
		panic(fmt.Sprintf("Error hier: %s, %s", wd, err))
	}

	for {
		data, err := valueR.ReadData()
		if err != nil {
			break
		}
		sender.Send(data)
	}

	sender.Send(protocol.NewDataMessage(reader.FinData()))
	valueR.Close()
	valueR, err = reader.NewDataReader("/home/francisco/Documents/Distribuidos/tp1/data/test.csv")
	if err != nil {
		return
	}
	sent := 0
	for {
		data, err := valueR.ReadData()
		if err != nil {
			log.Printf("error: %s", err)
			if err.Error() == io.EOF.Error() {
				break
			}
			continue
		}
		sent++
		data = protocol.NewDataMessage(data.Type().(*reader.FlightDataType).IntoDistanceData())
		sender.Send(data)
	}
	log.Printf("Sent: %d flights", sent)
	sender.Send(protocol.NewDataMessage(reader.FinData()))
	sender.Close()
}
func getResults() map[string]bool {
	file, _ := os.Open("/home/francisco/Documents/Distribuidos/tp1/data/test_results.csv")
	reader := csv.NewReader(file)
	reader.Read()
	results := make(map[string]bool)
	for {
		line, err := reader.Read()
		if err != nil {
			break
		}
		results[line[1]] = true
	}
	return results
}

func TestProcessingWithOneWorker(t *testing.T) {
	sender := dummies.NewDummyConnector()
	reciever := dummies.NewDummyConnector()

	clientConnector := protocol.NewProtocol(sender)
	go runClient(clientConnector)

	recieverConnector := protocol.NewProtocol(reciever)
	go recieverConnector.Accept()
	worker, err := lib.NewWorker(lib.WorkerConfig{
		DataConn:   sender,
		ResultConn: reciever,
		Times:      4,
	})
	if err != nil {
		log.Fatalf("could not create worker")
	}
	
	go worker.Run()

	results := make([]distance.AirportDataType, 0)
	dataType, err := distance.NewAirportData("", "", "", 0)
	if err != nil {
		log.Fatalf("Failed at creating data: %s", err)
	}
	data := protocol.NewDataMessage(dataType)
	for recieverConnector.Recover(data) == nil {

		results = append(results, *data.Type().(*distance.AirportDataType))
	}
	
	expected := getResults()
	if len(expected) != len(results) {
		t.Fatalf("results are not as much as expected: %d vs %d", len(expected), len(results))
	}
	for _, result := range results {
		if _, ok := expected[result.Type()[0]]; !ok {
			t.Fatalf("result: %s not found in expected", result.Type()[0])
		}
	}
}
