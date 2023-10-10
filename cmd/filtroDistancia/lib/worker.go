package lib

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/sirupsen/logrus"
	"github.com/umahmood/haversine"
)

type WorkerConfig struct {
	Mid    *middleware.Middleware
	Source string
	Sink   string
	Status string
	Times  int
	Ctx    context.Context
}

type Worker struct {
	config WorkerConfig

	computer     *distance.DistanceComputer
	finishedLoad bool
	finished     bool
}

func NewWorker(config WorkerConfig) (*Worker, error) {

	computer := distance.NewComputer()

	return &Worker{
		config:   config,
		computer: computer,
		finished: false,
	}, nil
}

func (worker *Worker) handleCoords(value middleware.CoordinatesData) {
	logrus.Infof("Adding airport: %s", value.AirportCode)
	coords := &haversine.Coord{
		Lat: value.Latitude,
		Lon: value.Longitud,
	}
	worker.computer.AddAirport(value.AirportCode, *coords)
}

func (worker *Worker) handleFilter(value middleware.DataQ2) {
	greaterThanX, err := worker.computer.GreaterThanXTimes(worker.config.Times, value)

	if err != nil {
		log.Printf("error processing data: %s", err)
		return
	}
	if greaterThanX {
		logrus.Infof("filtered flight: %s", string(value.ID[:]))
		worker.config.Mid.PublishWithContext(worker.config.Ctx, "",
			worker.config.Sink, middleware.Q2Marshal(value))
	}
}

func (worker *Worker) handleFinData() {

	logrus.Info("action: filtering | result: finished")
	worker.config.Mid.EOF(worker.config.Ctx, worker.config.Sink)
	worker.finished = true

}

func (worker *Worker) Start() error {
	runChan := make(chan error)
	sigChan := make(chan os.Signal, 1)
	defer close(runChan)
	defer close(sigChan)
	go func() {
		runChan <- worker.Run()
	}()

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-worker.config.Ctx.Done():
	case <-sigChan:
		worker.config.Mid.Close()
	case err := <-runChan:
		return err
	}
	return nil
}

func (worker *Worker) Shutdown() {
	worker.config.Mid.Close()
}

func (worker *Worker) handleData(buf []byte) error {

	data, err := middleware.DistanceFilterUnmarshal(buf)
	if err != nil {

		return err
	}

	switch data.(type) {
	case (middleware.CoordinatesData):
		coords := data.(middleware.CoordinatesData)
		worker.handleCoords(coords)
	case (middleware.DataQ2):
		airport := data.(middleware.DataQ2)
		worker.handleFilter(airport)
	}
	return nil
}

func (worker *Worker) Run() error {
	ch, err := worker.config.Mid.ConsumeWithContext(worker.config.Ctx, worker.config.Source)
	if err != nil {
		return err
	}
	worker.config.Mid.EOF(worker.config.Ctx, worker.config.Status)
	logrus.Info("distance filter worker up")
	for !worker.finished {
		select {
		case <-worker.config.Ctx.Done():
			return context.Cause(worker.config.Ctx)
		case buf, more := <-ch:
			if !more {
				worker.handleFinData()
				return nil
			}
			if err := worker.handleData(buf); err != nil {
				return err
			}
		}

	}

	return nil
}
