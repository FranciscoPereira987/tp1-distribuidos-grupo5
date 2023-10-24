package main

import (
	"context"
	"fmt"
	"strconv"

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
	sink := v.GetString("results")
	if sink == "" {
		return "", "", fmt.Errorf("%w: %s", utils.ErrMissingConfig, "results")
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

	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx := utils.WithSignal(parentCtx)

	source, sink, err := setupMiddleware(ctx, middleware, v)
	if err != nil {
		log.Fatal(err)
	}

	filter, err := common.NewFilter(middleware, source, sink, v.GetString("fares.dir"))
	if err != nil {
		log.Fatal(err)
	}

	if err := filter.Run(ctx); err != nil {
		log.Error(err)
	} else if err := middleware.Control(ctx, sink); err != nil {
		log.Error(err)
	}
}
