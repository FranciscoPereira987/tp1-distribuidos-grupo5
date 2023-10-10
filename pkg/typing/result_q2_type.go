package typing

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

type ResultQ2 struct {
	Value middleware.DataQ2
}

func NewResultQ2() *ResultQ2{
	return new(ResultQ2)
}

func (r *ResultQ2) Number() byte {
	return middleware.Query2Flag
}

func (r *ResultQ2) AsRecord() []string {
	record := []string{hex.EncodeToString(r.Value.ID[:])}

	record = append(record, r.Value.Origin, r.Value.Destination)

	return append(record, fmt.Sprintf("%d", r.Value.TotalDistance))
}

func (r *ResultQ2) Serialize() []byte {
	return middleware.Q2Marshal(r.Value)
}

func (r *ResultQ2) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(r, stream); err != nil {
		return err
	}
	data, err := middleware.Q2Unmarshal(bytes.NewReader(stream[1:]))
	r.Value = data
	return err
}
