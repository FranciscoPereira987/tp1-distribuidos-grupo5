package middleware

import (
	"hash/fnv"
	"strconv"
)

type KeyGenerator int

func NewKeyGenerator(mod int) KeyGenerator {
	return KeyGenerator(mod)
}

func ShardKey(id string) string {
	return id
}

func (kg KeyGenerator) KeyFrom(origin, destination string) string {
	h := fnv.New32()

	h.Write([]byte(origin))
	h.Write([]byte("."))
	h.Write([]byte(destination))

	v := h.Sum32()%uint32(kg) + 1

	return strconv.Itoa(int(v))
}
