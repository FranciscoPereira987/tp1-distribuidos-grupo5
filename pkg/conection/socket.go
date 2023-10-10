package conection

import (
	"bytes"
	"io"
	"net"
)

type Conn io.ReadWriteCloser

/*
Not thread safe implementation of a safe socket
*/
type socket struct {
	dial net.Conn
}

func NewSocketConnection(at string) (con Conn, err error) {
	dial, err := net.Dial("tcp", at)
	con = &socket{
		dial: dial,
	}
	return
}

func FromListener(listener net.Listener) (Conn, error) {
	sckt, err := listener.Accept()
	return &socket{
		dial: sckt,
	}, err
}

/*
Safe read implementation, either reads the whole buffer
or returns an error
*/
func (s *socket) Read(buf []byte) (readed int, err error) {
	for readed < len(buf) {
		newlyReaded, readErr := s.dial.Read(buf[readed:])
		if readErr != nil {
			err = readErr
			break
		}
		readed += newlyReaded
	}

	return
}

/*
Safe write implementation, either writes the whole buffer
or returns an error
*/
func (s *socket) Write(buf []byte) (writen int, err error) {
	return io.Copy(s.dial, bytes.NewReader(buf))
}

func (s *socket) Close() error {
	return s.dial.Close()
}
