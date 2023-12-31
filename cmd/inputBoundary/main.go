package main

import (
	"context"
	"encoding/hex"
	"errors"
	"net"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/inputBoundary/common"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/beater"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/connection"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

// Describes the topology around this node.
func setupMiddleware(ctx context.Context, m *mid.Middleware, v *viper.Viper) (string, string, error) {
	coords, err := m.ExchangeDeclare(v.GetString("sink.coords"))
	if err != nil {
		return "", "", err
	}
	flights, err := m.ExchangeDeclare(v.GetString("sink.flights"))
	if err != nil {
		return "", "", err
	}

	status, err := m.QueueDeclare(v.GetString("status"))
	if err != nil {
		return "", "", err
	}

	log.Info("input boundary up")
	// wait for workers + 1*(outputBoundary)
	if err := m.WaitReady(ctx, status, v.GetInt("workers")+1); err != nil {
		return "", "", err
	}
	log.Info("all workers are ready")

	return coords, flights, nil
}

func main() {
	v, err := utils.InitConfig("in", "cmd/inputBoundary")
	if err != nil {
		log.Fatal(err)
	}
	if err := utils.InitLogger(v.GetString("log.level")); err != nil {
		log.Fatal(err)
	}

	middleware, err := mid.Dial(v.GetString("server.url"))
	if err != nil {
		log.Fatal(err)
	}
	defer middleware.Close()

	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCtx := utils.WithSignal(parentCtx)

	coords, flights, err := setupMiddleware(signalCtx, middleware, v)
	if err != nil {
		log.Fatal(err)
	}

	clients, err := connection.Listen(signalCtx, ":"+v.GetString("port"))
	if err != nil {
		log.Fatal(err)
	}

	demuxers := v.GetInt("demuxers")
	beaterClient := beater.StartBeaterClient(v)
	beaterClient.Run()
	for conn := range clients {
		go func(conn net.Conn) {
			ctx, cancel := context.WithCancel(signalCtx)
			defer cancel()

			defer conn.Close()
			reconnecting, id, err := connection.Accept(conn)
			if err != nil {
				log.Error(err)
				return
			}
			workdir := filepath.Join("clients", hex.EncodeToString([]byte(id)))
			sm := state.NewStateManager(workdir)
			if reconnecting {
				sm.RecoverState()
				offset, err := sm.GetInt64("offset")
				switch {
				case err == nil:
				case errors.Is(err, state.ErrNotFound):
					offset = -2
				default:
					log.Fatal(err)
				}
				if err := connection.SendState(conn, offset); err != nil {
					log.Error(err)
					return
				}
			}
			gateway, err := common.NewGateway(middleware, id, coords, flights, workdir, sm)
			if err != nil {
				log.Fatal(err)
			}
			defer gateway.Close()
			if err := gateway.Run(ctx, conn, demuxers); err != nil {
				log.Fatal(err)
			}
			connection.Done(conn)
		}(conn)
	}

	select {
	case <-signalCtx.Done():
		log.Info(context.Cause(signalCtx))
	default:
	}
	beater.StopBeaterClient(beaterClient)
}
