package protocol

var (
	ACK_OP_CODE = byte(0x01)
)

type AckMessage struct {
}

func (hello *AckMessage) Marshall() []byte {
	return nil
}

func (hello *AckMessage) UnMarshall(stream []byte) error {
	return nil
}

func (hello *AckMessage) Response() Message {
	return &AckMessage{}
}
