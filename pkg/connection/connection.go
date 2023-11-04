package connection

import (
	"context"
	"net"

	log "github.com/sirupsen/logrus"
)

func Listen(ctx context.Context, addr string) (<-chan net.Conn, error) {
	var lc net.ListenConfig

	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		log.Info("closing listener")
		ln.Close()
	}()

	ch := make(chan net.Conn)
	go func() {
		defer close(ch)
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Error(err)
				return
			}
			ch <- conn
		}
	}()

	return ch, nil
}

func Dial(ctx context.Context, addr string) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		log.Infof("closing connection to %s", addr)
		conn.Close()
	}()

	return conn, nil
}
