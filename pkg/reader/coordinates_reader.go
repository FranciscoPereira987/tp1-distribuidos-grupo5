package reader

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

var (
	AIRPORT_NAME = 0
	LAT_CODE     = 5
	LON_CODE     = 6

	COORDFIELDS = 11
)

type CoordinatesReader struct {
	file *os.File
	buf  *bufio.Scanner
}

func NewCoordinatesReader(filepath string) (Reader, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewScanner(file)
	reader.Split(bufio.ScanLines)
	if !reader.Scan() {
		return nil, io.ErrUnexpectedEOF
	}
	reader.Bytes()
	return &CoordinatesReader{
		file,
		reader,
	}, nil
}

func IntoCoordinates(stream string) (data *distance.CoordWrapper, err error) {
	data = new(distance.CoordWrapper)
	recovered := strings.Split(stream, ";")
	airport_code := recovered[AIRPORT_NAME]
	lat, err := strconv.ParseFloat(recovered[LAT_CODE], 64)
	if err != nil {
		return nil, err
	}
	lon, err := strconv.ParseFloat(recovered[LON_CODE], 64)
	if err != nil {
		return nil, err
	}

	return &distance.CoordWrapper{
		Value: middleware.CoordinatesData{
			Latitude:    lat,
			Longitud:    lon,
			AirportCode: airport_code,
		},
	}, nil
}

func (reader *CoordinatesReader) ReadData() (protocol.Data, error) {
	if !reader.buf.Scan() {
		err := reader.buf.Err()
		if err == nil {
			err = io.EOF
		}
		return nil, err
	}
	line := reader.buf.Text()
	data := typing.NewData(typing.COORDINATES, line)
	return protocol.NewDataMessage(data), nil
}

func (reader *CoordinatesReader) Close() error {
	return reader.file.Close()
}
