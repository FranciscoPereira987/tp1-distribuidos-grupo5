package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/avgFilter/common"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

// Describes the topology around this node.
func setupMiddleware(ctx context.Context, m *mid.Middleware, v *viper.Viper) (string, string, error) {
	source, err := m.ExchangeDeclare(v.GetString("source"))
	if err != nil {
		return "", "", err
	}

	q, err := m.QueueDeclare(v.GetString("queue"))
	if err != nil {
		return "", "", err
	}

	// Subscribe to shards specific and EOF events.
	shardKey := mid.ShardKey(v.GetString("id"))
	if err := m.QueueBind(q, source, []string{shardKey, "avg", "control"}); err != nil {
		return "", "", err
	}
	sink, err := m.ExchangeDeclare(v.GetString("results"))
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
	if err := m.QueueBind(status, status, []string{"control"}); err != nil {
		return "", "", err
	}

	log.Info("average filter worker up")
	return q, sink, m.Control(ctx, status)
}

func main() {
	v, err := utils.InitConfig("avg", "cmd/avgFilter")
	if err != nil {
		log.Fatal(err)
	}
	if err := utils.InitLogger(v.GetString("log.level")); err != nil {
		log.Fatal(err)
	}
	if _, err := strconv.Atoi(v.GetString("id")); err != nil {
		log.Fatal(fmt.Errorf("Could not parse AVG_ID env var as int: %w", err))
	}

	middleware, err := mid.Dial(v.GetString("server.url"))
	if err != nil {
		log.Fatal(err)
	}
	defer middleware.Close()

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		cancel(fmt.Errorf("Signal received"))
	}()

	source, sink, err := setupMiddleware(ctx, middleware, v)
	if err != nil {
		log.Fatal(err)
	}

	filter := common.NewFilter(middleware, source, sink)

	if err := filter.Run(ctx); err != nil {
		log.Error(err)
	} else if err := middleware.Control(ctx, sink); err != nil {
		log.Error(err)
	}
}
