package common

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/sirupsen/logrus"
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
		state.NewStateManager(workdir),
	}, err
}

func RecoverFromState(m *mid.Middleware, id, sink, workdir string, stateMan *state.StateManager) (f *Filter) {
	f = new(Filter)
	f.m = m
	f.id = id
	f.sink = sink
	f.workdir = workdir
	f.stateMan = stateMan
	return
}

func (f *Filter) recoverSended() map[string]bool {
	mapped := make(map[string]bool)
	value, ok := f.stateMan.Get("sended").(map[any]any)
	if ok {
		for key, val := range value {
			mapped[key.(string)] = val.(bool)
		}
	}
	return mapped
}

// TODO: Implement
func (f *Filter) Restart(ctx context.Context, toRestart map[string]*Filter) {
	processed := f.stateMan.Get("processed").(bool)
	if processed {
		log.Info("action: re-start worker | result: re-sending results")
		go func() {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			value := f.recoverSended()
			fastest, err := f.loadFastest()
			if err == nil {
				err = f.SendResults(ctx, fastest, value)
			}
			if err != nil {
				logrus.Infof("action: re-sending results | status: failed | reason: %s", err)
			}
		}()
	} else {
		log.Info("action: re-start worker | result: add to map")
		toRestart[f.id] = f
	}
}

func (f *Filter) StoreState() error {
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

	processed := make(map[string]bool)
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
		f.stateMan.AddToState("processed", true)
		for key := range fastest {
			processed[key] = false
		}
		f.stateMan.AddToState("sended", processed)
		if err := f.StoreState(); err != nil {
			return err
		}
	}

	return f.SendResults(ctx, fastest, processed)
}

func (f *Filter) SendResults(ctx context.Context, fastest FastestFlightsMap, sended map[string]bool) error {

	log.Infof("start publishing results into %q queue", f.sink)
	var bc mid.BasicConfirmer
	i := mid.MaxMessageSize / typing.ResultQ3Size
	b := bytes.NewBufferString(f.id)
	for key, arr := range fastest {
		if processed, _ := sended[key]; processed {
			continue
		}
		for _, v := range arr {
			typing.ResultQ3Marshal(b, &v)
			if i--; i <= 0 {
				if err := bc.Publish(ctx, f.m, "", f.sink, b.Bytes()); err != nil {
					return err
				}
				f.stateMan.AddToState("sended", sended)
				if err := f.StoreState(); err != nil {
					return err
				}
				i = mid.MaxMessageSize / typing.ResultQ3Size
				b = bytes.NewBufferString(f.id)
			}
		}
		delete(fastest, key)
		sended[key] = true
	}
	if i != mid.MaxMessageSize/typing.ResultQ3Size {
		if err := bc.Publish(ctx, f.m, "", f.sink, b.Bytes()); err != nil {
			return err
		}
	}
	log.Infof("finished publishing results into %q queue", f.sink)
	return nil
}
