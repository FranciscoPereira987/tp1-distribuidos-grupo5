package common

import (
	"bytes"
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

const distanceFactor = 4

type Filter struct {
	m       *mid.Middleware
	coords  string
	flights string
	sink    string
}

func NewFilter(m *mid.Middleware, coords, flights, sink string) *Filter {
	return &Filter{
		m:       m,
		coords:  coords,
		flights: flights,
		sink:    sink,
	}
}

func (f *Filter) Run(ctx context.Context) error {
	distanceComputer := distance.NewComputer()
	ch, err := f.m.ConsumeWithContext(ctx, f.coords)
	if err != nil {
		return err
	}

	log.Infof("start consuming coordinates from %q queue", f.coords)
	for msg := range ch {
		data, err := typing.AirportCoordsUnmarshal(msg)
		if err != nil {
			return err
		}

		distanceComputer.AddAirportCoords(data.Code, data.Lat, data.Lon)
		log.Debugf("got coordinates for airport %s", data.Code)
	}
	log.Infof("finished consuming coordinates from %q queue", f.coords)

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
	}

	ch, err = f.m.ConsumeWithContext(ctx, f.flights)
	if err != nil {
		return err
	}

	log.Infof("start consuming flights from %q queue", f.flights)
	for msg := range ch {
		data, err := typing.DistanceFilterUnmarshal(msg)
		if err != nil {
			return err
		}

		log.Debugf("new flight for route %s-%s", data.Origin, data.Destination)
		distanceMi, err := distanceComputer.Distance(data.Origin, data.Destination)
		if err != nil {
			return err
		}
		if float64(data.Distance) > distanceFactor*distanceMi {
			var b bytes.Buffer
			log.Debugf("long flight: %x", data.ID)
			typing.ResultQ2Marshal(&b, msg)
			if err := f.m.PublishWithContext(ctx, f.sink, f.sink, b.Bytes()); err != nil {
				return err
			}
		}
	}
	log.Infof("finished consuming flights from %q queue", f.flights)

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
		return nil
	}
}
