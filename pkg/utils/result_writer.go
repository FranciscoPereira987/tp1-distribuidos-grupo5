package utils

import (
	"encoding/csv"
	"errors"
	"os"
)

type ResultWriter struct {
	files map[byte]*csv.Writer
	fds   []*os.File
}

func openFiles(dir string, files []string) ([]*os.File, error) {
	var fds []*os.File
	for _, file := range files {
		f, err := os.Create(dir + "/" + file)
		if err != nil {
			return fds, err
		}
		fds = append(fds, f)
	}
	return fds, nil
}

func mapWriters(writers []*csv.Writer, results []Numbered) (map[byte]*csv.Writer, error) {
	if len(writers) != len(results) {
		return nil, errors.New("files and results are not equal")
	}
	mapped := make(map[byte]*csv.Writer)
	for index, number := range results {
		mapped[number.Number()] = writers[index]
	}
	return mapped, nil
}

func NewResultWriter(dir string, files []string, results []Numbered) (*ResultWriter, error) {
	fds, err := openFiles(dir, files)
	if err != nil {
		for _, fd := range fds {
			fd.Close()
		}
		return nil, err
	}
	var writers []*csv.Writer
	for _, fd := range fds {
		writers = append(writers, csv.NewWriter(fd))
	}
	mapped, err := mapWriters(writers, results)
	if err != nil {
		for _, fd := range fds {
			fd.Close()
		}
		return nil, err
	}
	return &ResultWriter{
		files: mapped,
		fds:   fds,
	}, nil
}

func (writer *ResultWriter) WriteInto(into Numbered, record []string) error {
	csvWriter, ok := writer.files[into.Number()]
	csvWriter.Flush()
	if !ok {
		return errors.New("invalid result")
	}
	return csvWriter.Write(record)
}

func (writer *ResultWriter) Close() {
	for _, fd := range writer.fds {
		fd.Close()
	}
}
