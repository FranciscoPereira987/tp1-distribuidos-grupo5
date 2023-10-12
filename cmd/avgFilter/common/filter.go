package common

import (
	"context"
	"strings"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
)

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
			if !more {
				break loop
			}
			if mid.IsAvgPriceMessage(msg) {
				avg, err = mid.AvgUnmarshal(msg)
				if err != nil {
					return err
				}
			} else {
				data, err = mid.AvgFilterUnmarshal(msg)
				if err != nil {
					return err
				}
				key := data.Origin + "." + data.Destination
				prices[key] = append(prices[key], data.Price)
			}
		}
	}
	return f.aggregate(ctx, prices, avg)
}

func (f *Filter) aggregate(ctx context.Context, prices map[string][]float32, avg float32) error {
	for k, arr := range prices {
		priceSum, priceMax, count := 0.0, float32(0), 0

		for _, v := range arr {
			if avg < v {
				priceSum += float64(v)
				count++
				priceMax = max(priceMax, v)
			}
		}
		if count == 0 {
			continue
		}

		origin, destination, _ := strings.Cut(k, ".")
		v := mid.ResultQ4{
			Origin:      origin,
			Destination: destination,
			AvgPrice:    float32(priceSum / float64(count)),
			MaxPrice:    priceMax,
		}
		err := f.m.PublishWithContext(ctx, f.sink, f.sink, mid.Q4Marshal(v))
		if err != nil {
			return err
		}
	}
	return nil
}
