package main

import (
	"context"
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/distanceFilter/common"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

// Describes the topology around this node.
func setupMiddleware(ctx context.Context, m *mid.Middleware, v *viper.Viper) (string, string, string, error) {
	source, err := m.ExchangeDeclare(v.GetString("exchange.source"))
	if err != nil {
		return "", "", "", err
	}

	qCoords, err := m.QueueDeclare(v.GetString("queue.coords"))
	if err != nil {
		return "", "", "", err
	}
	// Subscribe to coordinates and EOF events.
	if err := m.QueueBind(qCoords, source, []string{"coords", mid.ControlRoutingKey+".coords"}); err != nil {
		return "", "", "", err
	}

	qFlights, err := m.QueueDeclare(v.GetString("queue.flights"))
	if err != nil {
		return "", "", "", err
	}
	// Subscribe to shards specific and EOF events.
	shardKey := mid.ShardKey(v.GetString("id"))
	if err := m.QueueBind(qFlights, source, []string{shardKey, mid.ControlRoutingKey}); err != nil {
		return "", "", "", err
	}
	m.SetExpectedControlCount(qFlights, v.GetInt("demuxers"))

	sink := v.GetString("exchange.sink")
	if sink == "" {
		return "", "", "", fmt.Errorf("%w: %q", utils.ErrMissingConfig, "exchange.sink")
	}

	status, err := m.QueueDeclare(v.GetString("status"))
	if err != nil {
		return "", "", "", err
	}
	if _, err := m.ExchangeDeclare(status); err != nil {
		return "", "", "", err
	}
	if err := m.QueueBind(status, status, []string{mid.ControlRoutingKey}); err != nil {
		return "", "", "", err
	}

	log.Info("distance filter worker up")
	return qCoords, qFlights, sink, m.Control(ctx, status)
}

func main() {
	v, err := utils.InitConfig("distance", "cmd/distanceFilter")
	if err != nil {
		log.Fatal(err)
	}
	if err := utils.InitLogger(v.GetString("log.level")); err != nil {
		log.Fatal(err)
	}
	if _, err := strconv.Atoi(v.GetString("id")); err != nil {
		log.Fatal(fmt.Errorf("Could not parse DISTANCE_ID env var as int: %w", err))
	}

	middleware, err := mid.Dial(v.GetString("server.url"))
	if err != nil {
		log.Fatal(err)
	}
	defer middleware.Close()

	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx := utils.WithSignal(parentCtx)

	coordsSource, flightsSource, sink, err := setupMiddleware(ctx, middleware, v)
	if err != nil {
		log.Fatal(err)
	}

	filter := common.NewFilter(middleware, coordsSource, flightsSource, sink)

	if err := filter.Run(ctx); err != nil {
		log.Error(err)
	} else if err := middleware.EOF(ctx, sink); err != nil {
		log.Error(err)
	}
}
