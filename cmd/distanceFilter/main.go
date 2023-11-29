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
	coords, err := m.ExchangeDeclare(v.GetString("source.coords.exchange"))
	if err != nil {
		return "", "", "", err
	}

	qCoords, err := mid.QueueName(v.GetString("source.coords.queue"), v.GetString("id"))
	if err != nil {
		return "", "", "", err
	}
	if _, err = m.QueueDeclare(qCoords); err != nil {
		return "", "", "", err
	}

	// Subscribe to coordinates and EOF events.
	if err := m.QueueBind(qCoords, coords); err != nil {
		return "", "", "", err
	}

	eof, err := m.ExchangeDeclare(v.GetString("source.flights.eof"))
	if err != nil {
		return "", "", "", err
	}

	qFlights, err := mid.QueueName(v.GetString("source.flights.queue"), v.GetString("id"))
	if err != nil {
		return "", "", "", err
	}
	if _, err = m.QueueDeclare(qFlights); err != nil {
		return "", "", "", err
	}

	// Subscribe to EOF events.
	if err := m.QueueBind(qFlights, eof); err != nil {
		return "", "", "", err
	}
	m.SetExpectedEofCount(qFlights, v.GetInt("demuxers"))

	sink := v.GetString("sink.results")
	if sink == "" {
		return "", "", "", fmt.Errorf("%w: %q", utils.ErrMissingConfig, "sink.results")
	}

	status, err := m.QueueDeclare(v.GetString("status"))
	if err != nil {
		return "", "", "", err
	}

	log.Info("distance filter worker up")
	return qCoords, qFlights, sink, m.Ready(ctx, status, qCoords)
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
	workerId := coordsSource
	workdir := fmt.Sprintf("/clients/%d", v.GetInt("id"))
	flightsChs := make(map[string]chan (<-chan mid.Delivery))
	var mtx sync.Mutex
	recovered := state.RecoverStateFiles(workdir)
	for _, rec := range recovered {
		id, workdir, stateMan := rec.Id, rec.Workdir, rec.State
		if onFlights, _ := stateMan.Get("coordinates-load").(bool); !onFlights {
			continue
		}
		ch := make(chan (<-chan mid.Delivery))
		flightsChs[id] = ch

		go func() {
			ctx, cancel := context.WithCancel(signalCtx)
			defer cancel()
			filter, _ := common.NewFilter(middleware, id, sink, workdir)
			defer filter.Close()
			select {
			case <-ctx.Done():
				return
			case delivery := <-ch:
				mtx.Lock()
				delete(flightsChs, id)
				mtx.Unlock()
				log.Info("action: restart_worker | status: on_flights")
				if err := filter.Run(ctx, delivery); err != nil {
					log.Errorf("action: restarted worker | status: failed | reason: %s", err)
				} else if err := middleware.EOF(ctx, sink, workerId, id); err != nil {
					log.Fatal(err)
				}
			}
		}()
	}
	coordsQueues, err := middleware.Consume(signalCtx, coordsSource)
	if err != nil {
		log.Fatal(err)
	}
	flightsQueues, err := middleware.Consume(signalCtx, flightsSource)
	if err != nil {
		log.Fatal(err)
	}
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
				log.Fatal(err)
				return
			}
			defer filter.Close()

			if err := filter.AddCoords(ctx, ch); err != nil {
				log.Fatal(err)
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
				log.Fatal(err)
			} else if err := middleware.EOF(ctx, sink, workerId, id); err != nil {
				log.Fatal(err)
			}
		}(coordsQueue.Id, coordsQueue.Ch)
	}
	beater.StopBeaterClient(beaterClient)
}
