package connection

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware/id"
	log "github.com/sirupsen/logrus"
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
	d.Timeout = 10 * time.Second
	return d.DialContext(ctx, "tcp", addr)
}

type Backoff time.Duration

func NewBackoff() Backoff {
	return Backoff(0)
}

func (t *Backoff) Backoff() {
	if *t == 0 {
		*t = Backoff(time.Second)
	} else {
		*t = Backoff(min(time.Duration(*t) * 2, time.Second * 15))
	}
}

func (t Backoff) Wait() <-chan time.Time {
	return time.After(time.Duration(t))
}

const (
	hello = iota
	reconnect
)

func Accept(conn net.Conn) (bool, string, error) {
	var buf [1]byte
	_, err := conn.Read(buf[:])
	if err != nil {
		return false, "", err
	}

	var clientId []byte
	switch v := buf[0]; v {
	case hello:
		clientId = id.Generate()
		_, err = conn.Write(clientId)
	case reconnect:
		clientId = make([]byte, id.Len)
		_, err = io.ReadFull(conn, clientId)
	default:
		err = fmt.Errorf("%w: %q", ErrNotProto, v)
	}

	return buf[0] == reconnect, string(clientId), err
}

func ReconnectInput(conn net.Conn, clientId string) (int64, error) {
	send := [1 + id.Len]byte{reconnect}
	copy(send[1:], clientId)
	if _, err := conn.Write(send[:]); err != nil {
		return 0, err
	}

	var receive [8]byte
	_, err := io.ReadFull(conn, receive[:])

	return int64(binary.LittleEndian.Uint64(receive[:])), err
}

func SendState(conn net.Conn, offset int64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(offset))
	_, err := conn.Write(buf[:])
	return err
}

func ConnectInput(conn net.Conn) (string, error) {
	if _, err := conn.Write([]byte{hello}); err != nil {
		return "", err
	}

	var buf [id.Len]byte
	_, err := io.ReadFull(conn, buf[:])

	return string(buf[:]), err
}

func ConnectOutput(conn net.Conn, clientId string, progress int) error {
	var buf [id.Len + 8]byte
	copy(buf[:], clientId)
	binary.LittleEndian.PutUint64(buf[id.Len:], uint64(progress))
	_, err := conn.Write(buf[:])
	return err
}

func ReceiveState(conn net.Conn) (string, int, error) {
	var buf [id.Len + 8]byte
	_, err := io.ReadFull(conn, buf[:])
	progress := binary.LittleEndian.Uint64(buf[id.Len:])

	return string(buf[:id.Len]), int(progress), err
}

func Done(conn net.Conn) error {
	var ok [1]byte
	_, err := conn.Write(ok[:])
	return err
}

func WaitDone(conn net.Conn) error {
	var ok [1]byte
	_, err := conn.Read(ok[:])
	return err
}
