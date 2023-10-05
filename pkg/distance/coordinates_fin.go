package distance

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	COORD_FIN_TYPE_NUMBER = byte(0x07)
)

type CoordFin struct{}

func (coord *CoordFin) Number() byte {
	return COORD_FIN_TYPE_NUMBER
}

func (coord *CoordFin) Trim(stream []byte) []byte {
	if err := utils.CheckHeader(coord, stream); err != nil {
		return stream
	}
	return stream[1:]
}

func (coord *CoordFin) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(coord, stream); err != nil {
		return err
	}
	if err := typing.CheckTypeLength(1, stream); err != nil {
		return err
	}
	return nil
}

func (coord *CoordFin) Serialize() []byte {
	return utils.GetHeader(coord)
}

func (coord *CoordFin) AsRecord() []string {
	return []string{}
}
