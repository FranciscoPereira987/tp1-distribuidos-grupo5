package common

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/duplicates"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	log "github.com/sirupsen/logrus"
)

type Filter struct {
	m       *mid.Middleware
	id      string
	sink    string
	workdir string
	filter  *duplicates.DuplicateFilter
}

func NewFilter(m *mid.Middleware, id, sink, workdir string) (*Filter, error) {
	err := os.MkdirAll(filepath.Join(workdir, "fares"), 0755)
	return &Filter{
		m,
		id,
		sink,
		workdir,
		duplicates.NewDuplicateFilter(nil),
	}, err
}

func (f *Filter) Close() error {
	return os.RemoveAll(f.workdir)
}

func (f *Filter) Run(ctx context.Context, ch <-chan mid.Delivery) (err error) {
	fares, fareSum, count := make(map[string]fareWriter), 0.0, 0
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
		if f.filter.IsDuplicate(msg) {
			f.m.Ack(tag)
			continue
		}
		for r := bytes.NewReader(msg); r.Len() > 0; {
			data, err := typing.AverageFilterUnmarshal(r)
			if err != nil {
				return err
			}

			switch v := data.(type) {
			case typing.AverageFare:
				fareSum += v.Sum
				count += int(v.Count)
				log.Infof("updated average fare: %f", fareSum/float64(count))
			case typing.AverageFilterFlight:
				key := v.Origin + "." + v.Destination
				if err := f.appendFare(fares, key, v.Fare); err != nil {
					return err
				}
				log.Debugf("new fare for route %s-%s: %f", v.Origin, v.Destination, v.Fare)
			}
		}
		f.filter.ChangeLast(msg)
		// TODO: store state
		if err := f.m.Ack(tag); err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
		var errs []error
		for _, fw := range fares {
			errs = append(errs, fw.Flush())
		}
		if err := errors.Join(errs...); err != nil {
			return err
		}
	}

	return f.sendResults(ctx, fares, float32(fareSum/float64(count)))
}

func (f *Filter) sendResults(ctx context.Context, fares map[string]fareWriter, avg float32) error {
	log.Infof("start publishing results into %q queue", f.sink)
	var bc mid.BasicConfirmer
	i := mid.MaxMessageSize / typing.ResultQ4Size
	b := bytes.NewBufferString(f.id)
	for file := range fares {
		if newResult, err := f.aggregate(ctx, b, file, avg); err != nil {
			return err
		} else if newResult {
			if i--; i <= 0 {
				if err := bc.Publish(ctx, f.m, f.sink, f.sink, b.Bytes()); err != nil {
					return err
				}
				i = mid.MaxMessageSize / typing.ResultQ4Size
				b = bytes.NewBufferString(f.id)
			}
		}
	}
	if i != mid.MaxMessageSize/typing.ResultQ4Size {
		if err := bc.Publish(ctx, f.m, f.sink, f.sink, b.Bytes()); err != nil {
			return err
		}
	}
	log.Infof("finished publishing results into %q queue", f.sink)

	return nil
}

func (f *Filter) aggregate(ctx context.Context, b *bytes.Buffer, file string, avg float32) (bool, error) {
	faresFile, err := os.Open(filepath.Join(f.workdir, "fares", file))
	if err != nil {
		return false, err
	}
	defer faresFile.Close()

	r := newFareReader(faresFile)
	fareSum, fareMax, count := 0.0, float32(0), 0

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
	typing.ResultQ4Marshal(b, &v)
	log.Debugf("route: %s-%s | average: %f | max: %f", origin, destination, v.AverageFare, fareMax)
	return true, nil
}

type fareWriter struct {
	bw   *bufio.Writer
	file *os.File
}

func newFareWriter(path string) (fw fareWriter, err error) {
	fw.file, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	fw.bw = bufio.NewWriter(fw.file)

	return fw, err
}

func (fw *fareWriter) Write(fare float32) error {
	return binary.Write(fw.bw, binary.LittleEndian, fare)
}

func (fw *fareWriter) Flush() error {
	return fw.bw.Flush()
}

func (fw *fareWriter) Close() error {
	return fw.file.Close()
}

func (f *Filter) appendFare(fares map[string]fareWriter, file string, fare float32) (err error) {
	if v, ok := fares[file]; ok {
		return v.Write(fare)
	}
	fw, err := newFareWriter(filepath.Join(f.workdir, "fares", file))
	if err != nil {
		return err
	}

	fares[file] = fw
	return fw.Write(fare)
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
