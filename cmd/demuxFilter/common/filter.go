package common

import (
	"bytes"
	"context"
	"strings"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

const (
	Query2 = iota
	Query3
	Query4
	Result
)

type Filter struct {
	m       *mid.Middleware
	id      string
	sinks   []string
	keyGens []mid.KeyGenerator
}

func NewFilter(m *mid.Middleware, id string, sinks []string, nWorkers []int) *Filter {
	kgs := make([]mid.KeyGenerator, 0, len(nWorkers))
	for _, mod := range nWorkers {
		kgs = append(kgs, mid.NewKeyGenerator(mod))
	}
	return &Filter{
		m:       m,
		id:      id,
		sinks:   sinks,
		keyGens: kgs,
	}
}

func (f *Filter) Run(ctx context.Context, ch <-chan []byte) error {
	fareSum, fareCount := 0.0, 0

	for msg := range ch {
		data, err := typing.FlightUnmarshal(msg)
		if err != nil {
			return err
		}
		fareSum += float64(data.Fare)
		fareCount++
		if err := f.sendDistanceFilter(ctx, &data); err != nil {
			return err
		}
		if err := f.sendAverageFilter(ctx, &data); err != nil {
			return err
		}
		if strings.Count(data.Stops, "||") >= 3 {
			if err := f.sendFastestFilter(ctx, &data); err != nil {
				return err
			}
			if err := f.sendResult(ctx, &data); err != nil {
				return err
			}
		}
	}

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
	}

	return f.sendAverageFare(ctx, fareSum, fareCount)
}

func (f *Filter) sendDistanceFilter(ctx context.Context, data *typing.Flight) error {
	// ignore flights with missing required field
	if data.Distance == 0 {
		return nil
	}

	b := bytes.NewBufferString(f.id)
	sink := f.sinks[Query2]
	typing.DistanceFilterMarshal(b, data)
	return f.m.Publish(ctx, sink, sink, b.Bytes())
}

func (f *Filter) sendAverageFilter(ctx context.Context, data *typing.Flight) error {
	b := bytes.NewBufferString(f.id)
	key := f.keyGens[Query4].KeyFrom(data.Origin, data.Destination)
	typing.AverageFilterMarshal(b, data)
	return f.m.Publish(ctx, f.sinks[Query4], key, b.Bytes())
}

func (f *Filter) sendFastestFilter(ctx context.Context, data *typing.Flight) error {
	b := bytes.NewBufferString(f.id)
	key := f.keyGens[Query3].KeyFrom(data.Origin, data.Destination)
	typing.FastestFilterMarshal(b, data)
	return f.m.Publish(ctx, f.sinks[Query3], key, b.Bytes())
}

func (f *Filter) sendResult(ctx context.Context, data *typing.Flight) error {
	b := bytes.NewBufferString(f.id)
	sink := f.sinks[Result]
	typing.ResultQ1Marshal(b, data)
	return f.m.Publish(ctx, sink, sink, b.Bytes())
}

func (f *Filter) sendAverageFare(ctx context.Context, fareSum float64, fareCount int) error {
	b := bytes.NewBufferString(f.id)
	typing.AverageFareMarshal(b, fareSum, fareCount)
	return f.m.Publish(ctx, f.sinks[Query4], "average", b.Bytes())
}
