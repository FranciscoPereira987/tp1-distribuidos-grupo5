package typing

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"strconv"
)

const AirportCoordsSize = CodeSize + LatSize + LonSize

var CoordinatesFields = []string{"Airport Code", "Latitude", "Longitude"}

type AirportCoords struct {
	Code string
	Lat  float64
	Lon  float64
}

const (
	CodeSize = 4
	LatSize  = 8
	LonSize  = 8
)

func AirportCoordsMarshal(b *bytes.Buffer, record []string, indices []int) error {
	lat, err := strconv.ParseFloat(record[indices[1]], 64)
	if err != nil {
		return err
	}
	lon, err := strconv.ParseFloat(record[indices[2]], 64)
	if err != nil {
		return err
	}

	WriteString(b, record[indices[0]])
	binary.Write(b, binary.LittleEndian, lat)
	binary.Write(b, binary.LittleEndian, lon)

	return nil
}

func AirportCoordsUnmarshal(r *bufio.Reader) (data AirportCoords, err error) {
	data.Code, err = ReadString(r)

	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &data.Lat)
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &data.Lon)
	}

	return data, err
}
