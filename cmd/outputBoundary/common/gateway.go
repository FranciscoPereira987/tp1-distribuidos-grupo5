package common

import (
	"context"
	"encoding/csv"
	"io"

	// log "github.com/sirupsen/logrus"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

type Gateway struct {
	m      *mid.Middleware
	source string
}

func NewGateway(m *mid.Middleware, source string) *Gateway {
	return &Gateway{
		m:      m,
		source: source,
	}
}

func (g *Gateway) Run(ctx context.Context, out io.Writer) (err error) {
	ch, err := g.m.ConsumeWithContext(ctx, g.source)
	if err != nil {
		return err
	}

	w := csv.NewWriter(out)
	defer func() {
		w.Flush()
		if err == nil {
			err = w.Error()
		}
	}()

	if err := writeHeaders(w); err != nil {
		return err
	}

	for msg := range ch {
		result, err := typing.ResultUnmarshal(msg)
		if err != nil {
			return err
		}
		if err := w.Write(result); err != nil {
			return err
		}
	}

	return w.Write(protocol.ResultEOF)
}

func writeHeaders(w *csv.Writer) error {
	err := w.Write(typing.ResultQ1Header)
	if err == nil {
		w.Write(typing.ResultQ2Header)
	}
	if err == nil {
		w.Write(typing.ResultQ3Header)
	}
	if err == nil {
		w.Write(typing.ResultQ4Header)
	}
	return err
}
