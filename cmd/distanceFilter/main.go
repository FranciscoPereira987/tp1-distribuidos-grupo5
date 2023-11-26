package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/franciscopereira987/tp1-distribuidos/cmd/distanceFilter/common"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/beater"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

// Describes the topology around this node.
func setupMiddleware(ctx context.Context, m *mid.Middleware, v *viper.Viper) (string, string, string, error) {
	source, err := m.ExchangeDeclare(v.GetString("source"))
	if err != nil {
		return "", "", "", err
	}

	qCoords := v.GetString("queue.coords")
	if qCoords == "" {
		qCoords = "coords." + v.GetString("id")
	}
	if _, err = m.QueueDeclare(qCoords); err != nil {
		return "", "", "", err
	}

	// Subscribe to coordinates and EOF events.
	if err := m.QueueBind(qCoords, source, []string{"coords", mid.ControlRoutingKey + ".coords"}); err != nil {
		return "", "", "", err
	}

	qFlights := v.GetString("queue.flights")
	if qFlights == "" {
		qFlights = "distance." + v.GetString("id")
	}
	if _, err := m.QueueDeclare(qFlights); err != nil {
		return "", "", "", err
	}
	// Subscribe to filter specific and EOF events.
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

	signalCtx := utils.WithSignal(parentCtx)

	coordsSource, flightsSource, sink, err := setupMiddleware(signalCtx, middleware, v)
	if err != nil {
		log.Fatal(err)
	}
	workdir := fmt.Sprintf("/clients/%d", v.GetInt("id"))
	files := state.RecoverStateFiles(workdir)
	for _, file := range files {
		filter := common.RecoverFromState(middleware, workdir, file)
		filter.Restart(signalCtx)
	}
	coordsQueues, err := middleware.Consume(signalCtx, coordsSource)
	if err != nil {
		log.Fatal(err)
	}
	flightsQueues, err := middleware.Consume(signalCtx, flightsSource)
	if err != nil {
		log.Fatal(err)
	}
	flightsChs := make(map[string]chan (<-chan mid.Delivery))
	var mtx sync.Mutex
	beaterClient := beater.StartBeaterClient(v)
	beaterClient.Run()
	go func() {
		for flightsQueue := range flightsQueues {
			id := flightsQueue.Id
			mtx.Lock()
			ch, ok := flightsChs[id]
			if !ok {
				ch = make(chan (<-chan mid.Delivery), 1)
				flightsChs[id] = ch
			}
			mtx.Unlock()
			ch <- flightsQueue.Ch
		}
	}()
	for coordsQueue := range coordsQueues {
		go func(id string, ch <-chan mid.Delivery) {
			ctx, cancel := context.WithCancel(signalCtx)
			defer cancel()
			workdir := filepath.Join(workdir, hex.EncodeToString([]byte(id)))
			filter, err := common.NewFilter(middleware, id, sink, workdir)
			if err != nil {
				log.Error(err)
				return
			}
			//defer filter.Close()

			if err := filter.AddCoords(ctx, ch); err != nil {
				log.Error(err)
				return
			}
			mtx.Lock()
			flightsCh, ok := flightsChs[id]
			if !ok {
				flightsCh = make(chan (<-chan mid.Delivery))
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
	beater.StopBeaterClient(beaterClient)
}
