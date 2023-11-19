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
	m    *mid.Middleware
	id   string
	sink string
	comp *distance.DistanceComputer
}

func NewFilter(m *mid.Middleware, id, sink string) *Filter {
	return &Filter{
		m:    m,
		id:   id,
		sink: sink,
		comp: distance.NewComputer(),
	}
}

func (f *Filter) AddCoords(ctx context.Context, coords <-chan mid.Delivery) error {
	for d := range coords {
		msg, tag := d.Msg, d.Tag
		for r := bytes.NewReader(msg); r.Len() > 0; {
			data, err := typing.AirportCoordsUnmarshal(r)
			if err != nil {
				return err
			}

			f.comp.AddAirportCoords(data.Code, data.Lat, data.Lon)
			log.Debugf("got coordinates for airport %s", data.Code)
		}
		// TODO: store state
		if err := f.m.Ack(tag); err != nil {
			return err
		}
	}

	return context.Cause(ctx)
}

func (f *Filter) Run(ctx context.Context, flights <-chan mid.Delivery) error {
	var bc mid.BasicConfirmer

	for d := range flights {
		msg, tag := d.Msg, d.Tag
		b := bytes.NewBufferString(f.id)
		for r := bytes.NewReader(msg); r.Len() > 0; {
			data, err := typing.DistanceFilterUnmarshal(r)
			if err != nil {
				return err
			}

			log.Debugf("new flight for route %s-%s", data.Origin, data.Destination)
			distanceMi, err := f.comp.Distance(data.Origin, data.Destination)
			if err != nil {
				return err
			}
			if float64(data.Distance) > distanceFactor*distanceMi {
				log.Debugf("long flight: %x", data.ID)
				typing.ResultQ2Marshal(b, &data)
			}
		}
		if b.Len() > len(f.id) {
			if err := bc.Publish(ctx, f.m, f.sink, f.sink, b.Bytes()); err != nil {
				return err
			}
		}
		// TODO: store state
		if err := f.m.Ack(tag); err != nil {
			return err
		}
	}

	return context.Cause(ctx)
}
