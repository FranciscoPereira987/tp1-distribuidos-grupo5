package common

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	log "github.com/sirupsen/logrus"
)

var ErrResultFiles = errors.New("need 4 result files")

type ResultsReader struct {
	bws   []*bufio.Writer
	files []*os.File
}

func NewResultsReader(dir string, files []string) (*ResultsReader, error) {
	if len(files) != 4 {
		return nil, fmt.Errorf("%w: have %v", ErrResultFiles, files)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	var rr ResultsReader
	for _, file := range files {
		f, err := os.Create(filepath.Join(dir, file))
		if err != nil {
			rr.Close()
			return nil, err
		}
		rr.bws = append(rr.bws, bufio.NewWriter(f))
		rr.files = append(rr.files, f)
	}

	return &rr, nil
}

func (rr *ResultsReader) Close() error {
	var errs []error
	for _, bw := range rr.bws {
		errs = append(errs, bw.Flush())
	}
	for _, f := range rr.files {
		errs = append(errs, f.Close())
	}
	return errors.Join(errs...)
}

func (rr *ResultsReader) ReadResults(r io.Reader) (progress int, err error) {
	scanner := bufio.NewScanner(r)

	// Create a custom split function by wrapping the existing ScanLines
	// function. Lines must contain a terminating newline.
	split := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		return bufio.ScanLines(data, false)
	}
	scanner.Split(split)

	for scanner.Scan() {
		line := scanner.Bytes()
		index, record, err := protocol.SplitRecord(line)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return progress, err
		}
		log.Infof("query: %d | value: %s", index, record)
		// Add back the newline removed by the scanner
		record = append(record, '\n')
		if _, err := rr.bws[index-1].Write(record); err != nil {
			return progress, err
		}
		progress++
	}

	return progress, fmt.Errorf("%w reading results", io.ErrUnexpectedEOF)
}
