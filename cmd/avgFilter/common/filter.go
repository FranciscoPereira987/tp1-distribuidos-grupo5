package common

import (
	"context"
	"strings"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
)

const stopsSep = "||"

type Filter struct {
	m      *mid.Middleware
	source string
	sink   string
}

func NewFilter(m *mid.Middleware, source, sink string) *Filter {
	return &Filter{
		m:      m,
		source: source,
		sink:   sink,
	}
}

func (f *Filter) Run(ctx context.Context) error {
	prices, avg := make(map[string][]float32), 0.0
	ch, err := f.m.ConsumeWithContext(ctx, f.source)
	if err != nil {
		return err
	}
loop:
	for {
		var data mid.AvgFilterData
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case msg, more := <-ch:
			if !more {
				return context.Cause(ctx)
			}
			if mid.IsAvgPriceMessage(msg) {
				avg = mid.AvgUnmarshal(msg)
				break loop
			}
			data, err = mid.AvgFilterUnmarshal(msg)
			if err != nil {
				return err
			}
		}
		key := data.Origin + "." + data.Destination
		prices[key] = append(prices[key], data.Price)
	}
	for k, a := range aggregate(prices, avg) {
		origin, destination, _ := strings.Cut(k, ".")
		v := mid.ResultQ4{
			Origin:      origin,
			Destination: destination,
			Avg:         a.Sum / float32(a.Count),
			Max:         a.Max,
		}
		err := f.m.PublishWithContext(ctx, "", f.sink, mid.Q4Marshal(v))
		if err != nil {
			return err
		}
	}
	return nil
}

func aggregate(prices map[string][]float32, avg float32) map[string]resultAggregator {
	acc := make(map[string]resultAggregator)
	for _, arr := range prices {
		for k, v := range arr {
			if avg < v {
				a := acc[k]
				a.Count++
				a.Sum += v
				a.Max = max(a.Max, v)
				acc[k] = a
			}
		}
	}
	return acc
}

type resultAggregator struct {
	Count uint
	Sum   float32
	Max   float32
}
