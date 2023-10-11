package main

import (
	"context"
	"log"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/interface/lib"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	SOURCE = "source"

	QUERY2EXCHANGE = "exchange.second"
	QUERY3EXCHANGE = "exchange.third"
	QUERY4EXCHANGE = "exchange.fourth"

	QUERY1WORKERS = "workers.first"
	QUERY2WORKERS = "workers.second"
	QUERY3WORKERS = "workers.third"
	QUERY4WORKERS = "workers.fourth"

	LISTENINGPORT = "server.dataport"
	RESULTSPORT   = "server.resultsport"

	AGG_QUEUE = "exchange.agregator"
	WAITQUEUE = "exchange.status"

	CONFIG_VARS = []string{
		QUERY3EXCHANGE,
		QUERY2EXCHANGE,
		QUERY4EXCHANGE,
		QUERY2WORKERS,
		QUERY3WORKERS,
		QUERY4WORKERS,
		LISTENINGPORT,
		RESULTSPORT,
		AGG_QUEUE,
	}

	WORKER_VARS = []string{
		QUERY1WORKERS,
		QUERY2WORKERS,
		QUERY3WORKERS,
		QUERY4WORKERS,
	}

	EXCHANGE_VARS = []string{
		QUERY2EXCHANGE,
		QUERY3EXCHANGE,
		QUERY4EXCHANGE,
	}
)

func getTotalWorkers(v *viper.Viper) (total int) {
	for _, value := range WORKER_VARS {
		total += v.GetInt(value)
	}
	return
}

func getAggregator(v *viper.Viper, agg_context context.Context) (*lib.Agregator, error) {
	mid, err := middleware.Dial(v.GetString(SOURCE))
	if err != nil {
		return nil, err
	}

	name, err := mid.QueueDeclare(v.GetString(AGG_QUEUE))
	if err != nil {
		return nil, err
	}
	mid.ExchangeDeclare(v.GetString(AGG_QUEUE))
	mid.QueueBind(name, v.GetString(AGG_QUEUE), []string{"", "control", v.GetString(AGG_QUEUE)})
	mid.SetExpectedControlCount(getTotalWorkers(v))
	config := lib.AgregatorConfig{
		Mid:            mid,
		AgregatorQueue: v.GetString(AGG_QUEUE),
		Ctx:            agg_context,
	}

	return lib.NewAgregator(config), nil
}

func DeclareExchanges(mid *middleware.Middleware, ctx context.Context, v *viper.Viper) (err error) {
	for _, name := range EXCHANGE_VARS {
		eName, err := mid.ExchangeDeclare(v.GetString(name))
		logrus.Infof("action: exchange declaration | info: declared: %s", eName)
		if err != nil {
			return err
		}
	}

	return
}

func getListener(v *viper.Viper, aggregator_chan chan<- *protocol.Protocol, list_context context.Context) (*lib.Parser, error) {
	mid, err := middleware.Dial(v.GetString(SOURCE))
	if err != nil {
		return nil, err
	}
	name, err := mid.QueueDeclare(v.GetString(WAITQUEUE))
	if err != nil {
		return nil, err
	}
	mid.ExchangeDeclare(name)
	mid.QueueBind(name, name, []string{"control"})
	config := lib.ParserConfig{
		ResultsQueue:  v.GetString(AGG_QUEUE),
		Query2:        v.GetString(QUERY2EXCHANGE),
		Workers2:      v.GetInt(QUERY2WORKERS),
		Query3:        v.GetString(QUERY3EXCHANGE),
		Workers3:      v.GetInt(QUERY3WORKERS),
		Query4:        v.GetString(QUERY4EXCHANGE),
		Workers4:      v.GetInt(QUERY4WORKERS),
		Mid:           mid,
		WaitQueue:     v.GetString(WAITQUEUE),
		TotalWorkers:  getTotalWorkers(v),
		Ctx:           list_context,
		ListeningPort: v.GetString(LISTENINGPORT),
		ResultsPort:   v.GetString(RESULTSPORT),
		ResultsChan:   aggregator_chan,
	}
	DeclareExchanges(mid, list_context, v)
	return lib.NewParser(config)
}

func main() {
	err := utils.DefaultLogger()
	if err != nil {
		logrus.Fatalf("could not initialize logger: %s", err)
	}
	v, err := utils.InitConfig("ifz", "config")
	if err != nil {
		logrus.Fatalf("could not initialize config: %s", err)
	}
	utils.PrintConfig(v, CONFIG_VARS...)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	agg, err := getAggregator(v, ctx)
	if err != nil {
		logrus.Fatalf("error creating agregator: %s", err)
	}
	aggC := agg.GetChan()

	parser, err := getListener(v, aggC, ctx)

	if err != nil {
		logrus.Fatalf("error creating parser: %s", err)
	}
	if err := parser.Start(agg); err != nil {
		log.Fatalf("error during run: %s", err)
	}
}
