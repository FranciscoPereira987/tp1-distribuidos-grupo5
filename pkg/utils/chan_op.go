package utils

import (
	"net"

	"github.com/sirupsen/logrus"
)

func Merge(c ...<-chan *net.UDPAddr) chan *net.UDPAddr {

	out := make(chan *net.UDPAddr)
	for _, channel := range c {
		go func(from <-chan *net.UDPAddr) {
			defer func() {
				if recover() != nil {
					logrus.Debug("action: closing run | status: send on closed channel")
				}
			}()
			for msg := range from {
				out <- msg
			}
		}(channel)
	}
	return out
}
