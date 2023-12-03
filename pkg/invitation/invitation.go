package invitation

import (
	"errors"

	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/beater"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
	"github.com/sirupsen/logrus"
)

const (
	Electing uint = iota
	Coordinator
	Member
)

const (
	Invite = iota + 1
	Reject
	Accept
	Change
	Heartbeat
	Ok
)

type invite struct {
	Id        uint
	GroupSize uint
}

type reject struct {
	LeaderId uint
}

type accept struct {
	From      uint
	GroupSize uint
	Members   []uint
}

type change struct {
	NewLeaderId uint
}

type heartbeat struct{}

type ok struct{}

type Status struct {
	peers *utils.Peers

	dial *net.UDPConn

	id       uint
	leaderId uint
	config   *Config
	control  beater.Runable
}

func Invitation(config *Config) *Status {
	control, _ := beater.NewBeaterClient(config.Name, "0.0.0.0:"+config.Heartbeat)
	control.Run()
	return &Status{
		peers:    utils.NewPeers(config.Peers, config.Mapping),
		id:       config.Id,
		dial:     config.Conn,
		leaderId: config.Id,
		config:   config,
		control:  control,
	}
}

func stopBeater(beat beater.Runable) (err error) {
	if beat != nil {
		err = beat.Stop()
	}
	return
}

func (st *Status) Run() (err error) {

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGTERM)
	resultChan := make(chan error, 1)
	go func() {
		defer close(resultChan)
		resultChan <- st.run()
	}()
	defer close(stopChan)
	defer st.dial.Close()
	select {
	case err = <-resultChan:
		return
	case <-stopChan:
		stopBeater(st.control)
		return
	}
}

func (st *Status) run() (err error) {
	state := Electing
	lastState := Electing
	for err == nil {
		if lastState != state {
			err := stopBeater(st.control)
			if err != nil && !errors.Is(err, net.ErrClosed) {
				logrus.Fatalf("action: stoping beater | result: error | reason: %s", err)
			}
			if state == Coordinator {
				st.control = beater.NewBeaterServer(st.config.Names, st.config.Names, st.config.Heartbeat)
			} else {
				st.control, err = beater.NewBeaterClient(st.config.Name, "0.0.0.0:"+st.config.Heartbeat)
				if err != nil {
					logrus.Fatalf("action: starting beater client | result: fatal | reason: %s", err)
				}
			}
			st.control.Run()
		}
		lastState = state
		switch state {
		case Electing:
			//Runninng an election
			st.leaderId = st.id
			state, err = st.runElection()
		case Coordinator:
			//Leader
			//logrus.Infof("Action: peer %d acting as leader", st.id)
			state, err = st.ActAsLeader()
		case Member:
			//Member of a group
			//logrus.Infof("Action: peer %d acting as member | leader: peer %d ", st.id, st.leaderId)
			state, err = st.ActAsMember()
		}
	}

	return
}

func writeTo(s serializable, at *net.UDPConn, where string) error {
	//Writes the whole stream into at, if failed to do so, returns an error
	addr, err := net.ResolveUDPAddr("udp", where)
	if err != nil {
		return err
	}

	return utils.SafeWriteTo(s.serialize(), at, addr)
}

func writeToWithRetry(s serializable, at *net.UDPConn, where string) ([]byte, error) {
	//Tries to pass a stream to the other endpoint and awaits a response
	//It tries three times, else, an error is returned
	//If the response is not from the peer i sent the message, sends a reject 0
	retries := 3
	backoff := utils.BackoffFrom(time.Now().Nanosecond())
	var err error
	var buf []byte
	for ; retries > 0 && err == nil; retries-- {
		err = writeTo(s, at, where)
		if err == nil {
			backoff.SetReadTimeout(at)
			buf, err = readFrom(at, where)
		}
		if err == nil {
			return buf, err
		}
		backoff.IncreaseTimeOut()
	}

	return nil, err
}

func readFrom(from *net.UDPConn, expected string) ([]byte, error) {
	//Tries reading from io.Reader
	//Returns the whole array of bytes readed if successful.
	expectedAddr, err := net.ResolveUDPAddr("udp", expected)
	if err != nil {
		return nil, err
	}
	buf, addr, err := utils.SafeReadFrom(from)
	if err == nil && !addr.IP.Equal(expectedAddr.IP) {
		writeTo(reject{
			0,
		}, from, addr.String())
		return readFrom(from, expected)
	}
	return buf, err
}
