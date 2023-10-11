package typing

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

type ResultQ1 struct {
	Value middleware.ResultQ1
}

func NewResultQ1() *ResultQ1 {
	return new(ResultQ1)
}

func (r *ResultQ1) Number() byte {
	return middleware.Query1Flag
}

func (r *ResultQ1) AsRecord() []string {
	record := []string{hex.EncodeToString(r.Value.ID[:])}

	record = append(record, r.Value.Origin, r.Value.Destination)

	return append(record, fmt.Sprintf("%f", r.Value.Price), r.Value.Stops)
}

func (r *ResultQ1) Serialize() []byte {
	return middleware.ResultQ1Marshal(r.Value)
}

func (r *ResultQ1) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(r, stream); err != nil {
		return err
	}
	data, err := middleware.Q1Unmarshal(bytes.NewReader(stream[1:]))
	r.Value = data
	return err
}
