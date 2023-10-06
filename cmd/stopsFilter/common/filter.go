package common

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	// log "github.com/sirupsen/logrus"
)

const stopsSep = "||"

type Filter struct {
	m       *mid.Middleware
	source  string
	resKey  string
	forward string
}

func NewFilter(m *mid.Middleware, source, resKey, forward string) *Filter {
	return &Filter{
		m:       m,
		source:  source,
		resKey:  resKey,
		forward: forward,
	}
}

func (f *Filter) Run(ctx context.Context) error {
	ch, err := f.m.ConsumeWithContext(ctx, f.source)
	if err != nil {
		return err
	}
	for msg := range ch {
		data, err := Unmarshal(msg)
		if err != nil {
			return err
		}
		if strings.Count(data.Stops, stopsSep) >= 3 {
			key := mid.KeyFrom(data.Origin, data.Destiny)
			errRes := f.m.PublishWithContext(ctx, "", f.resKey, ResultMarshal(data))
			errFwd := f.m.PublishWithContext(ctx, f.forward, key, ForwardMarshal(data))
			if err := errors.Join(errRes, errFwd); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *Filter) Start(ctx context.Context, sig <-chan os.Signal) error {
	done := make(chan error)

	go func() {
		done <- f.Run(ctx)
	}()

	select {
	case <-ctx.Done():
	case <-sig:
	case err := <-done:
		return err
	}
	f.m.Close()
	return <-done
}

type FlightData struct {
	ID       [16]byte
	Origin   string
	Destiny  string
	Duration uint32
	Price    float32
	Stops    string
}

func Unmarshal([]byte) (data FlightData, err error) {
	_, err = io.ReadFull(r, data.ID[:])
	if err == nil {
		data.Origin, err = mid.ReadString(r)
	}
	if err == nil {
		data.Destiny, err = mid.ReadString(r)
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &(data.Duration))
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &(data.Price))
	}
	if err == nil {
		data.Stops, err = mid.ReadString(r)
	}

	return data, err
}

func ResultMarshal(data FlightData) []byte {
	buf := make([]byte, 1)
	buf[0] = mid.Query1Flag

	buf = append(buf, data.ID[:]...)

	buf = mid.AppendString(buf, data.Origin)
	buf = mid.AppendString(buf, data.Destiny)

	var w bytes.Buffer
	_ = binary.Write(w, binary.LittleEndian, data.Price)
	buf = append(buf, w.Bytes()...)

	buf = binary.AppendUvarint(buf, len(data.Stops))
	buf = append(buf, data.Stops...)

	return buf
}
