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
	prices := make(map[string][]float32)
	var avg float32
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
			
			if mid.IsAvgPriceMessage(msg) {
				avg, err = mid.AvgUnmarshal(msg)
				if err != nil {
					return err
				}
				break loop
			}
			data, err = mid.AvgFilterUnmarshal(msg)
			if err != nil {
				return err
			}
			if !more {
				break loop
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
			AvgPrice:    a.Sum / float32(a.Count),
			MaxPrice:    a.Max,
		}
		err := f.m.PublishWithContext(ctx, f.sink, f.sink, mid.Q4Marshal(v))
		if err != nil {
			return err
		}
	}
	return nil
}

func aggregate(prices map[string][]float32, avg float32) map[string]resultAggregator {
	acc := make(map[string]resultAggregator)
	for k, arr := range prices {
		for _, v := range arr {
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
