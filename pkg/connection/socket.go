package connection

import (
	"io"
	"net"
)

type Conn io.ReadWriteCloser

func Dial(host, port string) (con Conn, err error) {
	return net.Dial("tcp", net.JoinHostPort(host, port))
}

func FromListener(listener net.Listener) (Conn, error) {
	return listener.Accept()
}
