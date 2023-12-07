package main

import (
	"context"
	"errors"
	"net"
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

	input := v.GetString("server.input")
	output := v.GetString("server.output")

	writer := common.NewDataWriter(coords, flights)
	reader, err := common.NewResultsReader(v.GetString("results.dir"), v.GetStringSlice("results.files"))
	if err != nil {
		log.Fatal(err)
	}

	idChan := make(chan string, 1)
	go func() {
		var (
			data net.Conn
			id string
		)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			conn, err := connection.Dial(ctx, input)
			if err != nil {
				log.Error(err)
				continue
			}

			id, err = connection.ConnectInput(conn)
			if err == nil {
				data = conn
				idChan <- id
				break
			}
			log.Error(err)
			conn.Close()
		}

		offset := int64(-1)
		for {
			err := writer.WriteData(data, offset)
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err == nil {
				err = connection.WaitDone(data)
			}
			switch {
			case err == nil:
				return
			case errors.Is(err, net.ErrClosed):
				log.Error(err)
			default:
				log.Fatal(err)
			}
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				conn, err := connection.Dial(ctx, input)
				if err != nil {
					log.Error(err)
					continue
				}

				offset, err = connection.ReconnectInput(conn, id)
				if err == nil {
					data = conn
					break
				}
				log.Error(err)
				conn.Close()
			}
		}
		data.Close()
	}()

	id := <- idChan
	var (
		results net.Conn
		progress int
	)

resultLoop:
	for {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			conn, err := connection.Dial(ctx, output)
			if err != nil {
				log.Error(err)
				continue
			}

			if err = connection.ConnectOutput(conn, id, progress); err == nil {
				results = conn
				break
			}
			log.Error(err)
			conn.Close()
		}
		progress, err = reader.ReadResults(results)
		select {
		case <-ctx.Done():
			return
		default:
		}
		switch {
		case err == nil:
			log.Infof("finished reading results: %d records", progress)
			break resultLoop
		case errors.Is(err, net.ErrClosed):
			log.Error(err)
		default:
			log.Fatal(err)
		}
	}

	if err := reader.Close(); err != nil {
		log.Error(err)
	}
}
