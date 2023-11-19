package typing

import (
	"bytes"
	"encoding/binary"
	"strconv"
)

type ResultQ4 struct {
	Origin      string
	Destination string
	AverageFare float32
	MaxFare     float32
}

const resultQ4Field = "4"
const ResultQ4Size = FlagSize + OriginSize + DestinationSize + FareSize + FareSize

var ResultQ4Header = []string{
	resultQ4Field,
	"startingAirport",
	"destinationAirport",
	"averageFare",
	"maxFare",
}

func ResultQ4Marshal(b *bytes.Buffer, data *ResultQ4) {
	b.WriteByte(Query4Flag)
	WriteString(b, data.Origin)
	WriteString(b, data.Destination)
	binary.Write(b, binary.LittleEndian, data.AverageFare)
	binary.Write(b, binary.LittleEndian, data.MaxFare)
}

func ResultQ4Unmarshal(r *bytes.Reader) (record []string, err error) {
	record = make([]string, 5)
	record[0] = resultQ4Field

	record[1], err = ReadString(r)
	if err == nil {
		record[2], err = ReadString(r)
	}

	if err == nil {
		var avgFare float32
		err = binary.Read(r, binary.LittleEndian, &avgFare)
		// Average may have extra decimal digits, return exact value
		record[3] = strconv.FormatFloat(float64(avgFare), 'f', -1, 32)
	}

	if err == nil {
		var maxFare float32
		err = binary.Read(r, binary.LittleEndian, &maxFare)
		record[4] = strconv.FormatFloat(float64(maxFare), 'f', 2, 32)
	}

	return record, err
}
