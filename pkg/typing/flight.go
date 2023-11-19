package typing

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

const FlightSize = IdSize + OriginSize + DestinationSize + DurationSize + FareSize + DistanceSize + StopsSize

var FlightFields = []string{
	"legId",
	"startingAirport",
	"destinationAirport",
	"travelDuration",
	"totalFare",
	"totalTravelDistance",
	"segmentsDepartureAirportCode",
}

var DurationRegexp = regexp.MustCompile(`P(\d+D)?T?(\d+H)?(\d+M)?`)

var (
	ErrInvalidDuration = errors.New("invalid duration format")
	ErrMissingDistance = errors.New("missing field 'totalTravelDistance'")
)

type Flight struct {
	ID          [16]byte
	Origin      string
	Destination string
	Duration    uint32
	Fare        float32
	Distance    uint32
	Stops       string
}

const (
	IdSize          = 16
	OriginSize      = 4
	DestinationSize = 4
	DurationSize    = 4
	FareSize        = 4
	DistanceSize    = 4
	StopsSize       = 24
)

func FlightMarshal(b *bytes.Buffer, record []string, indices []int) error {
	id, err := hex.DecodeString(record[indices[0]])
	if err != nil {
		return err
	}
	duration, err := ParseDuration(record[indices[3]])
	if err != nil {
		return err
	}
	fare, err := strconv.ParseFloat(record[indices[4]], 32)
	if err != nil {
		return err
	}
	// maybe None, ignore error
	distance, err := strconv.Atoi(record[indices[5]])
	if err != nil && record[indices[5]] == "" {
		err = ErrMissingDistance
	}

	b.Write(id)
	WriteString(b, record[indices[1]])
	WriteString(b, record[indices[2]])
	binary.Write(b, binary.LittleEndian, uint32(duration))
	binary.Write(b, binary.LittleEndian, float32(fare))
	binary.Write(b, binary.LittleEndian, uint32(distance))
	WriteString(b, record[indices[6]])

	return err
}

func FlightUnmarshal(r *bytes.Reader) (data Flight, err error) {
	_, err = io.ReadFull(r, data.ID[:])

	if err == nil {
		data.Origin, err = ReadString(r)
	}
	if err == nil {
		data.Destination, err = ReadString(r)
	}

	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &data.Duration)
	}

	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &data.Fare)
	}

	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &data.Distance)
	}

	if err == nil {
		data.Stops, err = ReadString(r)
	}

	return data, err
}

func ParseDuration(duration string) (minutes int, err error) {
	values := DurationRegexp.FindStringSubmatch(duration)
	if len(values) == 0 {
		return 0, fmt.Errorf("%w: '%s'", ErrInvalidDuration, duration)
	}

	days, hours, minutes := 0, 0, 0
	if values[1] != "" {
		days, err = strconv.Atoi(values[1][:len(values[1])-1])
	}
	if err == nil && values[2] != "" {
		hours, err = strconv.Atoi(values[2][:len(values[2])-1])
	}
	if err == nil && values[3] != "" {
		minutes, err = strconv.Atoi(values[3][:len(values[3])-1])
	}

	return days*24*60 + hours*60 + minutes, err
}
