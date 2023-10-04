package reader

import "github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"

type Reader interface {
	ReadData() (protocol.Data, error)
	Close() error
}
