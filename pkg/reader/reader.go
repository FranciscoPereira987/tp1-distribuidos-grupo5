package reader

import (
	"errors"
	"fmt"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

type Reader interface {
	ReadData() (protocol.Data, error)
	Close() error
}

func GetType(data typing.Type) (typing.Type, error) {
	rd, ok := data.(*typing.RawData)
	if !ok {
		return nil, fmt.Errorf("not raw data type: %s", data)
	}
	switch rd.DataType {
	case typing.FLIGHT:
		return IntoFlightData(rd.Data)
	case typing.COORDINATES:
		return IntoCoordinates(rd.Data)
	default:
		return nil, errors.New("invalid data type sent")
	}
}
