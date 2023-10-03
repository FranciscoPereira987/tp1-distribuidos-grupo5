package lib

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/conection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
)

type WorkerConfig struct {
	Addr string
}

type Worker struct {
	protocol   *protocol.Protocol
	connection conection.Conn

	computer     *distance.DistanceComputer
	finishedLoad bool
	unobtained   []distance.AirportDataType
}

func NewWorker(config WorkerConfig) (*Worker, error) {
	conection, err := conection.NewSocketConnection(config.Addr)

	if err != nil {
		return nil, err
	}

	protocol := protocol.NewProtocol(conection)

	computer := distance.NewComputer()

	return &Worker{
		protocol:     protocol,
		connection:   conection,
		computer:     computer,
		finishedLoad: false,
		unobtained:   nil,
	}, nil
}

func (worker *Worker) Shutdown() {
	worker.protocol.Close()
	worker.connection.Close()
}

func (worker *Worker) Run() error {
	coords := distance.IntoData(distance.Coordinates{})
	airport, _ := distance.NewAirportData("", "", "", 0)
	coordsEnd := &distance.CoordFin{}
	multi := protocol.NewMultiData()
	multi.Register(coords, protocol.NewDataMessage(airport), protocol.NewDataMessage(coordsEnd))
	for {
		if err := worker.protocol.Recover(multi); err != nil {
			return err
		}
		recovered := multi.Type()
		if value, ok := recovered.(*distance.CoordWrapper); ok {
			coords, _ := distance.CoordsFromData(multi)
			worker.computer.AddAirport(value.Name.Value(), *coords)
		} else if value, ok := recovered.(*distance.AirportDataType); ok {
			if worker.finishedLoad {
				//Calculate the query and send the result
			} else {
				worker.unobtained = append(worker.unobtained, *value)
			}
		} else if _, ok := recovered.(*distance.CoordFin); ok {
			worker.finishedLoad = true
		}
	}

	return nil
}
