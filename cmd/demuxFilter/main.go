package main

import (
	"context"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/demuxFilter/common"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/beater"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

// Describes the topology around this node.
func setupMiddleware(ctx context.Context, m *mid.Middleware, v *viper.Viper) (string, []string, error) {
	source, err := m.ExchangeDeclare(v.GetString("source"))
	if err != nil {
		return "", nil, err
	}

	q := v.GetString("queue")
	if q == "" {
		q = source + "." + v.GetString("id")
	}
	if _, err = m.QueueDeclare(q); err != nil {
		return "", nil, err
	}

	// Subscribe to filter specific and EOF events.
	shardKey := mid.ShardKey(v.GetString("id"))
	if err := m.QueueBind(q, source, []string{shardKey, mid.ControlRoutingKey}); err != nil {
		return "", nil, err
	}
	distance, err := m.ExchangeDeclare(v.GetString("exchange.distance"))
	if err != nil {
		return "", nil, err
	}
	fastest, err := m.ExchangeDeclare(v.GetString("exchange.fastest"))
	if err != nil {
		return "", nil, err
	}
	average, err := m.ExchangeDeclare(v.GetString("exchange.average"))
	if err != nil {
		return "", nil, err
	}
	results := v.GetString("exchange.results")
	if results == "" {
		return "", nil, fmt.Errorf("%w: %q", utils.ErrMissingConfig, "exchange.results")
	}

	status, err := m.QueueDeclare(v.GetString("status"))
	if err != nil {
		return "", nil, err
	}
	if _, err := m.ExchangeDeclare(status); err != nil {
		return "", nil, err
	}
	if err := m.QueueBind(status, status, []string{mid.ControlRoutingKey}); err != nil {
		return "", nil, err
	}

	log.Info("demux filter worker up")
	return q, []string{distance, fastest, average, results}, m.Ready(ctx, status)
}

func main() {
	v, err := utils.InitConfig("demux", "cmd/demuxFilter")
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

	source, sinks, err := setupMiddleware(signalCtx, middleware, v)
	if err != nil {
		log.Fatal(err)
	}
	workerId := source

	nWorkers := []int{
		v.GetInt("workers.q2"),
		v.GetInt("workers.q3"),
		v.GetInt("workers.q4"),
	}
	workdir := fmt.Sprintf("/clients/%d", v.GetInt("id"))
	toRestart := make(map[string]*common.Filter)
	recovered := state.RecoverStateFiles(workdir)
	for _, rec := range recovered {
		id, _, stateMan := rec.Id, rec.Workdir, rec.State
		filter := common.RecoverFromState(middleware, id, sinks, stateMan)
		filter.Restart(signalCtx, toRestart)
	}
	queues, err := middleware.Consume(signalCtx, source)
	if err != nil {
		log.Error(err)
	}
	beaterClient := beater.StartBeaterClient(v)
	beaterClient.Run()
	for queue := range queues {
		if f, ok := toRestart[queue.Id]; ok {
			delete(toRestart, queue.Id)
			go func(id string, ch <-chan mid.Delivery, f *common.Filter) {
				ctx, cancel := context.WithCancel(signalCtx)
				defer cancel()
				f.Run(ctx, ch)
			}(queue.Id, queue.Ch, f)
		} else {
			go func(id string, ch <-chan mid.Delivery) {
				ctx, cancel := context.WithCancel(signalCtx)
				defer cancel()

				filter := common.NewFilter(middleware, id, sinks, nWorkers, workdir)

				if err := filter.Run(ctx, ch); err != nil {
					log.Error(err)
				}

				select {
				case <-ctx.Done():
					return
				default:
				}

				// send EOF to sinks
				errs := make([]error, 0, len(sinks))
				for _, exchange := range sinks {
					errs = append(errs, middleware.EOF(ctx, exchange, workerId, id))
				}
				if err := errors.Join(errs...); err != nil {
					log.Error(err)
				}
			}(queue.Id, queue.Ch)
		}
	}
	beater.StopBeaterClient(beaterClient)
}
