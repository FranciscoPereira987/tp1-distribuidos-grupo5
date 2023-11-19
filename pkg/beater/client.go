package beater

import (
	"errors"
	"os"

	"net"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

/*
Each client has a name which serves as an ID of the client
*/
type BeaterClient struct {
	conn *net.UDPConn

	resultChan chan error

	name string
}

func NewBeaterClient(name string, addr string) (*BeaterClient, error) {
	address, err := net.ResolveUDPAddr("udp", addr)

	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", address)

	return &BeaterClient{
		conn,
		make(chan error, 1),
		name,
	}, err
}

func (st *BeaterClient) run() error {
	var err error
	for err == nil {
		recovered, server, err_read := utils.SafeReadFrom(st.conn)
		err = err_read
		if err == nil {
			if recovered[0] == Heartbeat {
				err = utils.SafeWriteTo(ok{st.name}.serialize(), st.conn, server)
			}
		}
	}
	return err
}

func (st *BeaterClient) Run() {
	go func() {
		termination := make(chan os.Signal, 1)
		defer close(termination)
		select {
		case st.resultChan <- st.run():
		case <-termination:
			st.resultChan <- nil
			st.Stop()
		}
	}()
}

func (st *BeaterClient) Stop() error {
	defer close(st.resultChan)
	err := st.conn.Close()
	err = errors.Join(err, <-st.resultChan)
	return err
}
