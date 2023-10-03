package distance

import (
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	AIRPORT_DATA_TYPE_NUMBER = byte(0x08)
)

type AirportDataType struct {
	id          *typing.StrType
	origin      *typing.StrType
	destination *typing.StrType

	totalDistance *typing.FloatType
}

func NewAirportData(id string, origin string, destination string, distance float64) (*AirportDataType, error) {
	dataId, err := typing.NewStr(id)
	if err != nil {
		return nil, err
	}
	originData, err := typing.NewStr(origin)
	if err != nil {
		return nil, err
	}
	destinationData, err := typing.NewStr(destination)
	if err != nil {
		return nil, err
	}
	distanceData := &typing.FloatType{Value: distance}

	return &AirportDataType{
		id:            dataId,
		destination:   destinationData,
		origin:        originData,
		totalDistance: distanceData,
	}, nil
}

func (data *AirportDataType) Number() byte {
	return AIRPORT_DATA_TYPE_NUMBER
}

func (data *AirportDataType) Serialize() []byte {
	stream := utils.GetHeader(data)
	stream = append(stream, data.id.Serialize()...)
	stream = append(stream, data.origin.Serialize()...)
	stream = append(stream, data.destination.Serialize()...)
	return append(stream, data.totalDistance.Serialize()...)
}

func (data *AirportDataType) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(data, stream); err != nil {
		return err
	}
	id, rest := typing.GetTypeFromStream(data.id, stream[1:])
	if err := data.id.Deserialize(id); err != nil {
		return err
	}
	origin, rest := typing.GetTypeFromStream(data.origin, rest)
	if err := data.origin.Deserialize(origin); err != nil {
		return err
	}
	destination, rest := typing.GetTypeFromStream(data.destination, rest)
	if err := data.destination.Deserialize(destination); err != nil {
		return err
	}
	if err := data.totalDistance.Deserialize(rest); err != nil {
		return err
	}
	return nil
}

func (data *AirportDataType) Trim(stream []byte) []byte {
	if err := utils.CheckHeader(data, stream); err != nil {
		return stream
	}
	rest := data.id.Trim(stream[1:])
	rest = data.origin.Trim(rest)
	rest = data.destination.Trim(rest)
	return data.totalDistance.Trim(rest)
}

func AirportFromData(data protocol.Data) (*AirportDataType, error) {
	unwrapped, ok := data.Type().(*AirportDataType)
	if !ok {
		return nil, errors.New("not an airport data type")
	}
	return unwrapped, nil
}
