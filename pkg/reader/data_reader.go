package reader

import (
	"bufio"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

const (
	ID           = 0
	ORIGIN       = 3
	DESTINATION  = 4
	DURATION     = 6
	FARE         = 12
	DISTANCE     = 14
	STOPS        = 19
	FLIGHTFIELDS = 27
)

var ErrField error = errors.New("invalid number of fields")

type DataReader struct {
	file *os.File
	buf  *bufio.Scanner
}

func NewDataReader(filepath string) (Reader, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	buf := bufio.NewScanner(file)
	buf.Split(bufio.ScanLines)
	if !buf.Scan() {
		return nil, io.ErrUnexpectedEOF
	}
	buf.Bytes()
	return &DataReader{
		file,
		buf,
	}, nil
}
func IntoFlightData(stream string) (data *typing.FlightDataType, err error) {
	data = new(typing.FlightDataType)
	line := strings.Split(stream, ",")
	if len(line) != FLIGHTFIELDS {
		err = ErrField
	}

	if err == nil {
		var id []byte
		id, err = hex.DecodeString(line[ID])
		if err != nil {
			return
		}
		data.Id = [16]byte(id)
		data.Origin = line[ORIGIN]
		data.Destination = line[DESTINATION]
		data.Duration = line[DURATION]
		data.Fare = line[FARE]
		data.Distance = line[DISTANCE]
		data.Stops = line[STOPS]
	}
	return
}
func (reader *DataReader) ReadData() (protocol.Data, error) {
	if !reader.buf.Scan() {
		err := io.EOF
		return nil, err
	}
	line := reader.buf.Text()
	data := typing.NewData(typing.FLIGHT, line)

	return protocol.NewDataMessage(data), nil
}

func (reader *DataReader) Close() error {
	return reader.file.Close()
}
