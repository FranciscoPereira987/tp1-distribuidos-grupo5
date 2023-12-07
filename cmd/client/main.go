package main

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"syscall"

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

	var (
		data    *net.Conn
		results *net.Conn
	)

	go func(c1, c2 **net.Conn) {
		<-ctx.Done()
		if *c1 != nil {
			(**c1).Close()
		}
		if *c2 != nil {
			(**c2).Close()
		}
	}(&data, &results)

	idChan := make(chan string, 1)
	go func() {
		var (
			id     string
			offset int64 = -2
		)

		for {
			for timer := connection.NewBackoff(); ; timer.Backoff() {
				select {
				case <-ctx.Done():
					return
				case <-timer.Wait():
				}
				conn, err := connection.Dial(ctx, input)
				if err != nil {
					log.Error(err)
					continue
				}
				data = &conn

				if id == "" {
					id, err = connection.ConnectInput(conn)
					if err == nil {
						idChan <- id
						break
					}
				} else {
					offset, err = connection.ReconnectInput(conn, id)
					if err == nil {
						break
					}
				}
				log.Error(err)
				conn.Close()
			}
			err := writer.WriteData(*data, offset)
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err == nil {
				err = connection.WaitDone(*data)
			}
			switch {
			case err == nil:
				return
			case errors.Is(err, syscall.ECONNRESET):
			case errors.Is(err, syscall.EPIPE):
			default:
				log.Fatal("writing data", err)
			}
			log.Error(err)
		}
		(*data).Close()
	}()

	id := <-idChan
	progress := 0

resultLoop:
	for {
		for timer := connection.NewBackoff(); ; timer.Backoff() {
			select {
			case <-ctx.Done():
				return
			case <-timer.Wait():
			}
			conn, err := connection.Dial(ctx, output)
			if err != nil {
				log.Error(err)
				continue
			}
			results = &conn

			if err = connection.ConnectOutput(conn, id, progress); err == nil {
				break
			}
			log.Error(err)
			conn.Close()
		}
		recordsRead, err := reader.ReadResults(*results)
		progress += recordsRead
		select {
		case <-ctx.Done():
			return
		default:
		}
		switch {
		case err == nil:
			log.Infof("finished reading results: %d records", progress)
			break resultLoop
		case errors.Is(err, io.ErrUnexpectedEOF):
		default:
			log.Fatal("reading results", err)
		}
		log.Error(err)
	}

	if err := reader.Close(); err != nil {
		log.Error(err)
	}
}
