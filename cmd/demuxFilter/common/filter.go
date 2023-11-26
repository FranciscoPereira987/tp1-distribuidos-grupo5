package common

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/duplicates"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

const (
	Distance = iota
	Fastest
	Average
	Result
)

type Filter struct {
	m        *mid.Middleware
	id       string
	sinks    []string
	keyGens  []mid.KeyGenerator
	filter   *duplicates.DuplicateFilter
	stateMan *state.StateManager
}

func NewFilter(m *mid.Middleware, id string, sinks []string, nWorkers []int, workDir string) *Filter {

	kgs := make([]mid.KeyGenerator, 0, len(nWorkers))
	for _, mod := range nWorkers {
		kgs = append(kgs, mid.NewKeyGenerator(mod))
	}
	return &Filter{
		m:        m,
		id:       id,
		sinks:    sinks,
		keyGens:  kgs,
		filter:   duplicates.NewDuplicateFilter(nil),
		stateMan: state.NewStateManager(filepath.Join(workDir, "demux", fmt.Sprintf("filter-%s.state", workDir))),
	}
}

func recoverSinks(stateMan *state.StateManager) (sinks []string) {
	values, ok := stateMan.GetFromState("sinks")
	if array, casted := values.([]any); ok && casted {
		for _, value := range array {
			sinks = append(sinks, value.(string))
		}
	}
	return
}

func recoverKeyGens(stateMan *state.StateManager) (keyGens []mid.KeyGenerator) {
	values, ok := stateMan.GetFromState("generators")
	if array, casted := values.([]any); ok && casted {
		for _, generator := range array {
			keyGens = append(keyGens, generator.(mid.KeyGenerator))
		}
	}
	return
}

func RecoverFromState(m *mid.Middleware, stateMan *state.StateManager) (f *Filter) {
	f = new(Filter)
	f.m = m
	f.id = stateMan.GetString("id")
	f.sinks = recoverSinks(stateMan)
	f.keyGens = recoverKeyGens(stateMan)
	f.filter = duplicates.NewDuplicateFilter(nil)
	f.filter.RecoverFromState(stateMan)
	f.stateMan = stateMan
	return
}

func (f *Filter) StoreState() error {
	f.stateMan.AddToState("id", f.id)
	f.stateMan.AddToState("sinks", strings.Join(f.sinks, ";"))
	f.stateMan.AddToState("generators", f.keyGens)
	f.filter.AddToState(f.stateMan)

	return f.stateMan.DumpState()
}

func (f *Filter) Run(ctx context.Context, ch <-chan mid.Delivery) error {
	err := os.MkdirAll(filepath.Dir(f.stateMan.Filename), 0755)
	if err != nil {
		return err
	}
	fareSum, fareCount := 0.0, 0
	dc := f.m.NewDeferredConfirmer(ctx)

	rr := f.keyGens[Distance].NewRoundRobinKeysGenerator()
	for d := range ch {
		msg, tag := d.Msg, d.Tag
		if f.filter.IsDuplicate(msg) {
			f.m.Ack(tag)
			continue
		}
		var (
			bDistance = bytes.NewBufferString(f.id)
			bResult   = bytes.NewBufferString(f.id)
			mAverage  = make(map[string]*bytes.Buffer)
			mFastest  = make(map[string]*bytes.Buffer)
		)
		for r := bytes.NewReader(msg); r.Len() > 0; {
			data, err := typing.FlightUnmarshal(r)
			if err != nil {
				return err
			}
			fareSum += float64(data.Fare)
			fareCount++

			f.marshalDistanceFilter(bDistance, &data)
			f.marshalAverageFilter(mAverage, &data)
			if strings.Count(data.Stops, "||") >= 3 {
				f.marshalFastestFilter(mFastest, &data)
				f.marshalResult(bResult, &data)
			}
		}
		if err := f.sendBuffer(ctx, dc, f.sinks[Distance], rr.NextKey(), bDistance); err != nil {
			return err
		}
		if err := f.sendMap(ctx, dc, f.sinks[Average], mAverage); err != nil {
			return err
		}
		if len(mFastest) > 0 {
			if err := f.sendMap(ctx, dc, f.sinks[Fastest], mFastest); err != nil {
				return err
			}
			if err := f.sendBuffer(ctx, dc, f.sinks[Result], f.sinks[Result], bResult); err != nil {
				return err
			}
		}
		if err := dc.Confirm(ctx); err != nil {
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

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
	}

	return f.sendAverageFare(ctx, fareSum, fareCount)
}

func (f *Filter) marshalDistanceFilter(b *bytes.Buffer, data *typing.Flight) {
	// ignore flights with missing required field
	if data.Distance > 0 {
		typing.DistanceFilterMarshal(b, data)
	}
}

func (f *Filter) marshalAverageFilter(m map[string]*bytes.Buffer, data *typing.Flight) {
	key := f.keyGens[Average].KeyFrom(data.Origin, data.Destination)
	b, ok := m[key]
	if !ok {
		b = bytes.NewBufferString(f.id)
		m[key] = b
	}

	typing.AverageFilterMarshal(b, data)
}

func (f *Filter) marshalFastestFilter(m map[string]*bytes.Buffer, data *typing.Flight) {
	key := f.keyGens[Fastest].KeyFrom(data.Origin, data.Destination)
	b, ok := m[key]
	if !ok {
		b = bytes.NewBufferString(f.id)
		m[key] = b
	}

	typing.FastestFilterMarshal(b, data)
}

func (f *Filter) marshalResult(b *bytes.Buffer, data *typing.Flight) {
	typing.ResultQ1Marshal(b, data)
}

func (f *Filter) sendBuffer(ctx context.Context, c mid.Confirmer, sink, key string, b *bytes.Buffer) error {
	return f.m.Publish(ctx, c, sink, key, b.Bytes())
}

func (f *Filter) sendMap(ctx context.Context, c mid.Confirmer, sink string, m map[string]*bytes.Buffer) error {
	for key, b := range m {
		if err := f.m.Publish(ctx, c, sink, key, b.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (f *Filter) sendAverageFare(ctx context.Context, fareSum float64, fareCount int) error {
	var bc mid.BasicConfirmer
	b := bytes.NewBufferString(f.id)
	typing.AverageFareMarshal(b, fareSum, fareCount)
	return bc.Publish(ctx, f.m, f.sinks[Average], "average", b.Bytes())
}
