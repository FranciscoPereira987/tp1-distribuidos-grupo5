package typing

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"strconv"
)

const resultQ3Field = "3"

var ResultQ3Header = []string{
	resultQ3Field,
	"legId",
	"startingAirport",
	"destinationAirport",
	"travelDuration",
	"segmentsDepartureAirportCode",
}

func ResultQ3Marshal(b *bytes.Buffer, data *FastestFilter) {
	b.WriteByte(Query3Flag)
	b.Write(data.ID[:])
	WriteString(b, data.Origin)
	WriteString(b, data.Destination)
	binary.Write(b, binary.LittleEndian, data.Duration)
	WriteString(b, data.Stops)
}

func ResultQ3Unmarshal(r *bytes.Reader) ([]string, error) {
	record := make([]string, 6)
	record[0] = resultQ3Field

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
		var duration uint32
		err = binary.Read(r, binary.LittleEndian, &duration)
		record[4] = strconv.Itoa(int(duration))
	}

	if err == nil {
		record[5], err = ReadString(r)
	}

	return record, err
}
