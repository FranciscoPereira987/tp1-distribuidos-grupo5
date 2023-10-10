package distance

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
	"github.com/umahmood/haversine"
)

type Coordinates = haversine.Coord

var (
	COORD_TYPE_NUMBER = byte(0x06)
)

type CoordWrapper struct {
	Value middleware.CoordinatesData
}

func (coord *CoordWrapper) Number() byte {
	return middleware.CoordFlag
}

func (coord *CoordWrapper) Serialize() []byte {
	return middleware.CoordMarshal(coord.Value)
}

func (coord *CoordWrapper) Trim(stream []byte) []byte {
	return nil
}

func (coord *CoordWrapper) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(coord, stream); err != nil {
		return err
	}
	buffer := bytes.NewReader(stream[1:])
	data, err :=  middleware.CoordUnmarshal(buffer)
	coord.Value = data
	return err
}

func (coord *CoordWrapper) AsRecord() []string {
	record := []string{coord.Value.AirportCode}
	record = append(record, fmt.Sprintf("%f", coord.Value.Latitude))
	return append(record, fmt.Sprintf("%f", coord.Value.Longitud))
}

func IntoData(coord Coordinates, name string) *protocol.DataMessage {
	wrapper := &CoordWrapper{
		Value: middleware.CoordinatesData{
			AirportCode: name,
			Latitude: coord.Lat,
			Longitud: coord.Lon,
		},
	}

	message := protocol.NewDataMessage(wrapper)
	return message
}

func CoordsFromData(data protocol.Data) (*Coordinates, error) {
	wrapper, ok := data.Type().(*CoordWrapper)
	if !ok {
		return nil, errors.New("invalid data message")
	}
	coords := &Coordinates{
		Lat: wrapper.Value.Latitude,
		Lon: wrapper.Value.Longitud,
	}
	return coords, nil
}

