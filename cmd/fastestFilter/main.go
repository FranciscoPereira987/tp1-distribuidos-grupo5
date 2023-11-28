package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/fastestFilter/common"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/beater"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

// Describes the topology around this node.
func setupMiddleware(ctx context.Context, m *mid.Middleware, v *viper.Viper) (string, string, error) {
	eof, err := m.ExchangeDeclare(v.GetString("source.eof"))
	if err != nil {
		return "", "", err
	}

	q, err := mid.QueueName(v.GetString("source.queue"), v.GetString("id"))
	if err != nil {
		return "", "", err
	}
	if _, err = m.QueueDeclare(q); err != nil {
		return "", "", err
	}

	// Subscribe to EOF events.
	if err := m.QueueBind(q, eof); err != nil {
		return "", "", err
	}
	m.SetExpectedEofCount(q, v.GetInt("demuxers"))

	sink := v.GetString("sink.results")
	if sink == "" {
		return "", "", fmt.Errorf("%w: %q", utils.ErrMissingConfig, "sink.results")
	}

	status, err := m.QueueDeclare(v.GetString("status"))

	log.Info("fastest filter worker up")
	return q, sink, m.Ready(ctx, status)
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

	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCtx := utils.WithSignal(parentCtx)

	source, sink, err := setupMiddleware(signalCtx, middleware, v)
	if err != nil {
		log.Fatal(err)
	}
	workerId := source
	workdir := fmt.Sprintf("/clients/%d", v.GetInt("id"))
	recovered := state.RecoverStateFiles(workdir)
	toRestart := make(map[string]*common.Filter)
	for _, rec := range recovered {
		id, workdir, stateMan := rec.Id, rec.Workdir, rec.State
		filter := common.RecoverFromState(middleware, id, sink, workdir, stateMan)
		filter.Restart(signalCtx, toRestart)
	}
	queues, err := middleware.Consume(signalCtx, source)
	if err != nil {
		log.Fatal(err)
	}
	beaterClient := beater.StartBeaterClient(v)
	beaterClient.Run()
	for queue := range queues {
		if f, ok := toRestart[queue.Id]; ok {
			go func(f *common.Filter, ch <-chan mid.Delivery) {
				ctx, cancel := context.WithCancel(signalCtx)
				defer cancel()
				defer f.Close()
				if err := f.Run(ctx, ch); err != nil {
					logrus.Errorf("action: re-started | status: failed | reason: %s", err)
				}
			}(f, queue.Ch)
		} else {
			go func(id string, ch <-chan mid.Delivery) {
				ctx, cancel := context.WithCancel(signalCtx)
				defer cancel()

				workdir := filepath.Join("clients", hex.EncodeToString([]byte(id)))
				filter, err := common.NewFilter(middleware, id, sink, workdir)
				if err != nil {
					log.Error(err)
					return
				}
				defer filter.Close()

				if err := filter.Run(ctx, ch); err != nil {
					log.Error(err)
				} else if err := middleware.EOF(ctx, sink, workerId, id); err != nil {
					log.Error(err)
				}
			}(queue.Id, queue.Ch)
		}
	}
	beater.StopBeaterClient(beaterClient)
}
