package lib

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/sirupsen/logrus"
)

type ParserConfig struct {
	Query1   string
	Workers1 int
	Query2   string
	Workers2 int
	Query3   string
	Workers3 int
	Query4   string
	Workers4 int

	WaitQueue string
	TotalWorkers int

	Mid *middleware.Middleware

	ListeningPort string
	ResultsPort   string
	Ctx           context.Context

	ResultsChan chan<- *protocol.Protocol
}

/*
Parser is in charge of retrieving client data and distribute it
to the diferent workers
*/
type Parser struct {
	config     ParserConfig
	listener   *Listener
	query1Keys middleware.KeyGenerator
	query2Keys middleware.KeyGenerator
	query3Keys middleware.KeyGenerator
	query4Keys middleware.KeyGenerator

	processed uint32
}

func NewParser(config ParserConfig) (*Parser, error) {
	listener, err := NewListener(config.ListeningPort, config.ResultsPort)
	if err != nil {
		return nil, err
	}
	return &Parser{
		config:     config,
		listener:   listener,
		query1Keys: middleware.NewKeyGenerator(config.Workers1),
		query2Keys: middleware.NewKeyGenerator(config.Workers2),
		query3Keys: middleware.NewKeyGenerator(config.Workers3),
		query4Keys: middleware.NewKeyGenerator(config.Workers4),
		processed:  0,
	}, nil
}

func (parser *Parser) publishQuery1(data *typing.FlightDataType) (err error) {
	flight := data.IntoStopsFilterData()
	key := parser.query1Keys.KeyFrom(flight.Origin, flight.Destination)
	err = parser.config.Mid.PublishWithContext(parser.config.Ctx, parser.config.Query1, key, middleware.Q1Marshal(flight))
	return
}

func (parser *Parser) publishQuery2(data *typing.FlightDataType) (err error) {
	flight := data.IntoDistanceData()
	key := parser.query2Keys.KeyFrom(flight.Origin, flight.Destination)
	err = parser.config.Mid.PublishWithContext(parser.config.Ctx, parser.config.Query2, key, middleware.Q2Marshal(flight))
	return
}

func (parser *Parser) publishQuery3(data *typing.FlightDataType) (err error) {
	flight := data.IntoStopsFilterData()
	key := parser.query1Keys.KeyFrom(flight.Origin, flight.Destination)
	err = parser.config.Mid.PublishWithContext(parser.config.Ctx, parser.config.Query3, key, middleware.Q1Marshal(flight))

	return
}

func (parser *Parser) publishQuery4(data *typing.FlightDataType) (err error) {
	flight := data.IntoAvgFilterData()
	key := parser.query4Keys.KeyFrom(flight.Origin, flight.Destination)
	err = parser.config.Mid.PublishWithContext(parser.config.Ctx, parser.config.Query4, key, middleware.AvgMarshal(flight))
	return
}

func (parser *Parser) waitForWorkers() (wait chan error) {
	wait = make(chan error, 1)
	
	go func ()  {
		defer close(wait)
		logrus.Infof("action: waiting for %d workers at %s | result: in progress", parser.config.TotalWorkers, parser.config.WaitQueue)
		ch, err := parser.config.Mid.ConsumeWithContext(parser.config.Ctx, parser.config.WaitQueue)
		parser.config.Mid.SetExpectedEOFCount(parser.config.TotalWorkers)
		missing := parser.config.TotalWorkers
		for _, more := <- ch ;more; _, more = <- ch {
			missing--
			logrus.Infof("action: waiting for workers | info: missing: %d", missing)
		}
		logrus.Infof("action: waiting for %d workers | result: finished", parser.config.TotalWorkers)
		wait <- err
	}()

	return
}

func (parser *Parser) Start(agg *Agregator) error {
	sig := make(chan os.Signal, 1)
	result := make(chan error, 1)
	aggResult := make(chan error, 1)

	endResult := make(chan error, 1)

	defer close(sig)
	defer close(result)
	defer close(aggResult)
	defer close(endResult)

	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	workers := parser.waitForWorkers()
	go func() {
		result <- parser.Run(workers)
	}()

	go func() {
		aggResult <- agg.Run()
	}()

	go func() {
		parserResult := <-result
		agregatorResult := <-aggResult

		endResult <- errors.Join(parserResult, agregatorResult)
	}()
	
	

	select {
	case <-parser.config.Ctx.Done():
		return context.Cause(parser.config.Ctx)
	case <-sig:
		return nil
	case err := <-endResult:
		return err
	}

}

func (parser *Parser) Run(workers <-chan error) error {
	
	logrus.Info("action: waiting connection | result: in progress")
	data, results, err := parser.listener.Accept()
	if err != nil {
		logrus.Errorf("action: waiting connection | result: failed | reason: %s", err)

		return err
	}
	if err := <- workers; err != nil {
		logrus.Infof("action: waiting for workers | result: failed | reason: %s", err)
		return err
	}
	logrus.Info("action: waiting for workers | results: success")
	parser.config.ResultsChan <- results
	
	message := getDataMessages()
	for {
		if err := data.Recover(message); err != nil {
		
			if err.Error() == protocol.ErrConnectionClosed.Error() {
				logrus.Info("client finished sending its data")
				err = parser.config.Mid.EOF(parser.config.Ctx, parser.config.Query1)
				err = errors.Join(err, parser.config.Mid.EOF(parser.config.Ctx, parser.config.Query2))
				err = errors.Join(err, parser.config.Mid.EOF(parser.config.Ctx, parser.config.Query3))
				err = errors.Join(err, parser.config.Mid.EOF(parser.config.Ctx, parser.config.Query4))
				break
			}
			logrus.Errorf("action: recovering message | result: failed | reason: %s", err)
			continue
		}
		messageType := message.Type()
		switch v := messageType.(type) {
		case (*distance.CoordWrapper):
			data := v.Value
			parser.config.Mid.PublishWithContext(parser.config.Ctx, parser.config.Query2, "coord", middleware.CoordMarshal(data))
		case (*typing.FlightDataType):
			data := v
			//err = parser.publishQuery1(data)
			err = errors.Join(err, parser.publishQuery2(data))
			//err = errors.Join(err, parser.publishQuery3(data))
			//err = errors.Join(err, parser.publishQuery4(data))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
