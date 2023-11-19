package main

import (
	"context"
	"net"
	"sync"

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
	return source, m.Ready(ctx, status)
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

	signalCtx := utils.WithSignal(parentCtx)

	source, err := setupMiddleware(signalCtx, middleware, v)
	if err != nil {
		log.Fatal(err)
	}

	clients, err := connection.Listen(signalCtx, ":"+v.GetString("port"))
	if err != nil {
		log.Fatal(err)
	}

	queues, err := middleware.Consume(signalCtx, source)
	if err != nil {
		log.Fatal(err)
	}

	gateway := common.NewGateway(middleware)
	resultsChs := make(map[string]chan (<-chan mid.Delivery))
	var mtx sync.Mutex

	go func() {
		for queue := range queues {
			id := queue.Id
			mtx.Lock()
			ch, ok := resultsChs[id]
			if !ok {
				ch = make(chan (<-chan mid.Delivery), 1)
				resultsChs[id] = ch
			}
			mtx.Unlock()
			ch <- queue.Ch
		}
	}()

	for conn := range clients {
		go func(conn net.Conn) {
			ctx, cancel := context.WithCancel(signalCtx)
			defer cancel()

			defer conn.Close()
			id, err := connection.ReceiveId(conn)
			if err != nil {
				log.Error(err)
				return
			}
			mtx.Lock()
			resultsCh, ok := resultsChs[id]
			if !ok {
				resultsCh = make(chan (<-chan mid.Delivery))
				resultsChs[id] = resultsCh
			}
			mtx.Unlock()
			var ch <-chan mid.Delivery
			select {
			case <-ctx.Done():
				return
			case ch = <-resultsCh:
				mtx.Lock()
				delete(resultsChs, id)
				mtx.Unlock()
			}
			if err := gateway.Run(ctx, conn, ch); err != nil {
				log.Error(err)
			}
		}(conn)
	}

	select {
	case <-signalCtx.Done():
		log.Info(context.Cause(signalCtx))
	default:
	}
}
