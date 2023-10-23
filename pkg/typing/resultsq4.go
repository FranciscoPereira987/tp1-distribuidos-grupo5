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

func ResultQ4Marshal(b *bytes.Buffer, data *ResultQ4) {
	b.WriteByte(Query4Flag)
	WriteString(b, data.Origin)
	WriteString(b, data.Destination)
	binary.Write(b, binary.LittleEndian, data.AverageFare)
	binary.Write(b, binary.LittleEndian, data.MaxFare)
}

func ResultQ4Unmarshal(r *bytes.Reader) (record []string, err error) {
	record = make([]string, 5)
	record[0] = "4"

	record[1], err = ReadString(r)
	if err == nil {
		record[2], err = ReadString(r)
	}

	if err == nil {
		var avgFare float32
		err = binary.Read(r, binary.LittleEndian, &avgFare)
		record[3] = strconv.FormatFloat(float64(avgFare), 'f', 2, 32)
	}

	if err == nil {
		var maxFare float32
		err = binary.Read(r, binary.LittleEndian, &maxFare)
		record[4] = strconv.FormatFloat(float64(maxFare), 'f', 2, 32)
	}

	return record, err
}
