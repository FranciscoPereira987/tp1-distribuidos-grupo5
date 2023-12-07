package common

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/duplicates"
	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

const (
	_ = iota
	WritingResults
	WritingEof
)

var headers = [][]string{
	typing.ResultQ1Header,
	typing.ResultQ2Header,
	typing.ResultQ3Header,
	typing.ResultQ4Header,
}

type Gateway struct {
	m        *mid.Middleware
	workdir  string
	filter   *duplicates.DuplicateFilter
	stateMan *state.StateManager
}

func NewGateway(m *mid.Middleware, workdir string) (*Gateway, error) {
	err := os.MkdirAll(workdir, 0755)
	return &Gateway{
		m,
		workdir,
		duplicates.NewDuplicateFilter(),
		state.NewStateManager(workdir),
	}, err
}

func (g *Gateway) Close() error {
	return state.RemoveWorkdir(g.workdir)
}

func (g *Gateway) Run(ctx context.Context, out io.Writer, ch <-chan mid.Delivery, progress int) (err error) {
	if progress > 0 {
		g.stateMan.RecoverState()
		if err := g.filter.RecoverFromState(g.stateMan); err != nil {
			return err
		}
	}
	w := csv.NewWriter(out)
	records := make([][]string, 0, 64)
	recordsWritten, err := g.stateMan.GetInt("records")
	if err != nil && !errors.Is(err, state.ErrNotFound) {
		return err
	}

	switch step, _ := g.stateMan.GetInt("step"); step {
	case WritingResults:
		goto writingResults
	case WritingEof:
		goto writingEof
	}

	recordsWritten = 4
	g.stateMan.State["records"] = recordsWritten
	g.stateMan.State["step"] = WritingResults
	if err := g.stateMan.Prepare(); err != nil {
		return fmt.Errorf("failed to prepare state to write results: %w", err)
	}
	if err := w.WriteAll(headers[progress:]); err != nil {
		return err
	}
	if err := g.stateMan.Commit(); err != nil {
		return fmt.Errorf("failed to commit state to write results: %w", err)
	}

	progress = recordsWritten
writingResults:
	progress -= recordsWritten
	for d := range ch {
		msg, tag := d.Msg, d.Tag
		r := bytes.NewReader(msg)
		dup, err := g.filter.Update(r)
		if err != nil {
			log.Errorf("action: recovering batch | status: failed | reason: %s", err)
		}
		if dup || err != nil {
			g.m.Ack(tag)
			continue
		}
		for r.Len() > 0 {
			result, err := typing.ResultUnmarshal(r)
			if err != nil {
				return err
			}
			records = append(records, result)
		}
		recordsWritten += len(records)
		g.stateMan.State["records"] = recordsWritten
		if err := g.stateMan.Prepare(); err != nil {
			return err
		}
		if err := w.WriteAll(records[progress:]); err != nil {
			return err
		}
		if err := g.stateMan.Commit(); err != nil {
			return err
		}
		if err := g.m.Ack(tag); err != nil {
			return err
		}
		records = records[:0]
		progress = 0
	}

	g.stateMan.Remove("records")
	g.filter.RemoveFromState(g.stateMan)
	g.stateMan.State["step"] = WritingEof
	if err := g.stateMan.DumpState(); err != nil {
		log.Warnf("failed to dump state for writing EOF: %w", err)
	}

writingEof:
	if err := w.Write(protocol.ResultEOF); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}
