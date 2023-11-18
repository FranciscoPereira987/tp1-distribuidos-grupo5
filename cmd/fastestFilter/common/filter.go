package common

import (
	"bytes"
	"context"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	log "github.com/sirupsen/logrus"
)

type Filter struct {
	m    *mid.Middleware
	id   string
	sink string
}

func NewFilter(m *mid.Middleware, id, sink string) *Filter {
	return &Filter{
		m:    m,
		id:   id,
		sink: sink,
	}
}

type FastestFlightsMap map[string][]typing.FastestFilter

func updateFastest(fastest FastestFlightsMap, data typing.FastestFilter) string {
	key := data.Origin + "." + data.Destination
	if fast, ok := fastest[key]; !ok {
		tmp := [2]typing.FastestFilter{data}
		fastest[key] = tmp[:1]
	} else if data.Duration < fast[0].Duration {
		fastest[key] = append(fast[:0], data, fast[0])
	} else if (len(fast) == 1 || data.Duration < fast[1].Duration) && (data.ID != fast[0].ID) {
		fastest[key] = append(fast[:1], data)
	} else {
		return ""
	}
	log.Debugf("updated fastest flights for route %s-%s", data.Origin, data.Destination)
	return key
}

func (f *Filter) Run(ctx context.Context, ch <-chan mid.Delivery) error {
	fastest := make(FastestFlightsMap)

	for d := range ch {
		msg, tag := d.Msg, d.Tag
		updated := make(map[string]struct{})
		for r := bytes.NewReader(msg); r.Len() > 0; {
			data, err := typing.FastestFilterUnmarshal(r)
			if err != nil {
				return err
			}
			if key := updateFastest(fastest, data); key != "" {
				updated[key] = struct{}{}
			}
		}
		// TODO: store state
		if err := f.m.Ack(tag); err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
	}

	log.Infof("start publishing results into %q queue", f.sink)
	i := mid.MaxMessageSize / typing.ResultQ3Size
	b := bytes.NewBufferString(f.id)
	for _, arr := range fastest {
		for _, v := range arr {
			typing.ResultQ3Marshal(b, &v)
			if i--; i <= 0 {
				if err := f.m.Publish(ctx, f.sink, f.sink, b.Bytes()); err != nil {
					return err
				}
				i = mid.MaxMessageSize / typing.ResultQ3Size
				b = bytes.NewBufferString(f.id)
			}
		}
	}
	if i != mid.MaxMessageSize/typing.ResultQ3Size {
		if err := f.m.Publish(ctx, f.sink, f.sink, b.Bytes()); err != nil {
			return err
		}
	}
	log.Infof("finished publishing results into %q queue", f.sink)

	return nil
}
