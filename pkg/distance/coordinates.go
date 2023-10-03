package distance

import (
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
	"github.com/umahmood/haversine"
)

type Coordinates  = haversine.Coord

var (
	COORD_TYPE_NUMBER = byte(0x06)
)

type CoordWrapper struct {
	Lat typing.FloatType
	Long typing.FloatType
}

func (coord *CoordWrapper) Number() byte {
	return COORD_TYPE_NUMBER
}

func (coord *CoordWrapper) Serialize() []byte {
	buf := []byte{COORD_TYPE_NUMBER}
	buf = append(buf, coord.Lat.Serialize()...)
	
	return append(buf, coord.Long.Serialize()...)
}

func (coord *CoordWrapper) Trim(stream []byte) []byte {
	if err := utils.CheckHeader(coord, stream); err != nil {
		return stream	
	}
	stream = coord.Lat.Trim(stream[1:])
	return coord.Long.Trim(stream)
}

func (coord *CoordWrapper) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(coord, stream); err != nil {
		return err
	}
	lat, long := typing.GetTypeFromStream(&coord.Lat, stream[1:])
	if err := coord.Lat.Deserialize(lat); err != nil {
		return err
	}

	if err := coord.Long.Deserialize(long); err != nil {
		return err
	}

	return nil
}

func IntoData(coord Coordinates) *protocol.DataMessage {
	wrapper := &CoordWrapper{
		Lat: typing.FloatType{
			Value: coord.Lat,
		},
		Long: typing.FloatType{
			Value: coord.Lon,
		},
	}

	message := protocol.NewDataMessage(wrapper)
	return message
}

func FromData(data *protocol.DataMessage) (*Coordinates, error) {
	wrapper, ok := data.Type().(*CoordWrapper)
	if !ok {
		return nil, errors.New("invalid data message")
	}
	coords := &Coordinates{
		Lat: wrapper.Lat.Value,
		Lon: wrapper.Long.Value,
	}
	return coords, nil
}

