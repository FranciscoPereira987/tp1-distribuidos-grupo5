package main

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/inputBoundary/common"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/connection"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

// Describes the topology around this node.
func setupMiddleware(ctx context.Context, m *mid.Middleware, v *viper.Viper) (string, string, error) {
	coords, err := m.ExchangeDeclare(v.GetString("exchange.coords"))
	if err != nil {
		return "", "", err
	}
	flights, err := m.ExchangeDeclare(v.GetString("exchange.flights"))
	if err != nil {
		return "", "", err
	}

	status, err := m.QueueDeclare(v.GetString("status"))
	if err != nil {
		return "", "", err
	}
	if _, err := m.ExchangeDeclare(status); err != nil {
		return "", "", err
	}
	if err := m.QueueBind(status, status, []string{mid.ControlRoutingKey}); err != nil {
		return "", "", err
	}
	m.SetExpectedControlCount(status, v.GetInt("workers"))

	ch, err := m.ConsumeWithContext(ctx, status)
	if err != nil {
		return "", "", err
	}

	log.Info("input boundary up")

	<-ch
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

	ctx := utils.WithSignal(parentCtx)

	coords, flights, err := setupMiddleware(ctx, middleware, v)
	if err != nil {
		log.Fatal(err)
	}

	clients, err := connection.Listen(ctx, ":"+v.GetString("port"))
	if err != nil {
		log.Fatal(err)
	}

	gateway := common.NewGateway(middleware, coords, flights)
	demuxers := v.GetInt("demuxers")

	for conn := range clients {
		if err := gateway.Run(ctx, conn, demuxers); err != nil {
			log.Error(err)
		}
		conn.Close()
	}

	select {
	case <-ctx.Done():
		log.Info(context.Cause(ctx))
	default:
	}
}
