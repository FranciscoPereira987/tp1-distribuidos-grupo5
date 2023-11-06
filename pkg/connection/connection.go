package connection

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	log "github.com/sirupsen/logrus"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware/id"
)

var ErrNotProto = errors.New("unexpected message")

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

const (
	hello = iota
	reconnect
)

func Accept(conn net.Conn) (string, error) {
	var buf [1]byte
	if _, err := conn.Read(buf[:]); err != nil {
		return "", err
	}

	switch v := buf[0]; v {
	case hello:
	case reconnect:
		panic("Not implemented")
	default:
		return "", fmt.Errorf("%w: %q", ErrNotProto, v)
	}

	clientId := id.Generate()
	if _, err := conn.Write(clientId); err != nil {
		return "", err
	}

	return string(clientId), nil
}

func ConnectInput(conn net.Conn) (string, error) {
	if _, err := conn.Write([]byte{hello}); err != nil {
		return "", err
	}

	return ReceiveId(conn)
}

func ConnectOutput(conn net.Conn, id string) error {
	_, err := conn.Write([]byte(id))
	return err
}

func ReceiveId(conn net.Conn) (string, error) {
	var buf [id.Len]byte
	_, err := io.ReadFull(conn, buf[:])

	return string(buf[:]), err
}
