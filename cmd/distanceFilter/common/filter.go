package common

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"

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
	workerId string
	clientId string
	sink     string
	workdir  string
	stateMan *state.StateManager
}

func NewFilter(m *mid.Middleware, workerId, clientId, sink, workdir string) (*Filter, error) {
	err := os.MkdirAll(filepath.Join(workdir, "coordinates"), 0755)
	return &Filter{
		m,
		workerId,
		clientId,
		sink,
		workdir,
		state.NewStateManager(workdir),
	}, err
}

func NewWithState(m *mid.Middleware, workerId, clientId, sink, workdir string, stateMan *state.StateManager) (*Filter, error) {
	filter, err := NewFilter(m, workerId, clientId, sink, workdir)
	filter.stateMan = stateMan
	return filter, err
}

func (f *Filter) Close() error {
	return state.RemoveWorkdir(f.workdir)
}

func (f *Filter) AddCoords(ctx context.Context, coords <-chan mid.Delivery) error {

	if err := f.stateMan.Prepare(); err != nil {
		return err
	}
	for d := range coords {
		msg, tag := d.Msg, d.Tag
		code, err := typing.ReadString(bytes.NewReader(msg))
		if err != nil {
			return err
		}
		// We use state.WriteFile() because it conveniently syncs the
		// filesystem after it's done.
		// Otherwise, os.WriteFile() would suffice.
		if err := state.WriteFile(filepath.Join(f.workdir, "coordinates", code), msg); err != nil {
			return err
		}
		if err := f.m.Ack(tag); err != nil {
			return err
		}
	}
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
		return f.stateMan.Commit()
	}
}

func (f *Filter) Run(ctx context.Context, flights <-chan mid.Delivery) error {
	df := duplicates.NewDuplicateFilter()
	sm := f.stateMan
	if err := df.RecoverFromState(sm); err != nil {
		return err
	}
	var bc mid.BasicConfirmer

	h, err := typing.RecoverHeader(sm, f.workerId)
	if err != nil {
		return err
	}
	comp, err := f.loadDistanceComputer()
	if err != nil {
		return err
	}

	for d := range flights {
		msg, tag := d.Msg, d.Tag
		r := bytes.NewReader(msg)
		dup, err := df.Update(r)
		if err != nil {
			log.Errorf("action: reading_batch | status: failed | reason: %s", err)
		}
		if dup || err != nil {
			f.m.Ack(tag)
			continue
		}
		h.MessageId++
		h.AddToState(f.stateMan.State)
		df.AddToState(sm)
		if err := sm.Prepare(); err != nil {
			return err
		}
		b := bytes.NewBufferString(f.clientId)
		for r.Len() > 0 {
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
				f.marshalResult(b, &h, &data)
			}
		}
		if b.Len() > len(f.clientId) {
			if err := bc.Publish(ctx, f.m, "", f.sink, b.Bytes()); err != nil {
				return err
			}
		}
		if err := sm.Commit(); err != nil {
			return err
		}
		if err := f.m.Ack(tag); err != nil {
			return err
		}
	}

	return context.Cause(ctx)
}

func (f *Filter) marshalResult(b *bytes.Buffer, h *typing.BatchHeader, data *typing.DistanceFilter) {
	if b.Len() == len(f.clientId) {
		h.Marshal(b)
	}
	typing.ResultQ2Marshal(b, data)

}

func (f *Filter) loadDistanceComputer() (*distance.DistanceComputer, error) {
	coordsDir := filepath.Join(f.workdir, "coordinates")
	files, err := os.ReadDir(coordsDir)
	if err != nil {
		return nil, err
	}

	comp := distance.NewComputer()
	for _, file := range files {
		if file.IsDir() {
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
