package dummies

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/conection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
)

type DummyProtocol struct {
	sckt conection.Conn
	ack  []byte
}

func NewDummyProtocol(at string) (conection.Conn, error) {
	conn, err := conection.NewSocketConnection(at)
	if err != nil {
		return nil, err
	}
	return &DummyProtocol{
		sckt: conn,
	}, nil
}

func (proto *DummyProtocol) Close() error {
	return proto.sckt.Close()
}

func (proto *DummyProtocol) Read(buf []byte) (int, error) {

	if proto.ack != nil {

		for index, _ := range buf {
			buf[index] = proto.ack[index]
		}
		proto.ack = proto.ack[len(buf):]
		if len(proto.ack) == 0 {
			proto.ack = nil
		}
		return len(buf), nil
	}
	return proto.sckt.Read(buf)
}

func (proto *DummyProtocol) Write(buf []byte) (int, error) {
	if buf[0] == protocol.DATA_OP_CODE {
		proto.ack = protocol.NewDataAckMessage().Marshal()
	}
	if buf[0] == protocol.FIN_OP_CODE {
		proto.ack = protocol.NewFinAckMessage().Marshal()
	}
	if buf[0] == protocol.HELLO_OP_CODE {
		proto.ack = protocol.NewHelloAckMessage().Marshal()
		return len(buf), nil
	}

	return proto.sckt.Write(buf)
}
