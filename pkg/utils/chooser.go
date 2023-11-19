package utils

import (
	"math/rand"
	"time"
)

const (
	DefaultRetries = 3
)

type Choser struct {
	missing []uint
	retries uint
	retried map[uint]uint
	source  *rand.Rand
}

func NewChooser(from []uint) *Choser {
	missing := make([]uint, len(from))
	copy(missing, from)
	source := rand.New(rand.NewSource(time.Now().Unix()))

	return &Choser{
		missing,
		DefaultRetries,
		make(map[uint]uint),
		source,
	}
}

func NewChoserWithRetries(from []uint, retries uint) *Choser {
	choser := NewChooser(from)
	choser.retries = retries
	return choser
}

func (c *Choser) PeersLeft() bool {
	return len(c.missing) > 0
}

func (c *Choser) Retry(peer uint) {
	if value, ok := c.retried[peer]; ok && value < c.retries {
		c.missing = append(c.missing, peer)
		c.retried[peer] = value + 1
	}
}

func (c *Choser) Choose() uint {
	chosenOneAt := c.source.Intn(len(c.missing))
	chosenOne := c.missing[chosenOneAt]
	c.missing[chosenOneAt] = c.missing[len(c.missing)-1]
	c.missing = c.missing[:len(c.missing)-1]
	return chosenOne
}
