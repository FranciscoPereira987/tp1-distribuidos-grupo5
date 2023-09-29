package protocol

type Message interface {
	Marshall() []byte
	UnMarshall([]byte) error
	Response() Message
}
