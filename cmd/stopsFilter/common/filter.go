package common

import (
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

type FastestFlightsMap map[string]map[string][]FlightData

func updateFastest(fastest FastestFlightsMap, data FlightData) {
	if destinyMap, ok := fastest[data.Origin]; !ok {
		tmp := [2]FlightData{data}
		fastest[data.Origin] = map[string][]FlightData{
			data.Destiny: tmp[:1],
		}
	} else if fast, ok := destinyMap[data.Destiny]; !ok {
		tmp := [2]FlightData{data}
		destinyMap[data.Destiny] = tmp[:1]
	} else if data.Duration < fast[0].Duration {
		append(fast[:0], data, fast[0])
	} else if len(fast) == 1 || data.Duration < fast[1].Duration {
		append(fast[:1], data)
	}
}

func (f *Filter) Run(ctx context.Context) error {
	fastest := make(FastestFlightsMap)
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
			err := f.m.PublishWithContext(ctx, "", f.sink, ResultQ1Marshal(data))
			if err != nil {
				return err
			}
			updateFastest(fastest, data)
		}
	}
	for _, destinyMap := range fastest {
		for _, arr := range destinyMap {
			for _, v := range arr {
				err := f.m.PublishWithContext(ctx, "", f.sink, ResultQ2Marshal(v))
				if err != nil {
					return err
				}
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

func ResultQ1Marshal(data FlightData) []byte {
	buf := make([]byte, 1, 40)
	buf[0] = mid.Query1Flag

	buf = append(buf, data.ID[:]...)

	buf = mid.AppendString(buf, data.Origin)
	buf = mid.AppendString(buf, data.Destiny)

	var w bytes.Buffer
	_ = binary.Write(w, binary.LittleEndian, data.Price)
	buf = append(buf, w.Bytes()...)

	buf = mid.AppendString(buf, data.Stops)

	return buf
}

func ResultQ2Marshal(data FlightData) []byte {
	buf := make([]byte, 1, 40)
	buf[0] = mid.Query3Flag

	buf = append(buf, data.ID[:]...)

	buf = mid.AppendString(buf, data.Origin)
	buf = mid.AppendString(buf, data.Destiny)

	buf = binary.LittleEndian.AppendUint32(buf, data.Duration)

	buf = mid.AppendString(buf, data.Stops)

	return buf
}
