package id

import (
	"encoding/binary"
	"math/rand"
)

const Len = 4

func Generate() []byte {
	return binary.LittleEndian.AppendUint32(nil, rand.Uint32())
}
