package main

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/distanceFilter/common"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

// Describes the topology around this node.
func setupMiddleware(ctx context.Context, m *mid.Middleware, v *viper.Viper) (string, string, string, error) {
	source, err := m.ExchangeDeclare(v.GetString("source"))
	if err != nil {
		return "", "", "", err
	}

	qCoords, err := m.QueueDeclare(v.GetString("queue.coords"))
	if err != nil {
		return "", "", "", err
	}
	// Subscribe to coordinates and EOF events.
	if err := m.QueueBind(qCoords, source, []string{"coords", mid.ControlRoutingKey + ".coords"}); err != nil {
		return "", "", "", err
	}

	qFlights, err := m.QueueDeclare(source)
	if err != nil {
		return "", "", "", err
	}
	// Subscribe to filter specific and EOF events.
	if err := m.QueueBind(qFlights, source, []string{source, mid.ControlRoutingKey}); err != nil {
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
	return qCoords, qFlights, sink, m.Ready(ctx, status)
}

func main() {
	v, err := utils.InitConfig("distance", "cmd/distanceFilter")
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

	coordsSource, flightsSource, sink, err := setupMiddleware(ctx, middleware, v)
	if err != nil {
		log.Fatal(err)
	}

	coordsQueues, err := middleware.Consume(ctx, coordsSource)
	if err != nil {
		log.Fatal(err)
	}
	flightsQueues, err := middleware.Consume(ctx, flightsSource)
	if err != nil {
		log.Fatal(err)
	}
	flightsChs := make(map[string]chan (<-chan []byte))
	var mtx sync.Mutex

	go func() {
		for flightsQueue := range flightsQueues {
			id := flightsQueue.Id
			mtx.Lock()
			ch, ok := flightsChs[id]
			if !ok {
				ch = make(chan (<-chan []byte), 1)
				flightsChs[id] = ch
			}
			mtx.Unlock()
			ch <- flightsQueue.Ch
		}
	}()

	for coordsQueue := range coordsQueues {
		go func(id string, ch <-chan []byte) {
			filter := common.NewFilter(middleware, id, sink)
			if err := filter.AddCoords(ctx, ch); err != nil {
				log.Error(err)
			}
			mtx.Lock()
			flightsCh, ok := flightsChs[id]
			if !ok {
				flightsCh = make(chan (<-chan []byte))
				flightsChs[id] = flightsCh
			}
			mtx.Unlock()
			select {
			case <-ctx.Done():
				return
			case ch = <-flightsCh:
				mtx.Lock()
				delete(flightsChs, id)
				mtx.Unlock()
			}
			if err := filter.Run(ctx, ch); err != nil {
				log.Error(err)
			} else if err := middleware.EOF(ctx, sink, id); err != nil {
				log.Error(err)
			}
		}(coordsQueue.Id, coordsQueue.Ch)
	}
}
