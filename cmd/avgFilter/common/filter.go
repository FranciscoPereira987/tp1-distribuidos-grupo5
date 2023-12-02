package common

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/duplicates"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	log "github.com/sirupsen/logrus"
)

type Filter struct {
	m        *mid.Middleware
	workerId string
	clientId string
	sink     string
	workdir  string
	filter   *duplicates.DuplicateFilter
	stateMan *state.StateManager
}

func NewFilter(m *mid.Middleware, workerId, clientId, sink, workdir string) (*Filter, error) {
	err := os.MkdirAll(filepath.Join(workdir, "fares"), 0755)
	return &Filter{
		m,
		workerId,
		clientId,
		sink,
		workdir,
		duplicates.NewDuplicateFilter(),
		state.NewStateManager(workdir),
	}, err
}

func RecoverFromState(m *mid.Middleware, workerId, clientId, sink, workdir string, state *state.StateManager) *Filter {
	filter := duplicates.NewDuplicateFilter()
	filter.RecoverFromState(state)
	return &Filter{
		m,
		workerId,
		clientId,
		sink,
		workdir,
		filter,
		state,
	}
}

func (f *Filter) ShouldRestart() bool {
	processed, _ := f.stateMan.State["processed"].(bool)
	return processed
}

func (f *Filter) Restart(ctx context.Context) error {
	log.Info("action: re-start filter | status: sending results")

	fares, fareSum, count, err := f.GetRunVariables()
	if err != nil {
		return err
	}
	return f.sendResults(ctx, fares, float32(fareSum/float64(count)))
}

func (f *Filter) Close() error {
	return state.RemoveWorkdir(f.workdir)
}

func recoverFares(stateMan *state.StateManager, faresDir string) (fares map[string]fareWriter, err error) {
	fares = make(map[string]fareWriter)
	values, ok := stateMan.State["fares"].(map[string]int)
	if !ok {
		stateMan.State["fares"] = make(map[string]int)
	}

	defer func() {
		if err != nil {
			for _, fw := range fares {
				fw.Close()
			}
		}
	}()

	for file, count := range values {
		writer, err := recoverFareWriter(filepath.Join(faresDir, file), count)
		if err != nil {
			return fares, fmt.Errorf("recovering fares for %s: %w", file, err)
		} else {
			fares[file] = writer
		}
	}

	return fares, nil
}

func (f *Filter) GetRunVariables() (map[string]fareWriter, float64, int, error) {

	fareSum, _ := f.stateMan.State["sum"].(float64)
	fareCount, _ := f.stateMan.State["count"].(int)
	fares, err := recoverFares(f.stateMan, filepath.Join(f.workdir, "fares"))

	return fares, fareSum, fareCount, err
}

func (f *Filter) updateFares(sum float64, count int) {
	f.stateMan.State["count"] = count
	f.stateMan.State["sum"] = sum
}

func (f *Filter) Run(ctx context.Context, ch <-chan mid.Delivery) (err error) {
	fares, fareSum, count, err := f.GetRunVariables()
	if err != nil {
		return err
	}
	defer func() {
		var errs []error
		for _, fw := range fares {
			errs = append(errs, fw.Close())
		}
		if err == nil {
			err = errors.Join(errs...)
		}
	}()

	for d := range ch {
		msg, tag := d.Msg, d.Tag
		r := bytes.NewReader(msg)
		dup, err := f.filter.Update(r)
		if err != nil {
			log.Errorf("action: reading_batch | status: failed | reason: %s", err)
		}
		if dup || err != nil {
			f.m.Ack(tag)
			continue
		}
		f.filter.AddToState(f.stateMan)
		for r.Len() > 0 {
			data, err := typing.AverageFilterUnmarshal(r)
			if err != nil {
				return err
			}

			switch v := data.(type) {
			case typing.AverageFare:
				fareSum += v.Sum
				count += int(v.Count)
				log.Infof("updated average fare: %f", fareSum/float64(count))
				f.updateFares(fareSum, count)
			case typing.AverageFilterFlight:
				key := v.Origin + "." + v.Destination
				if err := f.appendFare(fares, key, v.Fare); err != nil {
					return err
				}
				log.Debugf("new fare for route %s-%s: %f", v.Origin, v.Destination, v.Fare)
			}
		}

		for _, fw := range fares {
			if err := fw.Flush(); err != nil {
				return err
			}
		}

		if err := f.stateMan.DumpState(); err != nil {
			return err
		}
		if err := f.m.Ack(tag); err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
		f.stateMan.State["processed"] = true
		f.filter.RemoveFromState(f.stateMan)
		if err := f.stateMan.DumpState(); err != nil {
			return err
		}
	}

	return f.sendResults(ctx, fares, float32(fareSum/float64(count)))
}

func (f *Filter) sendResults(ctx context.Context, fares map[string]fareWriter, avg float32) error {
	log.Infof("start publishing results into %q queue", f.sink)
	var bc mid.BasicConfirmer
	i := mid.MaxMessageSize / typing.ResultQ4Size
	b := bytes.NewBufferString(f.clientId)
	keys := make([]string, 0, len(fares))
	for key := range fares {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, file := range keys {
		fw := fares[file]
		if newResult, err := f.aggregate(ctx, b, file, fw, avg); err != nil {
			return err
		} else if newResult {
			delete(f.stateMan.State["fares"].(map[string]int), file)
			if i--; i <= 0 {
				if err := f.stateMan.Prepare(); err != nil {
					return err
				}
				if err := bc.Publish(ctx, f.m, "", f.sink, b.Bytes()); err != nil {
					return err
				}
				if err := f.stateMan.Commit(); err != nil {
					// This may result in duplicated messages
					log.Errorf("action: dump_state | status: failure | reason: %s", err)
				}
				i = mid.MaxMessageSize / typing.ResultQ4Size
				b = bytes.NewBufferString(f.clientId)
			}
		}
	}
	if i != mid.MaxMessageSize/typing.ResultQ4Size {
		if err := f.stateMan.Prepare(); err != nil {
			return err
		}
		if err := bc.Publish(ctx, f.m, "", f.sink, b.Bytes()); err != nil {
			return err
		}
		if err := f.stateMan.Commit(); err != nil {
			log.Errorf("action: dump_state | status: failure | reason: %s", err)
		}
	}
	log.Infof("finished publishing results into %q queue", f.sink)

	return nil
}

func (f *Filter) aggregate(ctx context.Context, b *bytes.Buffer, file string, fw fareWriter, avg float32) (bool, error) {
	fareSum, fareMax, count := 0.0, float32(0), 0
	r, err := fw.Reader()
	if err != nil {
		return false, err
	}

	for {
		v, err := r.ReadFloat()
		if err != nil {
			if err == io.EOF {
				break
			}
			return false, err
		}
		if avg < v {
			fareSum += float64(v)
			count++
			fareMax = max(fareMax, v)
		}
	}
	origin, destination, _ := strings.Cut(file, ".")
	if count == 0 {
		log.Debugf("no flights with above average fare for route %s-%s", origin, destination)
		return false, nil
	}

	v := typing.ResultQ4{
		Origin:      origin,
		Destination: destination,
		AverageFare: float32(fareSum / float64(count)),
		MaxFare:     fareMax,
	}
	if b.Len() == len(f.clientId) {
		h := typing.NewHeader(f.workerId, file)
		h.Marshal(b)
	}
	typing.ResultQ4Marshal(b, &v)
	log.Debugf("route: %s-%s | average: %f | max: %f", origin, destination, v.AverageFare, fareMax)
	return true, nil
}

type fareWriter struct {
	bw    *bufio.Writer
	file  *os.File
	Fares int
}

func newFareWriter(path string) (fw fareWriter, err error) {
	fw.file, err = os.Create(path)
	fw.bw = bufio.NewWriter(fw.file)

	return fw, err
}

func recoverFareWriter(path string, fares int) (fw fareWriter, err error) {
	fw.file, err = os.OpenFile(path, os.O_RDWR, 0644)
	fw.bw = bufio.NewWriter(fw.file)
	fw.Fares = fares
	if err == nil {
		if _, err = fw.file.Seek(int64(fares) * 4, 0); err != nil {
			fw.file.Close()
		}
	}

	return fw, err
}

func (fw *fareWriter) Write(fare float32) error {
	fw.Fares++
	return binary.Write(fw.bw, binary.LittleEndian, fare)
}

func (fw *fareWriter) Flush() error {
	return fw.bw.Flush()
}

func (fw *fareWriter) Close() error {
	return fw.file.Close()
}

func (f *Filter) appendFare(fares map[string]fareWriter, file string, fare float32) error {
	if v, ok := fares[file]; ok {
		err := v.Write(fare)
		if err == nil {
			f.stateMan.State["fares"].(map[string]int)[file] = v.Fares
		}
		return err
	}
	fw, err := newFareWriter(filepath.Join(f.workdir, "fares", file))
	if err != nil {
		return err
	}

	fares[file] = fw
	err = fw.Write(fare)
	if err == nil {
		f.stateMan.State["fares"].(map[string]int)[file] = fw.Fares
	}
	return err
}

func (fw fareWriter) Reader() (fareReader, error) {
	_, err := fw.file.Seek(0, 0)
	return newFareReader(fw.file), err
}

type fareReader struct {
	br *bufio.Reader
}

func newFareReader(r io.Reader) fareReader {
	return fareReader{
		bufio.NewReader(r),
	}
}

func (fr *fareReader) ReadFloat() (fare float32, err error) {
	err = binary.Read(fr.br, binary.LittleEndian, &fare)
	return fare, err
}
