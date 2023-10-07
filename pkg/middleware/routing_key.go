package middleware

import (
	"hash"
	"hash/fnv"
	"strconv"
	"strings"
)

type KeyGenerator struct {
	h   hash.Hash32
	mod int
}

func NewKeyGenerator(mod int) KeyGenerator {
	return KeyGenerator{
		h:   fnv.New32(),
		mod: mod,
	}
}

func ShardKey(id string) string {
	return id
}

func (kg KeyGenerator) KeyFrom(airports ...string) string {
	kg.h.Write([]byte(strings.Join(airports, ".")))
	v := kg.h.Sum32()
	kg.h.Reset()

	return strconv.Itoa(v % kg.mod + 1)
}
