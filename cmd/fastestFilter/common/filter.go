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
	workerId string
	clientId string
	sink     string
	workdir  string
	stateMan *state.StateManager
}

func NewFilter(m *mid.Middleware, workerId, clientId, sink, workdir string) (*Filter, error) {
	err := os.MkdirAll(filepath.Join(workdir, "fastest"), 0755)
	return &Filter{
		m,
		workerId,
		clientId,
		sink,
		workdir,
		state.NewStateManager(workdir),
	}, err
}

func RecoverFromState(m *mid.Middleware, workerId, clientId, sink, workdir string, stateMan *state.StateManager) (f *Filter) {
	f = new(Filter)
	f.m = m
	f.workerId = workerId
	f.clientId = clientId
	f.sink = sink
	f.workdir = workdir
	f.stateMan = stateMan
	return
}

func (f *Filter) Restart(ctx context.Context, toRestart map[string]*Filter) {
	sent, ok := f.stateMan.State["sent"].([]string)
	if ok {
		log.Info("action: re-start worker | result: re-sending results")
		go func() {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			fastest, err := f.loadFastest()
			if err == nil {
				err = f.SendResults(ctx, fastest, sent)
			}
			if err != nil {
				logrus.Infof("action: re-sending results | status: failed | reason: %s", err)
			}
		}()
	} else {
		log.Info("action: re-start worker | result: add to map")
		toRestart[f.clientId] = f
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
		if file.IsDir() {
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
	f.stateMan.State["sent"] = []string{}
	if err := f.stateMan.Prepare(); err != nil {
		return err
	}
	fastest, err := f.loadFastest()
	if err != nil {
		return err
	}

	for d := range ch {
		updated := make(map[string]bool)
		msg, tag := d.Msg, d.Tag
		for r := bytes.NewReader(msg); r.Len() > 0; {
			data, err := typing.FastestFilterUnmarshal(r)
			if err != nil {
				return err
			}
			if key := updateFastest(fastest, data); key != "" {
				updated[key] = true
			}
		}

		for key := range updated {
			var b bytes.Buffer
			for _, v := range fastest[key] {
				v.Marshal(&b)
			}
			if err := state.WriteFile(filepath.Join(f.workdir, "fastest", key), b.Bytes()); err != nil {
				return err
			}
		}

		if err := f.m.Ack(tag); err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
		if err := f.stateMan.Commit(); err != nil {
			log.Error("action: commit | result: failure | reason:", err)
		}
	}

	return f.SendResults(ctx, fastest, nil)
}

func (f *Filter) SendResults(ctx context.Context, fastest FastestFlightsMap, sent []string) error {

	log.Infof("start publishing results into %q queue", f.sink)
	var bc mid.BasicConfirmer
	i := mid.MaxMessageSize / typing.ResultQ3Size
	b := bytes.NewBufferString(f.clientId)
	for key, arr := range fastest {
		for _, s := range sent {
			if key == s {
				continue
			}
		}
		sent = append(sent, key)
		f.stateMan.State["sent"] = sent
		for _, v := range arr {
			f.marshalResult(b, &v)
			i--
		}
		if i <= 0 {
			if err := f.stateMan.Prepare(); err != nil {
				return err
			}
			if err := bc.Publish(ctx, f.m, "", f.sink, b.Bytes()); err != nil {
				return err
			}
			if err := f.stateMan.Commit(); err != nil {
				return err
			}
			i = mid.MaxMessageSize / typing.ResultQ3Size
			b = bytes.NewBufferString(f.clientId)
		}
	}
	if i != mid.MaxMessageSize/typing.ResultQ3Size {
		if err := f.stateMan.Prepare(); err != nil {
			return err
		}
		if err := bc.Publish(ctx, f.m, "", f.sink, b.Bytes()); err != nil {
			return err
		}
		if err := f.stateMan.Commit(); err != nil {
			return err
		}
	}
	log.Infof("finished publishing results into %q queue", f.sink)
	return nil
}

func (f *Filter) marshalResult(b *bytes.Buffer, v *typing.FastestFilter) {
	if b.Len() == len(f.clientId) {
		h := typing.NewHeader(f.workerId, string(v.ID[:]))
		h.Marshal(b)
	}
	typing.ResultQ3Marshal(b, v)

}
