package connection

import (
	"bufio"
	"io"
	"net"
)

type Conn struct {
	c io.ReadWriteCloser
}

func fromSckt(sckt io.ReadWriteCloser) (con *Conn) {
	con = new(Conn)
	con.c = sckt
	return
}

func (c *Conn) Buffer() bufio.ReadWriter {
	return *bufio.NewReadWriter(bufio.NewReader(c.c), bufio.NewWriter(c.c))
}

func Dial(host, port string) (con *Conn, err error) {
	sckt, err := net.Dial("tcp", net.JoinHostPort(host, port))
	if err == nil {
		con = fromSckt(sckt)
	}

	return
}

func FromListener(listener net.Listener) (*Conn, error) {
	c, err := listener.Accept()

	return fromSckt(c), err
}

func (c *Conn) Close() error {
	return c.c.Close()
}
