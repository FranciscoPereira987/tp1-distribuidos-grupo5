package common

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
)

const workerId = "input"

const (
	_ = iota
	SentCoords     // No EOF yet
	SentCoordsEof  // after coords EOF
	SendingFlights // after reading file size
)

type Gateway struct {
	m       *mid.Middleware
	id      string
	coords  string
	flights string
	workdir string

	stateMan *state.StateManager
}

func NewGateway(m *mid.Middleware, id, coords, flights, workdir string, sm *state.StateManager) (*Gateway, error) {
	err := os.MkdirAll(workdir, 0755)
	return &Gateway{
		m,
		id,
		coords,
		flights,
		workdir,
		sm,
	}, err
}

func (g *Gateway) Close() error {
	return state.RemoveWorkdir(g.workdir)
}

func (g *Gateway) Run(ctx context.Context, in io.Reader, demuxers int) error {
	var (
		flightsReader protocol.ExactReader
		lastOffset int64
		err error
	)
	r := bufio.NewReader(in)
	switch step, _ := g.stateMan.GetInt("step"); step {
	case SentCoords:
		goto coordsEof
	case SentCoordsEof:
		goto sentCoordsEof
	case SendingFlights:
		offset, err := g.stateMan.GetInt64("offset")
		if err != nil {
			return err
		}
		size, err := g.stateMan.GetInt64("flights-size")
		if err != nil {
			return err
		}
		if offset > 0 {
			lastOffset = offset
		}
		flightsReader = protocol.ExactReader{R: r, N: size}
		goto sendingFlights
	}

	if err := g.SendCoords(ctx, r); err != nil {
		return err
	}

coordsEof:
	if err := g.SendCoordsEof(ctx); err != nil {
		return err
	}

sentCoordsEof:
	if flightsReader, err = protocol.NewFileReader(r); err != nil {
		return err
	}
	g.stateMan.State["step"] = SendingFlights
	g.stateMan.State["flights-size"] = flightsReader.N
	g.stateMan.State["offset"] = -1
	if err := g.stateMan.DumpState(); err != nil {
		return fmt.Errorf("failed to dump state for sending flights", err)
	}
sendingFlights:
	if err := g.ForwardFlights(ctx, &flightsReader, demuxers, lastOffset); err != nil {
		return err
	}
	return g.m.EOF(ctx, g.flights, workerId, g.id)
}

func (g *Gateway) SendCoords(ctx context.Context, r io.Reader) error {
	if err := g.stateMan.DumpState(); err != nil {
		return fmt.Errorf("failed to dump initial state: %w", err)
	}
	g.stateMan.State["step"] = SentCoords
	if err := g.stateMan.Prepare(); err != nil {
		return fmt.Errorf("failed to prepare state for sent coordinates: %w", err)
	}
	coordsReader, err := protocol.NewFileReader(r)
	if err != nil {
		return err
	}
	n, err := g.ForwardCoords(ctx, &coordsReader)
	log.Infof("received %d airport coordinates records", n)
	if err != nil {
		return err
	}
	if err := g.stateMan.Commit(); err != nil {
		return fmt.Errorf("failed to commit state for sent coordinates: %s", err)
	}
	return nil
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

func (g *Gateway) SendCoordsEof(ctx context.Context) error {
	g.stateMan.State["step"] = SentCoordsEof
	if err := g.stateMan.Prepare(); err != nil {
		return fmt.Errorf("failed to prepare state for sent coordinates EOF: %s", err)
	}

	if err := g.m.EOF(ctx, g.coords, workerId, g.id); err != nil {
		return err
	}

	if err := g.stateMan.Commit(); err != nil {
		return fmt.Errorf("failed to commit state for sent coordinates EOF: %s", err)
	}
	return nil
}

func (g *Gateway) ForwardFlights(ctx context.Context, in io.Reader, demuxers int, lastOffset int64) error {
	var r *csv.Reader
	indices, err := g.stateMan.GetIntSlice("indices")
	switch {
	case err == nil:
		r = csv.NewReader(in)
	case errors.Is(err, state.ErrNotFound):
		r, indices, err = protocol.NewCsvReader(in, ',', typing.FlightFields)
	}
	if err != nil {
		return err
	}
	g.stateMan.State["indices"] = indices
	if err = g.Prepare(r, lastOffset); err != nil {
		return err
	}

	rr, err := mid.RoundRobinFromState(g.stateMan, mid.KeyGenerator(demuxers))
	if err != nil {
		return err
	}
	var bc mid.BasicConfirmer
	i := mid.MaxMessageSize / typing.FlightSize
	b := bytes.NewBufferString(g.id)
	h := typing.NewHeader("input", 0)
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
			h.MessageId++
			h.Marshal(b)
		}
	}
}

func (g *Gateway) Prepare(r *csv.Reader, lastOffset int64) error {
	g.stateMan.State["offset"] = r.InputOffset() + lastOffset
	return g.stateMan.Prepare()
}
