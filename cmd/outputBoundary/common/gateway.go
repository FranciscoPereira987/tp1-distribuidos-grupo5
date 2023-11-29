package common

import (
	"context"
	"encoding/csv"
	"io"

	// log "github.com/sirupsen/logrus"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/duplicates"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/sirupsen/logrus"
)

type Gateway struct {
	m      *mid.Middleware
	filter *duplicates.DuplicateFilter
}

func NewGateway(m *mid.Middleware) *Gateway {
	return &Gateway{
		m:      m,
		filter: duplicates.NewDuplicateFilter(""),
	}
}

func (g *Gateway) Run(ctx context.Context, out io.Writer, ch <-chan mid.Delivery) (err error) {
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

	for d := range ch {
		msg, tag := d.Msg, d.Tag
		if g.filter.IsDuplicate(msg) {
			g.m.Ack(tag)
			continue
		}
		r, err := g.filter.ChangeLast(msg)
		if err != nil {
			logrus.Errorf("action: recovering batch | status: failed | reason: %s", err)
			g.m.Ack(tag)
			continue
		}
		for r.Len() > 0 {
			result, err := typing.ResultUnmarshal(r)
			if err != nil {
				return err
			}
			if err := w.Write(result); err != nil {
				return err
			}
		}
		if err := g.m.Ack(tag); err != nil {
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
