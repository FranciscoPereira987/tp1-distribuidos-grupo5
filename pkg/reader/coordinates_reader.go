package reader

import (
	"encoding/csv"
	"os"
	"strconv"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/distance"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
)

var (
	AIRPORT_NAME = 0
	LAT_CODE     = 5
	LON_CODE     = 6
)

type CoordinatesReader struct {
	file *os.File
	csv  *csv.Reader
}

func NewCoordinatesReader(filepath string) (Reader, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.Read()
	return &CoordinatesReader{
		file,
		reader,
	}, nil
}

func (reader *CoordinatesReader) ReadData() (protocol.Data, error) {
	recovered, err := reader.csv.Read()

	if err != nil {
		return nil, err
	}
	airport_code := recovered[AIRPORT_NAME]
	lat, err := strconv.ParseFloat(recovered[LAT_CODE], 64)
	if err != nil {
		return nil, err
	}
	lon, err := strconv.ParseFloat(recovered[LON_CODE], 64)
	if err != nil {
		return nil, err
	}
	coords := distance.Coordinates{
		Lat: lat,
		Lon: lon,
	}
	return distance.IntoData(coords, airport_code), nil
}

func (reader *CoordinatesReader) Close() error {
	return reader.file.Close()
}
