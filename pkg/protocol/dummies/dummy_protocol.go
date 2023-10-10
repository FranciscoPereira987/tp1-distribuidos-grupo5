package dummies

import (
	"net"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/connection"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
)

type DummyProtocol struct {
	sckt connection.Conn
	ack  []byte
}

func NewDummyProtocol(at string) (connection.Conn, error) {
	host, port, _ := net.SplitHostPort(at)
	conn, err := connection.Dial(host, port)
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
	if buf[0] == protocol.HELLO_OP_CODE {
		proto.ack = protocol.NewHelloAckMessage().Marshal()
		return len(buf), nil
	}

	return proto.sckt.Write(buf)
}
