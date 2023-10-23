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
	source  string
	sinks   []string
	keyGens []mid.KeyGenerator
}

func NewFilter(m *mid.Middleware, source string, sinks []string, nWorkers []int) *Filter {
	kgs := make([]mid.KeyGenerator, 0, len(nWorkers))
	for _, mod := range nWorkers {
		kgs = append(kgs, mid.NewKeyGenerator(mod))
	}
	return &Filter{
		m:       m,
		source:  source,
		sinks:   sinks,
		keyGens: kgs,
	}
}

func (f *Filter) Run(ctx context.Context) error {
	fareSum, fareCount := 0.0, 0
	ch, err := f.m.ConsumeWithContext(ctx, f.source)
	if err != nil {
		return err
	}

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
	var b bytes.Buffer
	key := f.keyGens[Query2].KeyFrom(data.Origin, data.Destination)
	typing.DistanceFilterMarshal(&b, data)
	return f.m.PublishWithContext(ctx, f.sinks[Query2], key, b.Bytes())
}

func (f *Filter) sendAverageFilter(ctx context.Context, data *typing.Flight) error {
	var b bytes.Buffer
	key := f.keyGens[Query4].KeyFrom(data.Origin, data.Destination)
	typing.AverageFilterMarshal(&b, data)
	return f.m.PublishWithContext(ctx, f.sinks[Query4], key, b.Bytes())
}

func (f *Filter) sendFastestFilter(ctx context.Context, data *typing.Flight) error {
	var b bytes.Buffer
	key := f.keyGens[Query3].KeyFrom(data.Origin, data.Destination)
	typing.FastestFilterMarshal(&b, data)
	return f.m.PublishWithContext(ctx, f.sinks[Query3], key, b.Bytes())
}

func (f *Filter) sendResult(ctx context.Context, data *typing.Flight) error {
	var b bytes.Buffer
	sink := f.sinks[Result]
	typing.ResultQ1Marshal(&b, data)
	return f.m.PublishWithContext(ctx, sink, sink, b.Bytes())
}

func (f *Filter) sendAverageFare(ctx context.Context, fareSum float64, fareCount int) error {
	var b bytes.Buffer
	typing.AverageFareMarshal(&b, fareSum, fareCount)
	return f.m.PublishWithContext(ctx, f.sinks[Query4], "average", b.Bytes())
}
