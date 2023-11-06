package common

import (
	"bytes"
	"context"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware/id"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	log "github.com/sirupsen/logrus"
)

type Filter struct {
	m      *mid.Middleware
	id     string
	sink   string
}

func NewFilter(m *mid.Middleware, id, sink string) *Filter {
	return &Filter{
		m:      m,
		id:     id,
		sink:   sink,
	}
}

type FastestFlightsMap map[string][]typing.FastestFilter

func updateFastest(fastest FastestFlightsMap, data typing.FastestFilter) {
	key := data.Origin + "." + data.Destination
	if fast, ok := fastest[key]; !ok {
		tmp := [2]typing.FastestFilter{data}
		fastest[key] = tmp[:1]
	} else if data.Duration < fast[0].Duration {
		fastest[key] = append(fast[:0], data, fast[0])
	} else if len(fast) == 1 || data.Duration < fast[1].Duration {
		fastest[key] = append(fast[:1], data)
	} else {
		return
	}
	log.Debugf("updated fastest flights for route %s-%s", data.Origin, data.Destination)
}

func (f *Filter) Run(ctx context.Context, ch <-chan []byte) error {
	fastest := make(FastestFlightsMap)

	for msg := range ch {
		data, err := typing.FastestFilterUnmarshal(msg[id.Len:])
		if err != nil {
			return err
		}
		updateFastest(fastest, data)
	}

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
	}

	log.Infof("start publishing results into %q queue", f.sink)
	for _, arr := range fastest {
		for _, v := range arr {
			b := bytes.NewBufferString(f.id)
			typing.ResultQ3Marshal(b, &v)
			err := f.m.Publish(ctx, f.sink, f.sink, b.Bytes())
			if err != nil {
				return err
			}
		}
	}
	log.Infof("finished publishing results into %q queue", f.sink)

	return nil
}
