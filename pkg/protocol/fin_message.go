package protocol

var (
	FIN_OP_CODE = byte(0x01)
)

type FinMessage struct {
}

func (hello *FinMessage) Marshall() []byte {
	return nil
}

func (hello *FinMessage) UnMarshall(stream []byte) error {
	return nil
}

func (hello *FinMessage) Response() Message {
	return &FinMessage{}
}
