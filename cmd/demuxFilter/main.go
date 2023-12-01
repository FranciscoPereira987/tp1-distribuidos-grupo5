package main

import (
	"context"
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
func setupMiddleware(ctx context.Context, m *mid.Middleware, v *viper.Viper) (string, []string, string, error) {
	source, err := m.ExchangeDeclare(v.GetString("source.queue"))
	if err != nil {
		return "", nil, "", err
	}

	q, err := mid.QueueName(source, v.GetString("id"))
	if err != nil {
		return "", nil, "", err
	}
	if _, err = m.QueueDeclare(q); err != nil {
		return "", nil, "", err
	}

	// Subscribe to EOF events.
	if err := m.QueueBind(q, source); err != nil {
		return "", nil, "", err
	}

	distance := v.GetString("sink.distance")
	if distance == "" {
		return "", nil, "", fmt.Errorf("%w: %q", utils.ErrMissingConfig, "sink.distance")
	}
	fastest := v.GetString("sink.fastest")
	if fastest == "" {
		return "", nil, "", fmt.Errorf("%w: %q", utils.ErrMissingConfig, "sink.fastest")
	}
	average := v.GetString("sink.average")
	if average == "" {
		return "", nil, "", fmt.Errorf("%w: %q", utils.ErrMissingConfig, "sink.average")
	}
	results := v.GetString("sink.results")
	if results == "" {
		return "", nil, "", fmt.Errorf("%w: %q", utils.ErrMissingConfig, "sink.results")
	}
	eof := v.GetString("sink.eof")
	if eof == "" {
		return "", nil, "", fmt.Errorf("%w: %q", utils.ErrMissingConfig, "sink.eof")
	}

	status, err := m.QueueDeclare(v.GetString("status"))
	if err != nil {
		return "", nil, "", err
	}

	log.Info("demux filter worker up")
	return q, []string{distance, fastest, average, results}, eof, m.Ready(ctx, status, q)
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

	source, sinks, eof, err := setupMiddleware(signalCtx, middleware, v)
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
		filter := common.RecoverFromState(middleware, workerId, id, sinks, nWorkers, stateMan)
		filter.Restart(signalCtx, toRestart)
	}
	queues, err := middleware.Consume(signalCtx, source)
	if err != nil {
		log.Fatal(err)
	}
	beaterClient := beater.StartBeaterClient(v)
	beaterClient.Run()
	for queue := range queues {
		filter, ok := toRestart[queue.Id]
		if ok {
			delete(toRestart, queue.Id)
		} else {
			filter = common.NewFilter(middleware, workerId, queue.Id, sinks, nWorkers, workdir)
		}
		go func(clientId string, ch <-chan mid.Delivery) {
			ctx, cancel := context.WithCancel(signalCtx)
			defer cancel()

			if err := filter.Run(ctx, ch); err != nil {
				log.Fatal(err)
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			// send EOF to sinks
			if err := middleware.EOF(ctx, eof, workerId, clientId); err != nil {
				log.Fatal(err)
			}
		}(queue.Id, queue.Ch)
	}
	beater.StopBeaterClient(beaterClient)
}
