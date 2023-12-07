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

func (dw DataWriter) WriteData(data io.Writer, offset int64) error {
	bw := bufio.NewWriter(data)
	if offset >= -1 {
		goto writeFlights
	}

	if err := protocol.WriteFile(bw, dw.coords, -1); err != nil {
		return err
	}

writeFlights:
	if err := protocol.WriteFile(bw, dw.flights, offset); err != nil {
		return err
	}

	return bw.Flush()
}
