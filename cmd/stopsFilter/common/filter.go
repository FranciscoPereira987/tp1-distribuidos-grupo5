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

type FastestFlightsMap map[string]map[string][]mid.StopsFilterData

func updateFastest(fastest FastestFlightsMap, data mid.StopsFilterData) {
	if destinyMap, ok := fastest[data.Origin]; !ok {
		tmp := [2]mid.StopsFilterData{data}
		fastest[data.Origin] = map[string][]mid.StopsFilterData{
			data.Destiny: tmp[:1],
		}
	} else if fast, ok := destinyMap[data.Destiny]; !ok {
		tmp := [2]mid.StopsFilterData{data}
		destinyMap[data.Destiny] = tmp[:1]
	} else if data.Duration < fast[0].Duration {
		append(fast[:0], data, fast[0])
	} else if len(fast) == 1 || data.Duration < fast[1].Duration {
		append(fast[:1], data)
	}
}

func (f *Filter) Run(ctx context.Context) error {
	fastest := make(FastestFlightsMap)
	ch, err := f.m.ConsumeWithContext(ctx, f.source)
	if err != nil {
		return err
	}
loop:
	for {
		var data mid.StopsFilterData
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case msg, more := <-ch:
			if !more {
				break loop
			}
			data, err = mid.StopsFilterUnmarshal(msg)
			if err != nil {
				return err
			}
		}
		if strings.Count(data.Stops, stopsSep) >= 3 {
			err := f.m.PublishWithContext(ctx, "", f.sink, mid.Q1Marshal(data))
			if err != nil {
				return err
			}
			updateFastest(fastest, data)
		}
	}
	for _, destinyMap := range fastest {
		for _, arr := range destinyMap {
			for _, v := range arr {
				err := f.m.PublishWithContext(ctx, "", f.sink, mid.Q3Marshal(v))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
