package common

import (
	"bufio"
	"bytes"
	"context"
	"io"

	log "github.com/sirupsen/logrus"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

const workerId = "input"

type Gateway struct {
	m       *mid.Middleware
	id      string
	coords  string
	flights string
}

func NewGateway(m *mid.Middleware, id, coords, flights string) *Gateway {
	return &Gateway{
		m:       m,
		id:      id,
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
	n, err := g.ForwardCoords(ctx, &coordsReader)
	log.Infof("received %d airport coordinates records", n)
	if err != nil {
		return err
	}
	if err := g.m.EOF(ctx, g.coords, workerId, g.id); err != nil {
		return err
	}

	flightsReader, err := protocol.NewFileReader(r)
	if err != nil {
		return err
	}
	if err := g.ForwardFlights(ctx, &flightsReader, demuxers); err != nil {
		return err
	}
	return g.m.EOF(ctx, g.flights, workerId, g.id)
}

func (g *Gateway) ForwardCoords(ctx context.Context, in io.Reader) (int, error) {
	r, indices, err := protocol.NewCsvReader(in, ';', typing.CoordinatesFields)
	if err != nil {
		return 0, err
	}

	var bc mid.BasicConfirmer
	i := mid.MaxMessageSize / typing.AirportCoordsSize
	b := bytes.NewBufferString(g.id)
	for n := 0; ; n++ {
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				if i != mid.MaxMessageSize/typing.AirportCoordsSize {
					err = bc.Publish(ctx, g.m, g.coords, "coords", b.Bytes())
				} else {
					err = nil
				}
			}
			return n, err
		}

		if err := typing.AirportCoordsMarshal(b, record, indices); err != nil {
			return n, err
		}
		if i--; i <= 0 {
			if err := bc.Publish(ctx, g.m, g.coords, "coords", b.Bytes()); err != nil {
				return n, err
			}
			i = mid.MaxMessageSize / typing.AirportCoordsSize
			b = bytes.NewBufferString(g.id)
		}
	}
}

func (g *Gateway) ForwardFlights(ctx context.Context, in io.Reader, demuxers int) error {
	r, indices, err := protocol.NewCsvReader(in, ',', typing.FlightFields)
	if err != nil {
		return err
	}

	messageId := uint64(0)
	rr := mid.KeyGenerator(demuxers).NewRoundRobinKeysGenerator()
	var bc mid.BasicConfirmer
	i := mid.MaxMessageSize / typing.FlightSize
	b := bytes.NewBufferString(g.id)
	h := typing.NewHeader("input", messageId)
	h.Marshal(b)
	for {
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				if i != mid.MaxMessageSize/typing.FlightSize {
					err = bc.Publish(ctx, g.m, "", rr.NextKey(g.flights), b.Bytes())
				} else {
					err = nil
				}
			}
			return err
		}

		switch err := typing.FlightMarshal(b, record, indices); err {
		case nil:
		case typing.ErrMissingDistance:
			log.Warnf("action: ignore_error | id: %s | error: %s", record[0], err)
		default:
			log.Errorf("action: skip_flight | id: %s | error: %s", record[0], err)
			continue
		}
		if i--; i <= 0 {
			if err := bc.Publish(ctx, g.m, "", rr.NextKey(g.flights), b.Bytes()); err != nil {
				return err
			}
			i = mid.MaxMessageSize / typing.AirportCoordsSize
			b = bytes.NewBufferString(g.id)
			messageId++
			h := typing.NewHeader("input", messageId)
			h.Marshal(b)
		}
	}
}
