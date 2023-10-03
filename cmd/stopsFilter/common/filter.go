package common

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

type Filter struct {
	conn Middleware
}

func NewFilter(conn Middleware) *Filter {
	return &Filter{
		conn: conn,
	}
}

func countStops(v /* Flight */) int {
	return 0
}

func (f *Filter) Run() error {
	for {
		v, err := f.conn.Pull()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if countStops(v) >= 3 {
			if err := f.conn.Push(v); err != nil {
				return err
			}
		}
	}
}

func (f *Filter) Start(sig chan os.Signal) error {
	done := make(chan error)

	go func() {
		done <- f.Run()
	}()

	select {
	case <-sig:
		f.conn.Close()
		return <-done
	case err := <-done:
		return err
	}
}
