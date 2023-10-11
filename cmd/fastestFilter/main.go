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

	"github.com/franciscopereira987/tp1-distribuidos/cmd/fastestFilter/common"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

// Describes the topology around this node.
func setupMiddleware(m *mid.Middleware, v *viper.Viper) (string, string, error) {
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
	err = m.QueueBind(q, source, []string{shardKey, "control"})
	if err != nil {
		return "", "", err
	}

	sink, err := m.ExchangeDeclare(v.GetString("results"))
	return q, sink, err
}

func main() {
	v, err := utils.InitConfig("fast", "cmd/fastestFilter")
	if err != nil {
		log.Fatal(err)
	}
	if err := utils.InitLogger(v.GetString("log.level")); err != nil {
		log.Fatal(err)
	}
	if _, err := strconv.Atoi(v.GetString("id")); err != nil {
		log.Fatal(fmt.Errorf("Could not parse FAST_ID env var as int: %w", err))
	}

	middleware, err := mid.Dial(v.GetString("server.url"))
	if err != nil {
		log.Fatal(err)
	}
	defer middleware.Close()

	source, sink, err := setupMiddleware(middleware, v)
	if err != nil {
		log.Fatal(err)
	}

	filter := common.NewFilter(middleware, source, sink)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	go func() {
		<-sig
		cancel(fmt.Errorf("Signal received"))
	}()

	if err := filter.Run(ctx); err != nil {
		log.Error(err)
	} else if err := middleware.EOF(ctx, sink); err != nil {
		log.Error(err)
	}
}
