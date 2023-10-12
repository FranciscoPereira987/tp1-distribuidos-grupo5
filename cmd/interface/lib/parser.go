package lib

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/sirupsen/logrus"
)

type ParserConfig struct {
	ResultsQueue string

	Query2   string
	Workers2 int
	Query3   string
	Workers3 int
	Query4   string
	Workers4 int

	WaitQueue    string
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
	query2Keys middleware.KeyGenerator
	query3Keys middleware.KeyGenerator
	query4Keys middleware.KeyGenerator

	dataConn *protocol.Protocol

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
		query2Keys: middleware.NewKeyGenerator(config.Workers2),
		query3Keys: middleware.NewKeyGenerator(config.Workers3),
		query4Keys: middleware.NewKeyGenerator(config.Workers4),
		processed:  0,
	}, nil
}

func (parser *Parser) publishQuery1(data *typing.FlightDataType) (err error) {
	flight := data.IntoResultQ1()
	err = parser.config.Mid.PublishWithContext(
		parser.config.Ctx,
		parser.config.ResultsQueue,
		parser.config.ResultsQueue,
		middleware.ResultQ1Marshal(flight),
	)
	return
}

func (parser *Parser) publishQuery2(data *typing.FlightDataType) (err error) {
	flight := data.IntoDistanceData()
	key := parser.query2Keys.KeyFrom(flight.Origin, flight.Destination)
	err = parser.config.Mid.PublishWithContext(parser.config.Ctx, parser.config.Query2, key, middleware.Q2Marshal(flight))
	return
}

func (parser *Parser) publishQuery3(data *typing.FlightDataType) (err error) {
	flight := data.IntoFastestFilterData()
	key := parser.query3Keys.KeyFrom(flight.Origin, flight.Destination)
	err = parser.config.Mid.PublishWithContext(parser.config.Ctx, parser.config.Query3, key, middleware.Q3Marshal(flight))
	return
}

func (parser *Parser) publishQuery4(data *typing.FlightDataType) (price float64, err error) {
	flight := data.IntoAvgFilterData()
	key := parser.query4Keys.KeyFrom(flight.Origin, flight.Destination)
	err = parser.config.Mid.PublishWithContext(parser.config.Ctx, parser.config.Query4, key, middleware.AvgMarshal(flight))
	return float64(flight.Price), err
}

func (parser *Parser) publishQuery4Avg(totalPrice float64, count int) (err error) {
	err = parser.config.Mid.PublishWithContext(
		parser.config.Ctx,
		parser.config.Query4,
		"avg",
		middleware.AvgPriceMarshal(totalPrice, count),
	)
	return err
}

func (parser *Parser) waitForWorkers() (wait chan error) {
	wait = make(chan error, 1)

	go func() {
		defer close(wait)
		logrus.Infof("action: waiting for %d workers at %s | result: in progress", parser.config.TotalWorkers-1, parser.config.WaitQueue)
		ch, err := parser.config.Mid.ConsumeWithContext(parser.config.Ctx, parser.config.WaitQueue)
		parser.config.Mid.SetExpectedControlCount(parser.config.TotalWorkers - 1)
		missing := parser.config.TotalWorkers
		for _, more := <-ch; more; _, more = <-ch {
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
		logrus.Info("action: shutting down | reason: recieved signal")
		err := parser.Shutdown(agg)
		logrus.Info("action: shutting down | result: success")
		return err
	case err := <-endResult:
		return err
	}

}

func (parser *Parser) Shutdown(agg *Agregator) (err error) {
	err = parser.listener.Close()
	parser.config.Mid.Close()
	agg.Shutdown()
	if parser.dataConn != nil {
		err = errors.Join(err, parser.dataConn.Close())
		err = errors.Join(err, parser.dataConn.Shutdown())
	}
	return
}

func (parser *Parser) Run(workers <-chan error) error {
	defer parser.config.Mid.Close()
	defer parser.listener.Close()
	logrus.Info("action: waiting connection | result: in progress")
	data, results, err := parser.listener.Accept()
	parser.dataConn = data
	defer data.Close()
	if err != nil {
		logrus.Errorf("action: waiting connection | result: failed | reason: %s", err)

		return err
	}
	if err := <-workers; err != nil {
		logrus.Infof("action: waiting for workers | result: failed | reason: %s", err)
		return err
	}
	logrus.Info("action: waiting for workers | results: success")
	parser.config.ResultsChan <- results

	message := getDataMessages()
	totalPrice, totalFlights := float64(0), 0
	for ; ; totalFlights++ {
		if err := data.Recover(message); err != nil {

			if err == protocol.ErrConnectionClosed {
				logrus.Info("client finished sending its data")
				err = parser.publishQuery4Avg(totalPrice, totalFlights)
				break
			}
			logrus.Errorf("action: recovering message | result: failed | reason: %s", err)
			break
		}
		messageType := message.Type()
		switch v := messageType.(type) {
		case (*distance.CoordWrapper):
			data := v.Value
			parser.config.Mid.PublishWithContext(parser.config.Ctx, parser.config.Query2, "coord", middleware.CoordMarshal(data))
		case (*typing.FlightDataType):
			data := v
			logrus.Info("sending data")
			err = parser.publishQuery2(data)
			if strings.Count(data.Stops, "||") >= 3 {
				err = errors.Join(err, parser.publishQuery1(data))
				err = errors.Join(err, parser.publishQuery3(data))
			}
			price, errQ4 := parser.publishQuery4(data)
			err = errors.Join(err, errQ4)
			totalPrice += price
			if err != nil {
				break
			}
		}
	}
	err = errors.Join(err, parser.config.Mid.Control(parser.config.Ctx, parser.config.ResultsQueue))
	err = errors.Join(err, parser.config.Mid.Control(parser.config.Ctx, parser.config.Query2))
	err = errors.Join(err, parser.config.Mid.Control(parser.config.Ctx, parser.config.Query3))
	err = errors.Join(err, parser.config.Mid.Control(parser.config.Ctx, parser.config.Query4))
	return err
}
