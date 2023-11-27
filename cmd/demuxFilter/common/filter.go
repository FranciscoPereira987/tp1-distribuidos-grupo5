package common

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/duplicates"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/sirupsen/logrus"
)

const (
	Distance = iota
	Fastest
	Average
	Result
)

const (
	Recieving = iota
	NotYetSent
	Finished 
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
		stateMan: state.NewStateManager(workDir),
	}
}

func recoverKeyGens(stateMan *state.StateManager) (keyGens []mid.KeyGenerator) {
	values := stateMan.Get("generators")
	if array, ok := values.([]any); ok {
		for _, generator := range array {
			keyGens = append(keyGens, generator.(mid.KeyGenerator))
		}
	}
	return
}

func RecoverFromState(m *mid.Middleware, id string, sinks []string, stateMan *state.StateManager) (f *Filter) {
	f = new(Filter)
	f.m = m
	f.id = id
	f.sinks = sinks
	f.keyGens = recoverKeyGens(stateMan)
	f.filter = duplicates.NewDuplicateFilter(nil)
	f.filter.RecoverFromState(stateMan)
	f.stateMan = stateMan
	return
}
func (f *Filter) Restart(ctx context.Context, toRestart map[string]*Filter)  {

	switch f.stateMan.GetInt("state") {
	case Recieving:
		logrus.Infof("action: restarting reciving filter | result: in progress")
		toRestart[f.id] = f
	case NotYetSent:
		ctx, cancel := context.WithCancel(ctx)
		go func() {
			defer cancel()
			logrus.Info("action: restarting sending filter | result: in progress")
			fareSum, fareCount := f.GetFareInfo()
			if err := f.sendAverageFare(ctx, fareSum, fareCount); err != nil {
				logrus.Infof("action: sending average fare | result: failed | reason: %s", err)
			}
			if err := f.Close(); err != nil {
				logrus.Infof("action: removing state files | result: failed | reason: %s", err)
			}
		}()
	case Finished:
		//Already finished, so just go ahead and finished execution
		if err := f.Close(); err != nil {
			logrus.Infof("action: restarting finished filter | result: failed | reason: %s", err)
		}
	}
}

func (f *Filter) Close() error {
	return os.RemoveAll(filepath.Base(f.stateMan.Filename))
}

func (f *Filter) StoreState(sum float64, count, state int) error {
	f.stateMan.AddToState("generators", f.keyGens)
	f.filter.AddToState(f.stateMan)

	f.stateMan.AddToState("sum", sum)
	f.stateMan.AddToState("count", count)
	f.stateMan.AddToState("state", state)

	return f.stateMan.DumpState()
}

func (f *Filter) GetFareInfo() (fareSum float64, fareCount int) {
	fareCount = f.stateMan.GetInt("count")
	fareSum, _ = f.stateMan.Get("sum").(float64)
	return
}

func (f *Filter) Run(ctx context.Context, ch <-chan mid.Delivery) error {
	err := os.MkdirAll(filepath.Dir(f.stateMan.Filename), 0755)
	if err != nil {
		return err
	}
	fareSum, fareCount := f.GetFareInfo()
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
		if err := f.StoreState(fareSum, fareCount, Recieving); err != nil {
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
		f.StoreState(fareSum, fareCount, NotYetSent)
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
	err := bc.Publish(ctx, f.m, f.sinks[Average], "average", b.Bytes())

	if err == nil {
		f.StoreState(fareSum, fareCount, Finished)
	}

	return err
}

