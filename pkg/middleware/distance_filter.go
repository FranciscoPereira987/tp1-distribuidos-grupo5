package middleware

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type CoordinatesData struct {
	AirportCode string

	Latitude float64
	Longitud float64
}

func CoordMarshal(coords CoordinatesData) (buf []byte) {

	buf = append(buf, CoordFlag)

	buf = AppendString(buf, coords.AirportCode)

	var w bytes.Buffer
	binary.Write(&w, binary.LittleEndian, &coords.Latitude)
	binary.Write(&w, binary.LittleEndian, &coords.Longitud)

	buf = append(buf, w.Bytes()...)

	return
}

func CoordUnmarshal(r *bytes.Reader) (data CoordinatesData, err error) {

	if err == nil {
		data.AirportCode, err = ReadString(r)
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &(data.Latitude))
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &(data.Longitud))
	}

	return data, err
}

/*
Returns the corresponding data based on the buf value
1 => CoordinatesData type
2 => DataQ2 type
*/
func DistanceFilterUnmarshal(buf []byte) (data any, err error) {
	if len(buf) < 1 {
		err = errors.New("stream is empty")
		return
	}
	r := bytes.NewReader(buf[1:])
	switch buf[0] {
	case Query2Flag:
		data, err = Q2Unmarshal(r)
	case CoordFlag:
		data, err = CoordUnmarshal(r)
	default:
		err = errors.New("invalid data for distance filter")
	}

	return
}
