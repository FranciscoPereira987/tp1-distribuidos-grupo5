package reader

import (
	"encoding/csv"
	"encoding/hex"
	"os"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

const (
	ID          = 1
	ORIGIN      = 4
	DESTINATION = 5
	DURATION    = 7
	FARE        = 13
	DISTANCE    = 15
	STOPS       = 20
)

type DataReader struct {
	file *os.File
	csv  *csv.Reader
}

func NewDataReader(filepath string) (Reader, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	csv := csv.NewReader(file)
	csv.Read()
	return &DataReader{
		file,
		csv,
	}, nil
}

func (reader *DataReader) ReadData() (protocol.Data, error) {
	line, err := reader.csv.Read()
	//log.Printf("line: %s, len: %d, field: %s", line, len(line), line[DISTANCE])
	if err != nil {
		return nil, err
	}
	data := typing.NewFlightData()

	id, err := hex.DecodeString(line[ID])
	if err != nil {
		return nil, err
	}
	data.Id = [16]byte(id)
	data.Origin = line[ORIGIN]
	data.Destination = line[DESTINATION]
	data.Duration = line[DURATION]
	data.Fare = line[FARE]
	data.Distance = line[DISTANCE]
	data.Stops = line[STOPS]
	return protocol.NewDataMessage(data), nil
}

func (reader *DataReader) Close() error {
	return reader.file.Close()
}
