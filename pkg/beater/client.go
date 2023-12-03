package beater

import (
	"errors"

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

	stopChan chan struct{}
	running  bool
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
		nil,
		false,
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
		st.stopChan = make(chan struct{}, 1)
		select {
		case st.resultChan <- st.run():
		case <-st.stopChan:
			st.resultChan <- nil
			st.Stop()
		}
	}()
}

func (st *BeaterClient) Stop() (err error) {
	if st.running {
		defer close(st.resultChan)
		defer close(st.stopChan)
		st.stopChan <- struct{}{}
		err = st.conn.Close()
		err = errors.Join(err, <-st.resultChan)
		st.running = false
	}
	return err
}
