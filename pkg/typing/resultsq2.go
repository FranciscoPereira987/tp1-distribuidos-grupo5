package typing

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"strconv"
)

const resultQ2Field = "2"

var ResultQ2Header = []string{
	resultQ2Field,
	"legId",
	"startingAirport",
	"destinationAirport",
	"totalTravelDistance",
}

func ResultQ2Marshal(b *bytes.Buffer, data *DistanceFilter) {
	b.WriteByte(Query2Flag)
	b.Write(data.ID[:])
	WriteString(b, data.Origin)
	WriteString(b, data.Destination)
	binary.Write(b, binary.LittleEndian, data.Distance)
}

func ResultQ2Unmarshal(r *bytes.Reader) ([]string, error) {
	record := make([]string, 5)
	record[0] = resultQ2Field

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
		var distance uint32
		err = binary.Read(r, binary.LittleEndian, &distance)
		record[4] = strconv.Itoa(int(distance))
	}

	return record, err
}
