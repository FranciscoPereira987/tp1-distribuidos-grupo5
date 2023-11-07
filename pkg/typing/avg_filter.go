package typing

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	averageFareFlag = iota
	averageFilterFlag
)

type AverageFilterFlight struct {
	Origin      string
	Destination string
	Fare        float32
}

type AverageFare struct {
	Sum   float64
	Count uint64
}

func AverageFilterMarshal(b *bytes.Buffer, data *Flight) {
	b.WriteByte(averageFilterFlag)
	WriteString(b, data.Origin)
	WriteString(b, data.Destination)
	binary.Write(b, binary.LittleEndian, data.Fare)
}

func AverageFareMarshal(b *bytes.Buffer, fareSum float64, fareCount int) {
	b.WriteByte(averageFareFlag)
	binary.Write(b, binary.LittleEndian, fareSum)
	binary.Write(b, binary.LittleEndian, uint64(fareCount))
}

func AverageFilterUnmarshal(r *bytes.Reader) (data any, err error) {
	flag, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch flag {
	case averageFareFlag:
		return avgFareUnmarshal(r)
	case averageFilterFlag:
		return avgFilterUnmarshal(r)
	default:
		return nil, fmt.Errorf("Unknown format specifier: %d", flag)
	}
}

func avgFilterUnmarshal(r *bytes.Reader) (data AverageFilterFlight, err error) {
	data.Origin, err = ReadString(r)
	if err == nil {
		data.Destination, err = ReadString(r)
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &data.Fare)
	}

	return data, err
}

func avgFareUnmarshal(r *bytes.Reader) (data AverageFare, err error) {
	err = binary.Read(r, binary.LittleEndian, &data.Sum)
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &data.Count)
	}

	return data, err
}
