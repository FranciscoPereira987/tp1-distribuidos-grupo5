package typing

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

type ResultQ3 struct {
	Value middleware.ResultQ3
}

func NewResultQ3() *ResultQ3 {
	return new(ResultQ3)
}

func (r *ResultQ3) Number() byte {
	return middleware.Query3Flag
}

func (r *ResultQ3) AsRecord() []string {
	record := []string{hex.EncodeToString(r.Value.ID[:])}

	record = append(record, r.Value.Origin, r.Value.Destination)

	return append(record, fmt.Sprintf("%d", r.Value.Duration), r.Value.Stops)
}

func (r *ResultQ3) Serialize() []byte {
	return middleware.ResultQ3Marshal(r.Value)
}

func (r *ResultQ3) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(r, stream); err != nil {
		return err
	}
	data, err := middleware.Q3Unmarshal(bytes.NewReader(stream[1:]))
	r.Value = data
	return err
}
