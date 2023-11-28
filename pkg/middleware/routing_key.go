package middleware

import (
	"fmt"
	"hash/fnv"
)

type KeyGenerator int

func NewKeyGenerator(mod int) KeyGenerator {
	return KeyGenerator(mod)
}

func ShardKey(id string) string {
	return id
}

func (kg KeyGenerator) KeyFrom(sink, origin, destination string) string {
	h := fnv.New32()

	h.Write([]byte(origin))
	h.Write([]byte("."))
	h.Write([]byte(destination))

	v := h.Sum32()%uint32(kg) + 1

	return fmt.Sprintf("%s.%d", sink, v)
}

func (kg KeyGenerator) NewRoundRobinKeysGenerator() RoundRobinKeysGenerator {
	return RoundRobinKeysGenerator{
		mod:   int(kg),
		index: 0,
	}
}

type RoundRobinKeysGenerator struct {
	mod   int
	index int
}

func (rr *RoundRobinKeysGenerator) NextKey(sink string) string {
	v := rr.index + 1
	rr.index = v % rr.mod
	return fmt.Sprintf("%s.%d", sink, v)
}
