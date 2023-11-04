package typing

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"strconv"
)

const resultQ1Field = "1"

var ResultQ1Header = []string{
	resultQ1Field,
	"legId",
	"startingAirport",
	"destinationAirport",
	"totalFare",
	"segmentsDepartureAirportCode",
}

func ResultQ1Marshal(b *bytes.Buffer, data *Flight) {
	b.WriteByte(Query1Flag)
	b.Write(data.ID[:])
	WriteString(b, data.Origin)
	WriteString(b, data.Destination)
	binary.Write(b, binary.LittleEndian, data.Fare)
	WriteString(b, data.Stops)
}

func ResultQ1Unmarshal(r *bytes.Reader) ([]string, error) {
	record := make([]string, 6)
	record[0] = resultQ1Field

	var id [16]byte
	_, err := io.ReadFull(r, id[:])
	if err == nil {
		record[1] = hex.EncodeToString(id[:])
	}

	if err == nil {
		record[2], err = ReadString(r)
	}
	if err == nil {
		record[3], err = ReadString(r)
	}

	if err == nil {
		var price float32
		err = binary.Read(r, binary.LittleEndian, &price)
		record[4] = strconv.FormatFloat(float64(price), 'f', 2, 32)
	}

	if err == nil {
		record[5], err = ReadString(r)
	}

	return record, err
}
