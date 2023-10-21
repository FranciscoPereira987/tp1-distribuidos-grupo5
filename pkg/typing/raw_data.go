package typing

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

const (
	COORDINATES = iota
	FLIGHT
)

type RawData struct {
	Data     string
	DataType int
}

func NewData(dataType int, line string) *RawData {
	return &RawData{
		Data:     line,
		DataType: dataType,
	}
}

func (rd *RawData) Number() byte {
	return middleware.FlightType
}

func (rd *RawData) split() string {
	if rd.DataType == COORDINATES {
		return ";"
	}
	return ","
}

func (rd *RawData) AsRecord() []string {
	return strings.Split(rd.Data, rd.split())
}

func (rd *RawData) Serialize() []byte {
	buf := make([]byte, 1)
	buf[0] = rd.Number()

	buf = binary.BigEndian.AppendUint16(buf, uint16(rd.DataType))
	buf = middleware.AppendString(buf, rd.Data)

	return buf
}

func (rd *RawData) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(rd, stream); err != nil {
		return err
	}
	if len(stream) < 3 {
		return io.ErrUnexpectedEOF
	}
	rd.DataType = int(binary.BigEndian.Uint16(stream[1:3]))
	reader := bytes.NewReader(stream[3:])
	data, err := middleware.ReadString(reader)

	if err == nil {
		rd.Data = data
	}
	return err
}
