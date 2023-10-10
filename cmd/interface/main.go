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

	QUERY1EXCHANGE = "exchange.first"
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

	CONFIG_VARS = []string{
		QUERY1EXCHANGE,
		QUERY3EXCHANGE,
		QUERY2EXCHANGE,
		QUERY4EXCHANGE,
		QUERY1WORKERS,
		QUERY2WORKERS,
		QUERY3WORKERS,
		QUERY4WORKERS,
		LISTENINGPORT,
		RESULTSPORT,
		AGG_QUEUE,
	}
)

func getAggregator(v *viper.Viper, agg_context context.Context) (*lib.Agregator, error) {
	mid, err := middleware.Dial(v.GetString(SOURCE))
	if err != nil {
		return nil, err
	}

	name, err := mid.QueueDeclare(v.GetString(AGG_QUEUE))
	if err != nil {
		return nil, err
	}
	mid.ExchangeDeclare(v.GetString(AGG_QUEUE), "direct")
	mid.QueueBind(name, v.GetString(AGG_QUEUE), []string{"", "control"})
	mid.SetExpectedEOFCount(1)
	config := lib.AgregatorConfig{
		Mid:            mid,
		AgregatorQueue: v.GetString(AGG_QUEUE),
		Ctx:            agg_context,
	}

	return lib.NewAgregator(config), nil
}

func DeclareExchanges(mid *middleware.Middleware, ctx context.Context, v *viper.Viper) (err error) {
	_, err = mid.ExchangeDeclare(v.GetString(QUERY1EXCHANGE), "direct")
	if err == nil {
		_, err = mid.ExchangeDeclare(v.GetString(QUERY2EXCHANGE), "direct")
	}

	return
}

func getListener(v *viper.Viper, aggregator_chan chan<- *protocol.Protocol, list_context context.Context) (*lib.Parser, error) {
	mid, err := middleware.Dial(v.GetString(SOURCE))
	if err != nil {
		return nil, err
	}
	config := lib.ParserConfig{
		Query1:        v.GetString(QUERY1EXCHANGE),
		Workers1:      v.GetInt(QUERY1WORKERS),
		Query2:        v.GetString(QUERY2EXCHANGE),
		Workers2:      v.GetInt(QUERY2WORKERS),
		Query3:        v.GetString(QUERY3EXCHANGE),
		Workers3:      v.GetInt(QUERY3WORKERS),
		Query4:        v.GetString(QUERY4EXCHANGE),
		Workers4:      v.GetInt(QUERY4WORKERS),
		Mid:           mid,
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
	v, err := utils.InitConfig("ifz", "./cmd/interface/config/config.yaml")
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
