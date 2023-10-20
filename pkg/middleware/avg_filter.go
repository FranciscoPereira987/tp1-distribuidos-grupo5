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
	return len(msg) == 16
}

func AvgPriceMarshal(avgPrice float64, count int) []byte {
	var w bytes.Buffer
	binary.Write(&w, binary.LittleEndian, avgPrice)
	return binary.LittleEndian.AppendUint64(w.Bytes(), uint64(count))
}

func AvgPriceUnmarshal(msg []byte) (float64, int, error) {
	var priceSubtotal float64
	err := binary.Read(bytes.NewReader(msg[:8]), binary.LittleEndian, &priceSubtotal)
	priceCount := binary.LittleEndian.Uint64(msg[8:])
	return priceSubtotal, int(priceCount), err
}

func AvgFilterMarshal(data AvgFilterData) []byte {
	buf := make([]byte, 0)
	buf = AppendString(buf, data.Origin)
	buf = AppendString(buf, data.Destination)
	var w bytes.Buffer
	binary.Write(&w, binary.LittleEndian, data.Price)
	buf = append(buf, w.Bytes()...)
	return buf
}

func AvgFilterUnmarshal(msg []byte) (data AvgFilterData, err error) {
	r := bytes.NewReader(msg)

	data.Origin, err = ReadString(r)
	if err == nil {
		data.Destination, err = ReadString(r)
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &data.Price)
	}

	return data, err
}
