package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/filtroDistancia/lib"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/conection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	TIMES       = "times"
	DATA_ADDR   = "data.addr"
	DATA_PORT   = "data.port"
	RESULT_ADDR = "result.addr"
	RESULT_PORT = "result.port"

	CONFIG_VARS = []string{
		TIMES,
		DATA_ADDR,
		DATA_PORT,
		RESULT_ADDR,
		RESULT_PORT,
	}
)

/*
Agregar como parametro, para que se pueda configurar las N veces mayor que la
directa que deberia tener el filtro.
*/

func connecTo(addr string, port string) conection.Conn {
	conn, err := conection.NewSocketConnection(addr + ":" + port)
	if err != nil {
		panic(err.Error())
	}
	return conn
}

func getConfig(v *viper.Viper) (config lib.WorkerConfig) {
	config.Times = v.GetInt(TIMES)
	config.DataConn = connecTo(v.GetString(DATA_ADDR), v.GetString(DATA_PORT))
	config.ResultConn = connecTo(v.GetString(RESULT_ADDR), v.GetString(RESULT_PORT))
	return
}

func main() {
	utils.DefaultLogger()
	v, err := utils.InitConfig("./cmd/filtroDistancia/config/config.yaml", CONFIG_VARS...)
	if err != nil {
		log.Fatalf("could not initialize config: %s", err)
	}
	utils.PrintConfig(v, CONFIG_VARS...)

	config := getConfig(v)
	worker, err := lib.NewWorker(config)
	if err != nil {
		log.Fatalf("error at creating worker: %s", err)
	}
	if err := worker.Run(); err != nil {
		log.Fatalf("error during run: %s", err)
	}
	worker.Shutdown()
}
