package common

import (
	"encoding/binary"
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
)

type Filter struct {
	m      *mid.Middleware
	source string
	sink   string
	dir    string
}

func NewFilter(m *mid.Middleware, source, sink, dir string) (*Filter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Filter{
		m:      m,
		source: source,
		sink:   sink,
		dir:    dir,
	}, nil
}

func (f *Filter) Run(ctx context.Context) error {
	fares, priceSum, count := make(map[string]fareWriter), 0.0, 0
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
			if err := f.appendFare(fares, key, data.Price); err != nil {
				return err
			}
			log.Debugf("new fare for route: %s-%s | fare: %f", data.Origin, data.Destination, data.Price)
		}
	}

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
		var errs []error
		for _, fw := range fares {
			errs = append(errs, fw.Close())
		}
		if err := errors.Join(errs...); err != nil {
			return err
		}
	}

	return f.sendResults(ctx, fares, float32(priceSum/float64(count)))
}

func (f *Filter) sendResults(ctx context.Context, fares map[string]fareWriter, avg float32) error {
	for file := range fares {
		if err := f.aggregate(ctx, file, avg); err != nil {
			return err
		}
	}
	return nil
}

func (f *Filter) aggregate(ctx context.Context, file string, avg float32) error {
	faresFile, err := os.Open(filepath.Join(f.dir, file))
	if err != nil {
		return err
	}
	defer faresFile.Close()

	r := newFareReader(faresFile)
	priceSum, priceMax, count := 0.0, float32(0), 0

	for {
		v, err := r.ReadFloat()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if avg < v {
			priceSum += float64(v)
			count++
			priceMax = max(priceMax, v)
		}
	}
	if count == 0 {
		return nil
	}

	origin, destination, _ := strings.Cut(file, ".")
	v := mid.ResultQ4{
		Origin:      origin,
		Destination: destination,
		AvgPrice:    float32(priceSum / float64(count)),
		MaxPrice:    priceMax,
	}
	log.Infof("route: %s-%s | average: %f | max: %f", origin, destination, v.AvgPrice, priceMax)
	return f.m.PublishWithContext(ctx, f.sink, f.sink, mid.Q4Marshal(v))
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
	errFlush := fw.Flush()
	errClose := fw.file.Close()

	return errors.Join(errFlush, errClose)
}

func (f *Filter) appendFare(fares map[string]fareWriter, file string, fare float32) (err error) {
	if v, ok := fares[file]; ok {
		return v.Write(fare)
	}
	fw, err := newFareWriter(filepath.Join(f.dir, file))
	if err != nil {
		return err
	}

	fares[file] = fw
	return fw.Write(fare)
}

type fareReader struct {
	br   *bufio.Reader
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
