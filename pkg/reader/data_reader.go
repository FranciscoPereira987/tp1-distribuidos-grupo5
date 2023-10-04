package reader

import (
	"encoding/csv"
	"errors"
	"os"
	"regexp"
	"strconv"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
)

var (
	ID           = 1
	ORIGIN       = 4
	DESTINATION  = 5
	DURATION     = 7
	FARE         = 13
	DISTANCE     = 15
	STOPS        = 20
	DURATION_EXP = "[0-9]+"
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

func ParseDuration(duration string) (int, error) {
	exp, err := regexp.Compile(DURATION_EXP)
	if err != nil {
		return 0, err
	}
	values := exp.FindAllString(duration, 2)
	if len(values) < 1 {
		return 0, errors.New("invalid duration string")
	}
	if len(values) < 2 {
		values = append(values, "0")
	}
	hours, _ := strconv.Atoi(values[0])
	minutes, _ := strconv.Atoi(values[1])

	return minutes + hours*60, nil
}

func (reader *DataReader) ReadData() (protocol.Data, error) {
	line, err := reader.csv.Read()
	//log.Printf("line: %s, len: %d, field: %s", line, len(line), line[DISTANCE])
	if err != nil {
		return nil, err
	}
	fare, err := strconv.ParseFloat(line[FARE], 64)
	if err != nil {
		return nil, err
	}
	distance, err := strconv.ParseFloat(line[DISTANCE], 64)
	if err != nil {
		return nil, err
	}
	duration, err := ParseDuration(line[DURATION])
	if err != nil {
		return nil, err
	}
	data, err := NewFlightDataType(line[ID], line[ORIGIN], line[DESTINATION], duration,
		fare, int(distance), line[STOPS])
	if err != nil {
		return nil, err
	}
	return protocol.NewDataMessage(data), nil
}

func (reader *DataReader) Close() error {
	return reader.file.Close()
}
