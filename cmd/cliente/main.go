package main

import (
	"github.com/franciscopereira987/tp1-distribuidos/cmd/cliente/lib"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/conection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	RESULTS_DIR = "results.dir"
	//QUERY1 = "results.first"
	QUERY2 = "results.second"
	//QUERY3 = "results.third"
	//QUERY4 = "results.fourth"
	DATA_FILE   = "datafile"
	COORDS_FILE = "coordsfile"

	SERVER_ADDR  = "server.addr"
	DATA_PORT    = "server.data"
	RESULTS_PORT = "server.results"

	CONFIG_VARS = []string{
		RESULTS_DIR,
		QUERY2,
		DATA_FILE,
		COORDS_FILE,
		SERVER_ADDR,
		DATA_PORT,
		RESULTS_PORT,
	}
)

func connecTo(addr string, port string) conection.Conn {
	conn, err := conection.Dial(addr,  port)
	if err != nil {
		logrus.Fatalf("error connecting: %s", err)
	}
	return conn
}

func getConfig(v *viper.Viper) *lib.ClientConfig {
	config := &lib.ClientConfig{}
	config.Query2File = v.GetString(QUERY2)
	config.ResultsDir = v.GetString(RESULTS_DIR)
	config.ServerData = connecTo(v.GetString(SERVER_ADDR), v.GetString(DATA_PORT))
	config.ServerResults = connecTo(v.GetString(SERVER_ADDR), v.GetString(RESULTS_PORT))
	config.DataFile = v.GetString(DATA_FILE)
	config.CoordsFile = v.GetString(COORDS_FILE)
	return config
}

func main() {
	if err := utils.DefaultLogger(); err != nil {
		logrus.Errorf("error initializing logger: %s", err)
	}
	v, err := utils.InitConfig("cli", "./cmd/cliente/config/config.yaml")
	if err != nil {
		logrus.Fatalf("error initializing config: %s", err)
	}
	utils.PrintConfig(v, CONFIG_VARS...)
	config := getConfig(v)
	client, err := lib.NewClient(*config)
	if err != nil {
		logrus.Fatalf("error initializing client: %s", err)
	}
	client.Run()
	config.ServerData.Close()
	config.ServerResults.Close()
	logrus.Info("action: exiting client | result: success")
}
