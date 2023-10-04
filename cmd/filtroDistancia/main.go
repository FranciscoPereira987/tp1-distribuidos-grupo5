package main

import (
	"io"
	"log"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/filtroDistancia/lib"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/dummies"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/reader"
)

/*
Agregar como parametro, para que se pueda configurar las N veces mayor que la
directa que deberia tener el filtro.
*/
func runClient(sender *protocol.Protocol) {
	if err := sender.Accept(); err != nil {
		return
	}
	valueR, err := reader.NewCoordinatesReader("./data/airports-codepublic.csv")
	if err != nil {
		return
	}

	for {
		data, err := valueR.ReadData()
		if err != nil {
			break
		}
		sender.Send(data)
	}
	sender.Send(protocol.NewDataMessage(&distance.CoordFin{}))

	valueR.Close()
	valueR, err = reader.NewDataReader("./data/test.csv")
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
	sender.Send(protocol.NewDataMessage(&distance.CoordFin{}))
	sender.Close()
}
func main() {
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
	log.Printf("Results: %d", len(results))
}
