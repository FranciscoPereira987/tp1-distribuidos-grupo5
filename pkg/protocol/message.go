package protocol

type Message interface {
	Marshall() []byte
	UnMarshall([]byte)
	Response() Message
}
