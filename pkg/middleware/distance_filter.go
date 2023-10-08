package middleware

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)


type CoordinatesData struct {
	AirportCode string

	Latitude float64
	Longitud float64
}



func CoordMarshal(coords CoordinatesData) (buf []byte) {

	buf = append(buf, CoordFlag)

	buf = append(buf, coords.AirportCode[:]...)

	var w bytes.Buffer
	binary.Write(&w, binary.LittleEndian, &coords.Latitude)
	binary.Write(&w, binary.LittleEndian, &coords.Longitud)

	buf = append(buf, w.Bytes()...)

	return
}


func CoordUnmarshal(r *bytes.Reader) (data CoordinatesData, err error) {
	buf := make([]byte, 3)
	_, err = io.ReadFull(r, buf)
	if err == nil {
		data.AirportCode = string(buf)
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
func DistanceFilterUnmarshal(buf []byte) (dataType int, data any, err error) {
	if len(buf) < 1 {
		err = errors.New("stream is empty")
		return
	}
	r := bytes.NewReader(buf)
	switch buf[0] {
	case Query2Flag:
		dataType = 2
		data, err = Q2Unmarshal(r)
	case CoordFlag:
		dataType = 1
		data, err = CoordUnmarshal(r)
	default:
		err = errors.New("invalid data for distance filter")
	}

	return
}

