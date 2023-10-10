package typing

import (
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
)

func ResultsUnmarshal(stream []byte) (data Type, err error) {
	if len(stream) < 1 {
		err = errors.New("invalid stream for results")
		return
	}
	switch stream[0] {
	case middleware.Query1Flag:
		data = NewResultQ1()
		err = data.Deserialize(stream)

	case middleware.Query2Flag:
		data = NewResultQ2()
		err = data.Deserialize(stream)

	case middleware.Query3Flag:
		data = NewResultQ3()
		err = data.Deserialize(stream)

	case middleware.Query4Flag:
		data = NewResultQ4()
		err = data.Deserialize(stream)

	}

	return
}
