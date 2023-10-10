package main

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/filtroDistancia/lib"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/connection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	TIMES       = "times"
	DATA_ADDR   = "source.data"
	RESULT_ADDR = "source.sink"
	ID          = "id"
	SOURCE      = "source.url"
	STATUS      = "source.status"
	CONFIG_VARS = []string{
		TIMES,
		DATA_ADDR,
		RESULT_ADDR,
		ID,
		SOURCE,
		STATUS,
	}
)

/*
Agregar como parametro, para que se pueda configurar las N veces mayor que la
directa que deberia tener el filtro.
*/

func connecTo(addr string, port string) *connection.Conn {
	conn, err := connection.Dial(addr, port)
	if err != nil {
		panic(err.Error())
	}
	return conn
}

func getConfig(v *viper.Viper) (config lib.WorkerConfig, cancel context.CancelFunc) {
	config.Times = v.GetInt(TIMES)
	ctx, cancel := context.WithCancel(context.Background())
	config.Ctx = ctx
	config.Status = v.GetString(STATUS)
	return
}

func setupMiddleware(mid *middleware.Middleware, v *viper.Viper) (data string, sink string, err error) {
	data = v.GetString("source.data")
	sink = v.GetString("source.sink")
	id := v.GetString("id")
	name, err := mid.QueueDeclare("")
	if err != nil {
		return
	}
	_, err = mid.QueueDeclare(v.GetString(STATUS))
	if err != nil {
		return
	}
	shardKey := []string{id, "control", "coord"}
	mid.ExchangeDeclare(data)
	err = mid.QueueBind(name, data, shardKey)
	if err != nil {
		return
	}
	data = name
	return
}

func main() {
	utils.DefaultLogger()
	v, err := utils.InitConfig("CLI", "./cmd/filtroDistancia/config/config.yaml")
	if err != nil {
		log.Fatalf("could not initialize config: %s", err)
	}
	utils.PrintConfig(v, CONFIG_VARS...)

	config, cancel := getConfig(v)
	mid, err := middleware.Dial(v.GetString("source.url"))
	if err != nil {
		log.Fatalf("error dialing middleware: %s", err)
	}
	data, sink, err := setupMiddleware(mid, v)
	config.Mid = mid
	config.Source = data
	config.Sink = sink
	defer cancel()
	worker, err := lib.NewWorker(config)
	if err != nil {
		log.Fatalf("error at creating worker: %s", err)
	}
	if err := worker.Start(); err != nil {
		log.Fatalf("error during run: %s", err)
	}
	worker.Shutdown()
	utils.PrintConfig(v, CONFIG_VARS...)
}
