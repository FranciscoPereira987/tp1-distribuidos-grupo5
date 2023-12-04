package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/duplicates"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	log "github.com/sirupsen/logrus"
)

const (
	Distance = iota
	Fastest
	Average
	Result
)

const (
	Receiving = iota
	NotYetSent
	Finished
)

var ErrUnsupported = errors.New("unsupported operation")

type Filter struct {
	m        *mid.Middleware
	workerId string
	clientId string
	sinks    []string
	keyGens  []mid.KeyGenerator
	workdir  string
	filter   *duplicates.DuplicateFilter
	stateMan *state.StateManager
}

func NewFilter(m *mid.Middleware, workerId, clientId string, sinks []string, workdir string, nWorkers []int) (*Filter, error) {
	err := os.MkdirAll(workdir, 0755)

	kgs := make([]mid.KeyGenerator, 0, len(nWorkers))
	for _, mod := range nWorkers {
		kgs = append(kgs, mid.NewKeyGenerator(mod))
	}
	return &Filter{
		m:        m,
		workerId: workerId,
		clientId: clientId,
		sinks:    sinks,
		workdir:  workdir,
		keyGens:  kgs,
		filter:   duplicates.NewDuplicateFilter(),
		stateMan: state.NewStateManager(workdir),
	}, err
}

func RecoverFromState(m *mid.Middleware, workerId, clientId string, sinks []string, workdir string, nWorkers []int, stateMan *state.StateManager) (*Filter, error) {
	f, err := NewFilter(m, workerId, clientId, sinks, workdir, nWorkers)
	if err == nil {
		err = f.filter.RecoverFromState(stateMan)
	}
	f.stateMan = stateMan
	return f, err
}

func (f *Filter) ShouldRestart() bool {
	v, _ := f.stateMan.State["state"].(int)
	return v != Receiving
}

func (f *Filter) Restart(ctx context.Context) error {

	switch v, _ := f.stateMan.State["state"].(int); v {
	default:
		return fmt.Errorf("action: restarting | result: failure | reason: %w", ErrUnsupported)
	case NotYetSent:
		log.Info("action: restarting sending filter | result: in progress")
		fareSum, fareCount := f.GetFareInfo()
		h, err := typing.RecoverHeader(f.stateMan.State, f.workerId)
		if err == nil {
			err = f.sendAverageFare(ctx, &h, fareSum, fareCount)
		}
		return err
	case Finished:
		// Already finished, so just go ahead and finish execution
		return nil
	}
}

func (f *Filter) Close() error {
	return state.RemoveWorkdir(f.workdir)
}

func (f *Filter) Prepare(sum float64, count, state int) error {
	f.filter.AddToState(f.stateMan)
	f.stateMan.State["sum"] = sum
	f.stateMan.State["count"] = count
	f.stateMan.State["state"] = state

	return f.stateMan.Prepare()
}

func (f *Filter) GetFareInfo() (float64, int) {
	sum, _ := f.stateMan.State["sum"].(float64)
	count, _ := f.stateMan.State["count"].(int)
	return sum, count
}

func (f *Filter) GetRoundRobinGenerator() (gen mid.RoundRobinKeysGenerator) {

	return mid.RoundRobinFromState(f.stateMan, f.keyGens[Distance])
}

func (f *Filter) marshalHeaderInto(b *bytes.Buffer, h *typing.BatchHeader) {
	h.Marshal(b)
}

func (f *Filter) Run(ctx context.Context, ch <-chan mid.Delivery) error {
	fareSum, fareCount := f.GetFareInfo()
	dc := f.m.NewDeferredConfirmer(ctx)
	h, err := typing.RecoverHeader(f.stateMan.State, f.workerId)
	if err != nil {
		return err
	}

	rr := f.GetRoundRobinGenerator()
	for d := range ch {
		msg, tag := d.Msg, d.Tag
		r := bytes.NewReader(msg)
		dup, err := f.filter.Update(r)
		if err != nil {
			log.Errorf("action: reading_batch | status: failed | reason: %s", err)
		}
		if dup || err != nil {
			f.m.Ack(tag)
			continue
		}
		var (
			bDistance = bytes.NewBufferString(f.clientId)
			bResult   = bytes.NewBufferString(f.clientId)
			mAverage  = make(map[string]*bytes.Buffer)
			mFastest  = make(map[string]*bytes.Buffer)
		)
		f.marshalHeaderInto(bDistance, &h)
		f.marshalHeaderInto(bResult, &h)
		for r.Len() > 0 {
			data, err := typing.FlightUnmarshal(r)
			if err != nil {
				return err
			}
			fareSum += float64(data.Fare)
			fareCount++

			f.marshalDistanceFilter(bDistance, &data)
			f.marshalAverageFilter(mAverage, f.sinks[Average], &h, &data)
			if strings.Count(data.Stops, "||") >= 3 {
				f.marshalFastestFilter(mFastest, f.sinks[Fastest], &data)
				f.marshalResult(bResult, &data)
			}
		}
		h.MessageId++
		h.AddToState(f.stateMan.State)
		rr.AddToState(f.stateMan)
		if err := f.Prepare(fareSum, fareCount, Receiving); err != nil {
			return err
		}
		if err := f.sendBuffer(ctx, dc, rr.NextKey(f.sinks[Distance]), bDistance); err != nil {
			return err
		}
		if err := f.sendMap(ctx, dc, mAverage); err != nil {
			return err
		}
		if len(mFastest) > 0 {
			if err := f.sendMap(ctx, dc, mFastest); err != nil {
				return err
			}
			if err := f.sendBuffer(ctx, dc, f.sinks[Result], bResult); err != nil {
				return err
			}
		}
		if err := dc.Confirm(ctx); err != nil {
			return err
		}
		if err := f.stateMan.Commit(); err != nil {
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
		f.filter.RemoveFromState(f.stateMan)
		rr.RemoveFromState(f.stateMan)
		f.stateMan.State["state"] = NotYetSent
		if err := f.stateMan.DumpState(); err != nil {
			log.Error("action: dump_state | result: failure | reason:", err)
		}
	}

	return f.sendAverageFare(ctx, &h, fareSum, fareCount)
}

func (f *Filter) marshalDistanceFilter(b *bytes.Buffer, data *typing.Flight) {
	// ignore flights with missing required field
	if data.Distance > 0 {
		typing.DistanceFilterMarshal(b, data)
	}
}

func (f *Filter) marshalAverageFilter(m map[string]*bytes.Buffer, sink string, h *typing.BatchHeader, data *typing.Flight) {
	key := f.keyGens[Average].KeyFrom(sink, data.Origin, data.Destination)
	b, ok := m[key]
	if !ok {
		b = bytes.NewBufferString(f.clientId)
		f.marshalHeaderInto(b, h)
		m[key] = b
	}

	typing.AverageFilterMarshal(b, data)
}

func (f *Filter) marshalFastestFilter(m map[string]*bytes.Buffer, sink string, data *typing.Flight) {
	key := f.keyGens[Fastest].KeyFrom(sink, data.Origin, data.Destination)
	b, ok := m[key]
	if !ok {
		b = bytes.NewBufferString(f.clientId)
		m[key] = b
	}

	typing.FastestFilterMarshal(b, data)
}

func (f *Filter) marshalResult(b *bytes.Buffer, data *typing.Flight) {
	typing.ResultQ1Marshal(b, data)
}

func (f *Filter) sendBuffer(ctx context.Context, c mid.Confirmer, key string, b *bytes.Buffer) error {
	return f.m.Publish(ctx, c, "", key, b.Bytes())
}

func (f *Filter) sendMap(ctx context.Context, c mid.Confirmer, m map[string]*bytes.Buffer) error {
	for key, b := range m {
		if err := f.m.Publish(ctx, c, "", key, b.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (f *Filter) sendAverageFare(ctx context.Context, h *typing.BatchHeader, fareSum float64, fareCount int) error {
	var bc mid.BasicConfirmer
	b := bytes.NewBufferString(f.clientId)
	h.Marshal(b)
	typing.AverageFareMarshal(b, fareSum, fareCount)
	delete(f.stateMan.State, "sum")
	delete(f.stateMan.State, "count")
	f.stateMan.State["state"] = Finished
	if err := f.stateMan.Prepare(); err != nil {
		return err
	}

	err := bc.Publish(ctx, f.m, f.sinks[Average], "average", b.Bytes())
	if err == nil {
		f.stateMan.Commit()
	}

	return err
}
