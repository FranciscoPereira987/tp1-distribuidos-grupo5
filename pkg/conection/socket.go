package conection

import (
	"io"
	"net"
)

type Conn io.ReadWriteCloser

func NewSocketConnection(at string) (con Conn, err error) {
	return net.Dial("tcp", at)
}

func FromListener(listener net.Listener) (Conn, error) {
	return listener.Accept()
}
