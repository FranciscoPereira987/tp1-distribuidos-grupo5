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

func (kg KeyGenerator) NewRoundRobinKeysGenerator() RoundRobinKeysGenerator {
	keys := make([]string, 0, int(kg))
	for i := 1; i <= cap(keys); i++ {
		keys = append(keys, ShardKey(strconv.Itoa(i)))
	}

	return RoundRobinKeysGenerator{
		keys:  keys,
		index: 0,
	}
}

type RoundRobinKeysGenerator struct {
	keys  []string
	index int
}

func (rr *RoundRobinKeysGenerator) NextKey() string {
	key := rr.keys[rr.index]
	rr.index = (rr.index + 1) % len(rr.keys)
	return key
}
