package common

import (
	"bufio"
	"fmt"

	mid "github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/spf13/viper"
)

type Middleware struct {
	conn        mid.Connection
	readBuffer  bufio.Reader
	writeBuffer bufio.Writer
}

func NewMiddleware(v *viper.Viper, conn mid.Connection) *Middleware {
	return &Middleware{
		conn:        conn,
		readBuffer:  bufio.NewReader(conn),
		writeBuffer: bufio.NewWriter(conn),
	}
}

func (m *Middleware) Connect() error {
	return fmt.Errorf("Not Implemented")
}

func (m *Middleware) Close() error {
	errFlush := m.Flush()
	errClose := m.conn.Close()
	if errFlush != nil && errClose != nil {
		return fmt.Errorf("Flush: %w, Close: %w", errFlush, errClose)
	}
	if errFlush != nil {
		return errFlush
	}
	return errClose
}

func (m *Middleware) Flush() error {
	return m.writeBuffer.Flush()
}

func (m *Middleware) Pull() /* Filght */ {
}

func (m *Middleware) Push(v /* Flight */) error {
}
