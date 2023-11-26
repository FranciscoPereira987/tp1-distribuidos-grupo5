package common

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	log "github.com/sirupsen/logrus"
)

type Filter struct {
	m        *mid.Middleware
	id       string
	sink     string
	workdir  string
	stateMan *state.StateManager
}

func NewFilter(m *mid.Middleware, id, sink, workdir string) (*Filter, error) {
	err := os.MkdirAll(filepath.Join(workdir, "fastest"), 0755)
	return &Filter{
		m,
		id,
		sink,
		workdir,
		state.NewStateManager(filepath.Join(workdir, "fastest", fmt.Sprintf("filter-%s.state", id))),
	}, err
}

func RecoverFromState(m *mid.Middleware, workdir string, stateMan *state.StateManager) (f *Filter) {
	f = new(Filter)
	f.m = m
	f.id = stateMan.GetString("id")
	f.sink = stateMan.GetString("sink")
	f.workdir = workdir
	f.stateMan = stateMan
	return
}

// TODO: Implement
func (f *Filter) Restart(ctx context.Context) error {
	return nil
}

func (f *Filter) StoreState() error {
	f.stateMan.AddToState("id", f.id)
	f.stateMan.AddToState("sink", f.sink)
	return f.stateMan.DumpState()
}

func (f *Filter) Close() error {
	return os.RemoveAll(f.workdir)
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

func (f *Filter) loadFastest() (FastestFlightsMap, error) {
	fastest := make(FastestFlightsMap)

	files, err := os.ReadDir(filepath.Join(f.workdir, "fastest"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if state.IsTmp(file.Name()) {
			os.Remove(filepath.Join(f.workdir, "fastest", file.Name()))
			continue
		}
		if state.IsState(file.Name()) {
			continue
		}
		buf, err := os.ReadFile(filepath.Join(f.workdir, "fastest", file.Name()))
		if err != nil {
			return nil, err
		}

		var fast []typing.FastestFilter
		for r := bytes.NewReader(buf); r.Len() > 0; {
			data, err := typing.FastestFilterUnmarshal(r)
			if err != nil {
				return nil, err
			}
			fast = append(fast, data)
		}
		fastest[file.Name()] = fast
	}

	return fastest, nil
}

func (f *Filter) Run(ctx context.Context, ch <-chan mid.Delivery) error {
	var updated []string
	if err := f.StoreState(); err != nil {
		return err
	}
	fastest, err := f.loadFastest()
	if err != nil {
		return err
	}

	for d := range ch {
		msg, tag := d.Msg, d.Tag
		for r := bytes.NewReader(msg); r.Len() > 0; {
			data, err := typing.FastestFilterUnmarshal(r)
			if err != nil {
				return err
			}
			if key := updateFastest(fastest, data); key != "" {
				updated = append(updated, key)
			}
		}

		for _, key := range updated {
			var b bytes.Buffer
			for _, v := range fastest[key] {
				v.Marshal(&b)
			}
			if err := state.WriteFile(filepath.Join(f.workdir, "fastest", key), b.Bytes()); err != nil {
				return err
			}
		}
		updated = updated[:0]

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
	var bc mid.BasicConfirmer
	i := mid.MaxMessageSize / typing.ResultQ3Size
	b := bytes.NewBufferString(f.id)
	for _, arr := range fastest {
		for _, v := range arr {
			typing.ResultQ3Marshal(b, &v)
			if i--; i <= 0 {
				if err := bc.Publish(ctx, f.m, f.sink, f.sink, b.Bytes()); err != nil {
					return err
				}
				i = mid.MaxMessageSize / typing.ResultQ3Size
				b = bytes.NewBufferString(f.id)
			}
		}
	}
	if i != mid.MaxMessageSize/typing.ResultQ3Size {
		if err := bc.Publish(ctx, f.m, f.sink, f.sink, b.Bytes()); err != nil {
			return err
		}
	}
	log.Infof("finished publishing results into %q queue", f.sink)

	return nil
}
