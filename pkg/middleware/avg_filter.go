package middleware

import (
	"bytes"
	"encoding/binary"
)

type AvgFilterData struct {
	Origin      string
	Destination string
	Price       float32
}

func IsAvgPriceMessage(msg []byte) bool {
	return len(msg) == 4
}

func AvgUnmarshal(msg []byte) (float32, error) {
	var avgPrice float32
	err := binary.Read(bytes.NewReader(msg), binary.LittleEndian, &avgPrice)
	return avgPrice, err
}

func AvgFilterUnmarshal(msg []byte) (data AvgFilterData, err error) {
	r := bytes.NewReader(msg)

	data.Origin, err = ReadString(r)
	if err == nil {
		data.Destination, err = ReadString(r)
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &(data.Price))
	}

	return data, err
}
