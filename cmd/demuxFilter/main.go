package main

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/demuxFilter/common"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

// Describes the topology around this node.
func setupMiddleware(ctx context.Context, m *mid.Middleware, v *viper.Viper) (string, []string, error) {
	source, err := m.ExchangeDeclare(v.GetString("source"))
	if err != nil {
		return "", nil, err
	}

	if _, err = m.QueueDeclare(source); err != nil {
		return "", nil, err
	}

	// Subscribe to filter specific and EOF events.
	if err := m.QueueBind(source, source, []string{source, mid.ControlRoutingKey}); err != nil {
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
	results, err := m.ExchangeDeclare(v.GetString("exchange.results"))
	if err != nil {
		return "", nil, err
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
	return source, []string{distance, fastest, average, results}, m.Control(ctx, status)
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

	ctx := utils.WithSignal(parentCtx)

	source, sinks, err := setupMiddleware(ctx, middleware, v)
	if err != nil {
		log.Fatal(err)
	}

	nWorkers := []int{
		v.GetInt("workers.q2"),
		v.GetInt("workers.q3"),
		v.GetInt("workers.q4"),
	}
	filter := common.NewFilter(middleware, source, sinks, nWorkers)

	if err := filter.Run(ctx); err != nil {
		log.Error(err)
	}

	// send EOF to sinks
	errs := make([]error, 0, len(sinks))
	for _, exchange := range sinks {
		errs = append(errs, middleware.EOF(ctx, exchange))
	}
	if err := errors.Join(errs...); err != nil {
		log.Error(err)
	}
}
