package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/avgFilter/common"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/beater"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

// Describes the topology around this node.
func setupMiddleware(ctx context.Context, m *mid.Middleware, v *viper.Viper) (string, string, error) {
	source, err := m.ExchangeDeclare(v.GetString("exchange.source"))
	if err != nil {
		return "", "", err
	}

	q := v.GetString("queue")
	if q == "" {
		q = source + "." + v.GetString("id")
	}
	if _, err = m.QueueDeclare(q); err != nil {
		return "", "", err
	}
	// Subscribe to shards specific and EOF events.
	shardKey := mid.ShardKey(v.GetString("id"))
	if err := m.QueueBind(q, source, []string{shardKey, "average", mid.ControlRoutingKey}); err != nil {
		return "", "", err
	}
	m.SetExpectedControlCount(q, v.GetInt("demuxers"))

	sink := v.GetString("exchange.sink")
	if sink == "" {
		return "", "", fmt.Errorf("%w: %q", utils.ErrMissingConfig, "exchange.sink")
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

	log.Info("average filter worker up")
	return q, sink, m.Ready(ctx, status)
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

	signalCtx := utils.WithSignal(parentCtx)

	source, sink, err := setupMiddleware(signalCtx, middleware, v)
	if err != nil {
		log.Fatal(err)
	}
	workdir := fmt.Sprintf("/clients/%d", v.GetInt("id"))
	recovered := state.RecoverStateFiles(workdir)
	for _, rec := range recovered {
		id, workdir, stateMan := rec.Id, rec.Workdir, rec.State
		filter := common.RecoverFromState(middleware, id, sink, workdir, stateMan)
		filter.Restart(signalCtx)
	}
	queues, err := middleware.Consume(signalCtx, source)
	if err != nil {
		log.Fatal(err)
	}
	beaterClient := beater.StartBeaterClient(v)
	beaterClient.Run()
	for queue := range queues {
		go func(id string, ch <-chan mid.Delivery) {
			ctx, cancel := context.WithCancel(signalCtx)
			defer cancel()

			workdir := filepath.Join("clients", hex.EncodeToString([]byte(id)))
			filter, err := common.NewFilter(middleware, id, sink, workdir)
			if err != nil {
				log.Fatal(err)
			}
			defer filter.Close()

			if err := filter.Run(ctx, ch); err != nil {
				log.Error(err)
			} else if err := middleware.EOF(ctx, sink, id); err != nil {
				log.Error(err)
			}
		}(queue.Id, queue.Ch)
	}
	beater.StopBeaterClient(beaterClient)
}
