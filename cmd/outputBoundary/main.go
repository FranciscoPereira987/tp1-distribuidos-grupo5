package main

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/outputBoundary/common"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/connection"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

// Describes the topology around this node.
func setupMiddleware(ctx context.Context, m *mid.Middleware, v *viper.Viper) (string, error) {
	source, err := m.ExchangeDeclare(v.GetString("exchange.source"))
	if err != nil {
		return "", err
	}
	_, err = m.QueueDeclare(source)
	if err != nil {
		return "", err
	}
	if err := m.QueueBind(source, source, []string{source, mid.ControlRoutingKey}); err != nil {
		return "", err
	}
	m.SetExpectedControlCount(source, v.GetInt("workers"))

	status, err := m.QueueDeclare(v.GetString("status"))
	if err != nil {
		return "", err
	}
	if _, err := m.ExchangeDeclare(status); err != nil {
		return "", err
	}
	if err := m.QueueBind(status, status, []string{mid.ControlRoutingKey}); err != nil {
		return "", err
	}

	log.Info("output boundary up")
	return source, m.Control(ctx, status)
}

func main() {
	v, err := utils.InitConfig("out", "cmd/outputBoundary")
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

	source, err := setupMiddleware(ctx, middleware, v)
	if err != nil {
		log.Fatal(err)
	}

	clients, err := connection.Listen(ctx, ":"+v.GetString("port"))
	if err != nil {
		log.Fatal(err)
	}

	gateway := common.NewGateway(middleware, source)

	for conn := range clients {
		if err := gateway.Run(ctx, conn); err != nil {
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
