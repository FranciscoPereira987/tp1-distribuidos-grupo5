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
	eof, err := m.ExchangeDeclare(v.GetString("source.eof"))
	if err != nil {
		return "", "", err
	}
	average, err := m.ExchangeDeclare(v.GetString("source.average"))
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
	// Subscribe to average and EOF events.
	if err := m.QueueBind(q, eof, average); err != nil {
		return "", "", err
	}
	m.SetExpectedEofCount(q, v.GetInt("demuxers"))

	sink := v.GetString("sink.results")
	if sink == "" {
		return "", "", fmt.Errorf("%w: %q", utils.ErrMissingConfig, "sink.results")
	}

	status, err := m.QueueDeclare(v.GetString("status"))
	if err != nil {
		return "", "", err
	}

	log.Info("average filter worker up")
	return q, sink, m.Ready(ctx, status, q)
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
	workerId := source
	beaterClient := beater.StartBeaterClient(v)
	beaterClient.Run()
	toRestart := make(map[string]*common.Filter)
	workdir := fmt.Sprintf("/clients/%d", v.GetInt("id"))
	recovered := state.RecoverStateFiles(workdir)
	for _, rec := range recovered {
		id, workdir, stateMan := rec.Id, rec.Workdir, rec.State
		filter := common.RecoverFromState(middleware, workerId, id, sink, workdir, stateMan)
		if filter.ShouldRestart() {
			go func() {
				ctx, cancel := context.WithCancel(signalCtx)
				defer cancel()
				defer filter.Close()
				if err := filter.Restart(ctx); err != nil {
					log.Errorf("action: re-start filter sending results | status: failed | reason: %s", err)
				} else if err := middleware.EOF(ctx, sink, workerId, id); err != nil {
					log.Fatal(err)
				}
			}()
		} else {
			toRestart[id] = filter
		}
	}
	queues, err := middleware.Consume(signalCtx, source)
	if err != nil {
		log.Fatal(err)
	}
	for queue := range queues {
		f, ok := toRestart[queue.Id]
		if ok {
			delete(toRestart, queue.Id)
		} else {
			workdir := filepath.Join("clients", hex.EncodeToString([]byte(queue.Id)))
			f, err = common.NewFilter(middleware, workerId, queue.Id, sink, workdir)
			if err != nil {
				log.Fatal(err)
			}
		}
		go func(id string, ch <-chan mid.Delivery) {
			ctx, cancel := context.WithCancel(signalCtx)
			defer cancel()
			defer f.Close()
			if err := f.Run(ctx, ch); err != nil {
				log.Fatal(err)
			} else if err := middleware.EOF(ctx, sink, workerId, id); err != nil {
				log.Fatal(err)
			}
		}(queue.Id, queue.Ch)
	}
	beater.StopBeaterClient(beaterClient)
}
