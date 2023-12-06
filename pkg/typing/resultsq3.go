package typing

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

const resultQ3Field = "3"
const ResultQ3Size = FlagSize + IdSize + OriginSize + DestinationSize + DurationSize + StopsSize

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
		record[4] = StringIso8601(duration)
	}

	if err == nil {
		record[5], err = ReadString(r)
	}

	return record, err
}

func StringIso8601(minutes uint32) string {
	hours := minutes / 60
	minutes -= hours * 60
	days := hours / 24
	hours -= days * 24

	var b strings.Builder
	b.WriteByte('P')
	if days > 0 {
		fmt.Fprintf(&b, "%dD", days)
	}
	b.WriteByte('T')
	if hours > 0 {
		fmt.Fprintf(&b, "%dH", days)
	}
	if minutes > 0 {
		fmt.Fprintf(&b, "%dM", days)
	}

	return b.String()
}
