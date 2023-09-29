package protocol

import "github.com/franciscopereira987/tp1-distribuidos/pkg/utils"

type Message interface {
	utils.Numbered
	Marshall() []byte
	UnMarshall([]byte) error
	Response() Message
}
