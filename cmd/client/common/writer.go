package common

import (
	"bufio"
	"io"
	"os"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
)

type DataWriter struct {
	coords, flights *os.File
}

func NewDataWriter(coords, flights *os.File) DataWriter {
	return DataWriter{
		coords:  coords,
		flights: flights,
	}
}

func (dw DataWriter) WriteData(data io.Writer) error {
	bw := bufio.NewWriter(data)

	if err := protocol.WriteFile(bw, dw.coords); err != nil {
		return err
	}
	if err := protocol.WriteFile(bw, dw.flights); err != nil {
		return err
	}

	return bw.Flush()
}
