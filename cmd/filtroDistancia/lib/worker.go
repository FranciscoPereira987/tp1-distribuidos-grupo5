package lib

import (
	"log"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/conection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/sirupsen/logrus"
)

type WorkerConfig struct {
	DataConn   conection.Conn
	ResultConn conection.Conn
	Times      int
}

type Worker struct {
	config  WorkerConfig
	data    *protocol.Protocol
	results *protocol.Protocol

	computer     *distance.DistanceComputer
	finishedLoad bool
	finished     bool
}

func NewWorker(config WorkerConfig) (*Worker, error) {

	data := protocol.NewProtocol(config.DataConn)
	if err := data.Connect(); err != nil {
		return nil, err
	}

	results := protocol.NewProtocol(config.ResultConn)
	if err := results.Connect(); err != nil {
		return nil, err
	}

	computer := distance.NewComputer()

	return &Worker{
		config:       config,
		data:         data,
		results:      results,
		computer:     computer,
		finishedLoad: false,
		finished:     false,
	}, nil
}

func (worker *Worker) Shutdown() {
	worker.data.Close()
	worker.results.Close()
	worker.config.DataConn.Close()
	worker.config.ResultConn.Close()
}

func getMultiData() *protocol.MultiData {
	coords := distance.IntoData(distance.Coordinates{}, "")
	airport, _ := distance.NewAirportData("", "", "", 0)
	coordsEnd := &distance.CoordFin{}
	multi := protocol.NewMultiData()
	multi.Register(coords, protocol.NewDataMessage(airport), protocol.NewDataMessage(coordsEnd))
	return multi
}

func (worker *Worker) handleCoords(value *distance.CoordWrapper, data protocol.Data) {
	coords, _ := distance.CoordsFromData(data)
	logrus.Infof("Adding airport: %s", value.Name.Value())
	worker.computer.AddAirport(value.Name.Value(), *coords)
}

func (worker *Worker) handleFilter(value *distance.AirportDataType, data protocol.Data) {
	greaterThanX, err := value.GreaterThanXTimes(worker.config.Times, *worker.computer)
	if err != nil {
		log.Printf("error processing data: %s", err)
		return
	}
	if greaterThanX {
		logrus.Infof("filtered airport: %s", value.AsRecord())
		worker.results.Send(data)
	}
}

func (worker *Worker) handleFinData(value *distance.CoordFin, data protocol.Data) {
	if worker.finishedLoad {
		logrus.Info("finishing work")
		worker.results.Send(data)
		worker.finished = true
	} else {
		logrus.Info("action: coordinates send | result: finished")
		worker.finishedLoad = true
	}

}

func (worker *Worker) Run() error {
	multi := getMultiData()
	for !worker.finished {
		if err := worker.data.Recover(multi); err != nil {
			log.Printf("failed recovering data: %s", err)
			return err
		}
		recovered := multi.Type()
		if value, ok := recovered.(*distance.CoordWrapper); ok {
			worker.handleCoords(value, multi)
		} else if value, ok := recovered.(*distance.AirportDataType); ok {
			worker.handleFilter(value, multi)
		} else if value, ok := recovered.(*distance.CoordFin); ok {
			worker.handleFinData(value, multi)
		}
	}

	return nil
}
