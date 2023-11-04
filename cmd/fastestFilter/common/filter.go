package common

import (
	"bytes"
	"context"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	log "github.com/sirupsen/logrus"
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

func (f *Filter) Run(ctx context.Context) error {
	fastest := make(FastestFlightsMap)
	ch, err := f.m.ConsumeWithContext(ctx, f.source)
	if err != nil {
		return err
	}

	log.Infof("start consuming messages from %q queue", f.source)
	for msg := range ch {
		data, err := typing.FastestFilterUnmarshal(msg)
		if err != nil {
			return err
		}
		updateFastest(fastest, data)
	}
	log.Infof("finished consuming messages from %q queue", f.source)

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
	}

	log.Infof("start publishing results into %q queue", f.sink)
	for _, arr := range fastest {
		for _, v := range arr {
			var b bytes.Buffer
			typing.ResultQ3Marshal(&b, &v)
			err := f.m.PublishWithContext(ctx, f.sink, f.sink, b.Bytes())
			if err != nil {
				return err
			}
		}
	}
	log.Infof("finished publishing results into %q queue", f.sink)

	return nil
}
