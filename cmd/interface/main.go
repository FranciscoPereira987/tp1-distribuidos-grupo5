package main

import (
	"github.com/franciscopereira987/tp1-distribuidos/cmd/interface/lib"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/conection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	//QUERY1PORT = "query.first"
	QUERY2PORT = "query.second"
	//QUERY3PORT = "query.third"
	//QUERY4PORT = "query.fourth"

	LISTENINGPORT = "server.dataport"
	RESULTSPORT   = "server.resultsport"

	AGG_QUEUE = "agregator"

	CONFIG_VARS = []string{
		QUERY2PORT,
		LISTENINGPORT,
		RESULTSPORT,
		AGG_QUEUE,
	}
)

func getAggregator(v *viper.Viper) (*lib.Agregator, error) {
	conn, err := conection.NewSocketConnection("127.0.0.1:" + v.GetString(AGG_QUEUE))
	if err != nil {
		return nil, err
	}
	proto := protocol.NewProtocol(conn)

	if err := proto.Connect(); err != nil {
		return nil, err
	}

	config := lib.AgregatorConfig{
		AgregatorQueue: proto,
	}

	return lib.NewAgregator(config), nil
}

func getListener(v *viper.Viper, aggregator_chan chan<- *protocol.Protocol) (*lib.Parser, error) {
	conn, err := conection.NewSocketConnection("127.0.0.1:" + v.GetString(QUERY2PORT))
	if err != nil {
		return nil, err
	}
	proto := protocol.NewProtocol(conn)

	if err := proto.Connect(); err != nil {
		return nil, err
	}

	config := lib.ParserConfig{
		Query2:        proto,
		ListeningPort: v.GetString(LISTENINGPORT),
		ResultsPort:   v.GetString(RESULTSPORT),
		ResultsChan:   aggregator_chan,
	}
	return lib.NewParser(config)
}

func main() {
	err := utils.DefaultLogger()
	if err != nil {
		logrus.Fatalf("could not initialize logger: %s", err)
	}
	v, err := utils.InitConfig("./cmd/interface/config/config.yaml", CONFIG_VARS...)
	if err != nil {
		logrus.Fatalf("could not initialize config: %s", err)
	}
	utils.PrintConfig(v, CONFIG_VARS...)
	agg, err := getAggregator(v)
	if err != nil {
		logrus.Fatalf("error creating agregator: %s", err)
	}
	parser, err := getListener(v, agg.GetChan())
	if err != nil {
		logrus.Fatalf("error creating parser: %s", err)
	}
	go parser.Run()
	agg.Run()
}
