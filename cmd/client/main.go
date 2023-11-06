package main

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/client/common"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/connection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

func main() {
	v, err := utils.InitConfig("cli", "client/config")
	if err != nil {
		log.Fatal(err)
	}
	if err := utils.InitLogger(v.GetString("log.level")); err != nil {
		log.Fatal(err)
	}

	coords, err := os.Open(v.GetString("data.coords"))
	if err != nil {
		log.Fatal(err)
	}
	defer coords.Close()

	flights, err := os.Open(v.GetString("data.flights"))
	if err != nil {
		log.Fatal(err)
	}
	defer flights.Close()

	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx := utils.WithSignal(parentCtx)

	data, err := connection.Dial(ctx, v.GetString("server.input"))
	if err != nil {
		log.Fatal(err)
	}
	defer data.Close()

	results, err := connection.Dial(ctx, v.GetString("server.output"))
	if err != nil {
		log.Fatal(err)
	}
	defer results.Close()

	writer := common.NewDataWriter(coords, flights)
	reader, err := common.NewResultsReader(v.GetString("results.dir"), v.GetStringSlice("results.files"))
	if err != nil {
		log.Fatal(err)
	}

	id, err := connection.ConnectInput(data)
	if err != nil {
		log.Fatal(err)
	}

	if err := connection.ConnectOutput(results, id); err != nil {
		log.Fatal(err)
	}

	go func() {
		if err := writer.WriteData(data); err != nil {
			select {
			case <-ctx.Done():
			default:
				log.Error(err)
			}
		}
	}()

	if err := reader.ReadResults(results); err != nil {
		select {
		case <-ctx.Done():
		default:
			log.Error(err)
		}
	} else {
		log.Info("finished reading results")
	}
	if err := reader.Close(); err != nil {
		log.Error(err)
	}
}
