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
	log "github.com/sirupsen/logrus"
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
	workerId string
	clientId string
	sinks    []string
	keyGens  []mid.KeyGenerator
	filter   *duplicates.DuplicateFilter
	stateMan *state.StateManager
}

func NewFilter(m *mid.Middleware, workerId, clientId string, sinks []string, nWorkers []int, workDir string) *Filter {

	kgs := make([]mid.KeyGenerator, 0, len(nWorkers))
	for _, mod := range nWorkers {
		kgs = append(kgs, mid.NewKeyGenerator(mod))
	}
	return &Filter{
		m:        m,
		workerId: workerId,
		clientId: clientId,
		sinks:    sinks,
		keyGens:  kgs,
		filter:   duplicates.NewDuplicateFilter(),
		stateMan: state.NewStateManager(workDir),
	}
}

func RecoverFromState(m *mid.Middleware, workerId, clientId string, sinks []string, nWorkers []int, stateMan *state.StateManager) *Filter {
	f := new(Filter)
	f.m = m
	f.clientId = clientId
	f.sinks = sinks
	for _, mod := range nWorkers {
		f.keyGens = append(f.keyGens, mid.NewKeyGenerator(mod))
	}
	f.filter = duplicates.NewDuplicateFilter()
	f.filter.RecoverFromState(stateMan)
	f.stateMan = stateMan
	return f
}

func (f *Filter) Restart(ctx context.Context, toRestart map[string]*Filter) {

	switch v, _ := f.stateMan.State["state"].(int); v {
	case Recieving:
		log.Infof("action: restarting reciving filter | result: in progress")
		toRestart[f.clientId] = f
	case NotYetSent:
		ctx, cancel := context.WithCancel(ctx)
		go func() {
			defer cancel()
			log.Info("action: restarting sending filter | result: in progress")
			fareSum, fareCount := f.GetFareInfo()
			if err := f.sendAverageFare(ctx, fareSum, fareCount); err != nil {
				log.Infof("action: sending average fare | result: failed | reason: %s", err)
			}
			if err := f.Close(); err != nil {
				log.Infof("action: removing state files | result: failed | reason: %s", err)
			}
		}()
	case Finished:
		//Already finished, so just go ahead and finished execution
		if err := f.Close(); err != nil {
			log.Infof("action: restarting finished filter | result: failed | reason: %s", err)
		}
	}
}

func (f *Filter) Close() error {
	return os.RemoveAll(filepath.Base(f.stateMan.Filename))
}

func (f *Filter) StoreState(sum float64, count, state int) error {
	f.filter.AddToState(f.stateMan)

	f.stateMan.AddToState("sum", sum)
	f.stateMan.AddToState("count", count)
	f.stateMan.AddToState("state", state)
	return f.stateMan.DumpState()
}

func (f *Filter) GetFareInfo() (fareSum float64, fareCount int) {
	fareCount, _ = f.stateMan.State["count"].(int)
	fareSum, _ = f.stateMan.State["sum"].(float64)
	return
}

func (f *Filter) GetRoundRobinGenerator() (gen mid.RoundRobinKeysGenerator) {

	return mid.RoundRobinFromState(f.stateMan, f.keyGens[Distance])
}

func (f Filter) marshalHeaderInto(b *bytes.Buffer) {
	h := typing.NewHeader(f.workerId, f.filter.LastMessage)
	h.Marshal(b)
}

func (f *Filter) Run(ctx context.Context, ch <-chan mid.Delivery) error {
	err := os.MkdirAll(filepath.Dir(f.stateMan.Filename), 0755)
	if err != nil {
		return err
	}
	fareSum, fareCount := f.GetFareInfo()
	dc := f.m.NewDeferredConfirmer(ctx)

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
		f.marshalHeaderInto(bDistance)
		f.marshalHeaderInto(bResult)
		for r.Len() > 0 {
			data, err := typing.FlightUnmarshal(r)
			if err != nil {
				return err
			}
			fareSum += float64(data.Fare)
			fareCount++

			f.marshalDistanceFilter(bDistance, &data)
			f.marshalAverageFilter(mAverage, f.sinks[Average], &data)
			if strings.Count(data.Stops, "||") >= 3 {
				f.marshalFastestFilter(mFastest, f.sinks[Fastest], &data)
				f.marshalResult(bResult, &data)
			}
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
		rr.AddToState(f.stateMan)
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

func (f *Filter) marshalAverageFilter(m map[string]*bytes.Buffer, sink string, data *typing.Flight) {
	key := f.keyGens[Average].KeyFrom(sink, data.Origin, data.Destination)
	b, ok := m[key]
	if !ok {
		b = bytes.NewBufferString(f.clientId)
		f.marshalHeaderInto(b)
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

func (f *Filter) sendAverageFare(ctx context.Context, fareSum float64, fareCount int) error {
	var bc mid.BasicConfirmer
	b := bytes.NewBufferString(f.clientId)
	typing.AverageFareMarshal(b, fareSum, fareCount)
	err := bc.Publish(ctx, f.m, f.sinks[Average], "average", b.Bytes())

	if err == nil {
		f.StoreState(fareSum, fareCount, Finished)
	}

	return err
}
