package common

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/duplicates"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

const distanceFactor = 4

type Filter struct {
	m        *mid.Middleware
	id       string
	sink     string
	workdir  string
	filter   *duplicates.DuplicateFilter
	stateMan *state.StateManager
}

func NewFilter(m *mid.Middleware, id, sink, workdir string) (*Filter, error) {
	err := os.MkdirAll(filepath.Join(workdir, "coordinates"), 0755)
	err = errors.Join(err, os.MkdirAll(filepath.Join(workdir, "distance"), 0755))
	return &Filter{
		m,
		id,
		sink,
		workdir,
		duplicates.NewDuplicateFilter(nil),
		state.NewStateManager(filepath.Join(workdir, "distance", fmt.Sprintf("fitler-%s.state", id))),
	}, err
}

func RecoverFromState(m *mid.Middleware, workdir string, stateMan *state.StateManager) (id string, f *Filter, onFlights bool) {
	f = new(Filter)
	id = stateMan.GetString("id")
	f.m = m
	f.id = id
	f.sink = stateMan.GetString("sink")
	f.workdir = workdir
	f.filter = duplicates.NewDuplicateFilter(nil)
	f.filter.RecoverFromState(stateMan)
	f.stateMan = stateMan
	onFlights, _ = stateMan.Get("coordinates-load").(bool)
	return
}

func (f *Filter) Restart(ctx context.Context, delivery <-chan mid.Delivery) error {
	coordinatedLoaded, _ := f.stateMan.Get("coordinates-load").(bool)
	if !coordinatedLoaded {
		logrus.Info("Restarting at AddCoordinates")
		if err := f.AddCoords(ctx, delivery); err != nil {
			return err
		}
	}

	return f.Run(ctx, delivery)
}

func (f *Filter) StoreState() error {
	f.stateMan.AddToState("id", f.id)
	f.stateMan.AddToState("sink", f.sink)
	f.filter.AddToState(f.stateMan)
	return f.stateMan.DumpState()
}

func (f *Filter) Close() error {
	return os.RemoveAll(f.workdir)
}

func (f *Filter) AddCoords(ctx context.Context, coords <-chan mid.Delivery) error {
	for d := range coords {
		msg, tag := d.Msg, d.Tag
		if f.filter.IsDuplicate(msg) {
			f.m.Ack(tag)
			continue
		}
		code, err := typing.ReadString(bytes.NewReader(msg))
		if err != nil {
			return err
		}
		if err := state.WriteFile(filepath.Join(f.workdir, "coordinates", code), msg); err != nil {
			return err
		}
		f.filter.ChangeLast(msg)
		if err := f.StoreState(); err != nil {
			return err
		}
		if err := f.m.Ack(tag); err != nil {
			return err
		}
	}
	f.stateMan.AddToState("coordinates-load", true)
	if err := f.StoreState(); err != nil {
		return err
	}
	return context.Cause(ctx)
}

func (f *Filter) Run(ctx context.Context, flights <-chan mid.Delivery) error {
	var bc mid.BasicConfirmer

	comp, err := f.loadDistanceComputer()
	if err != nil {
		return err
	}

	for d := range flights {
		msg, tag := d.Msg, d.Tag
		b := bytes.NewBufferString(f.id)
		for r := bytes.NewReader(msg); r.Len() > 0; {
			data, err := typing.DistanceFilterUnmarshal(r)
			if err != nil {
				return err
			}

			log.Debugf("new flight for route %s-%s", data.Origin, data.Destination)
			distanceMi, err := comp.Distance(data.Origin, data.Destination)
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
		if err := f.StoreState(); err != nil {
			return err
		}
		if err := f.m.Ack(tag); err != nil {
			return err
		}
	}

	return context.Cause(ctx)
}

func (f *Filter) loadDistanceComputer() (*distance.DistanceComputer, error) {
	coordsDir := filepath.Join(f.workdir, "coordinates")
	files, err := os.ReadDir(coordsDir)
	if err != nil {
		return nil, err
	}

	comp := distance.NewComputer()
	for _, file := range files {
		if state.IsTmp(file.Name()) {
			os.Remove(filepath.Join(coordsDir, file.Name()))
			continue
		}
		if err := loadCoordinates(comp, filepath.Join(coordsDir, file.Name())); err != nil {
			return nil, err
		}
	}

	return comp, nil
}

func loadCoordinates(comp *distance.DistanceComputer, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		data, err := typing.AirportCoordsUnmarshal(r)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		comp.AddAirportCoords(data.Code, data.Lat, data.Lon)
	}
}
