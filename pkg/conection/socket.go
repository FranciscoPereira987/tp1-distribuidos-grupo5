package conection

import (
	"io"
	"net"
)

type Conn interface {
	io.Reader
	io.Writer
	Close() error
}

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

	for len(buf) > 0 {
		newlyWriten, writeErr := s.dial.Write(buf)
		
		if writeErr != nil {
			err = writeErr
			break
		}
		buf = buf[newlyWriten:]
		writen += newlyWriten
	}

	return
}

func (s *socket) Close() error {
	return s.dial.Close()
}
