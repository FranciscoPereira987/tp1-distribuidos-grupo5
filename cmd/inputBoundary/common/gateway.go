package common

import (
	"bufio"
	"bytes"
	"context"
	"io"

	// log "github.com/sirupsen/logrus"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

type Gateway struct {
	m       *mid.Middleware
	coords  string
	flights string
}

func NewGateway(m *mid.Middleware, coords, flights string) *Gateway {
	return &Gateway{
		m:       m,
		coords:  coords,
		flights: flights,
	}
}

func (g *Gateway) Run(ctx context.Context, in io.Reader, demuxers int) error {
	r := bufio.NewReader(in)
	coordsReader, err := protocol.NewFileReader(r)
	if err != nil {
		return err
	}
	if err := g.ForwardCoords(ctx, &coordsReader); err != nil {
		return err
	}
	if err := g.m.EOF(ctx, g.coords); err != nil {
		return err
	}

	flightsReader, err := protocol.NewFileReader(r)
	if err != nil {
		return err
	}
	if err := g.ForwardFlights(ctx, &flightsReader); err != nil {
		return err
	}
	return g.m.SharedQueueEOF(ctx, g.flights, byte(demuxers))
}

func (g *Gateway) ForwardCoords(ctx context.Context, in io.Reader) error {
	r, indices, err := protocol.NewCsvReader(in, ';', typing.CoordinatesFields)
	if err != nil {
		return err
	}

	for {
		var b bytes.Buffer
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if err := typing.AirportCoordsMarshal(&b, record, indices); err != nil {
			return err
		}
		if err := g.m.PublishWithContext(ctx, g.coords, "coords", b.Bytes()); err != nil {
			return err
		}
	}
}

func (g *Gateway) ForwardFlights(ctx context.Context, in io.Reader) error {
	r, indices, err := protocol.NewCsvReader(in, ',', typing.FlightFields)
	if err != nil {
		return err
	}

	for {
		var b bytes.Buffer
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if err := typing.FlightMarshal(&b, record, indices); err != nil {
			return err
		}
		if err := g.m.PublishWithContext(ctx, g.flights, g.flights, b.Bytes()); err != nil {
			return err
		}
	}
}
