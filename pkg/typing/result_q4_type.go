package typing

import (
	"bytes"
	"fmt"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

type ResultQ4 struct {
	Value middleware.ResultQ4
}

func NewResultQ4() *ResultQ4 {
	return new(ResultQ4)
}

func (r *ResultQ4) Number() byte {
	return middleware.Query3Flag
}

func (r *ResultQ4) AsRecord() []string {
	record := []string{}

	record = append(record, r.Value.Origin, r.Value.Destination)

	return append(record, fmt.Sprintf("%f", r.Value.AvgPrice), fmt.Sprintf("%f", r.Value.MaxPrice))
}

func (r *ResultQ4) Serialize() []byte {
	return middleware.Q4Marshal(r.Value)
}

func (r *ResultQ4) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(r, stream); err != nil {
		return err
	}
	data, err := middleware.Q4Unmarshal(bytes.NewReader(stream[1:]))
	r.Value = data
	return err
}
