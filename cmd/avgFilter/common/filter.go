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
	prices, priceSum, count := make(map[string][]float32), 0.0, 0
	ch, err := f.m.ConsumeWithContext(ctx, f.source)
	if err != nil {
		return err
	}

	for msg := range ch {
		if mid.IsAvgPriceMessage(msg) {
			priceSubtotal, priceCount, err := mid.AvgPriceUnmarshal(msg)
			if err != nil {
				return err
			}
			priceSum += priceSubtotal
			count += priceCount
		} else {
			data, err := mid.AvgFilterUnmarshal(msg)
			if err != nil {
				return err
			}
			key := data.Origin + "." + data.Destination
			prices[key] = append(prices[key], data.Price)
		}
	}

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
		return f.aggregate(ctx, prices, float32(priceSum/float64(count)))
	}
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
