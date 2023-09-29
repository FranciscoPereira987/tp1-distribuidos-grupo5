package protocol

var (
	ERR_OP_CODE = byte(0x01)
)

type ErrMessage struct {
}

func (hello *ErrMessage) Marshall() []byte {
	return nil
}

func (hello *ErrMessage) UnMarshall(stream []byte) error {
	return nil
}

func (hello *ErrMessage) Response() Message {
	return &ErrMessage{}
}
