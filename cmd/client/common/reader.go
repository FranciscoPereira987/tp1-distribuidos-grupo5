package common

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
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
	if err := os.MkdirAll(dir, 0750); err != nil {
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

func (rr *ResultsReader) ReadResults(r io.Reader) error {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Bytes()
		index, record, err := protocol.SplitRecord(line)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		log.Infof("query: %d | value: %s", index, record)
		// Add back the newline removed by the scanner
		record = append(record, '\n')
		_, err = io.Copy(rr.bws[index-1], bytes.NewReader(record))
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return fmt.Errorf("%w reading results", io.ErrUnexpectedEOF)
}
