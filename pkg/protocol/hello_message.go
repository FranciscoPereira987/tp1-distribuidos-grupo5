package protocol

var (
	HELLO_OP_CODE = byte(0x01)
)

type HelloMessage struct {
}

func (hello *HelloMessage) Marshall() []byte {
	return nil
}

func (hello *HelloMessage) UnMarshall(stream []byte) error {
	return nil
}

func (hello *HelloMessage) Response() Message {
	return &HelloMessage{}
}
