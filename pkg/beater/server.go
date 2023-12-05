package beater

import (
	"errors"

	"net"
	"sync"
	"time"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/dood"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
	"github.com/sirupsen/logrus"
)

const (
	Heartbeat = iota + 1
	Ok
)

type heartbeat struct{}

func (hb heartbeat) serialize() []byte {
	return []byte{Heartbeat}
}

type ok struct {
	whom string
}

type client struct {
	Name string
	Addr string
}

func (o ok) serialize() []byte {
	header := []byte{Ok}
	body := utils.EncodeString(o.whom)
	return append(header, body...)
}

func (o *ok) deserialize(stream []byte) error {
	decoded, err := utils.DecodeString(stream)
	if err == nil {
		o.whom = decoded
	}
	return err
}

type url = any

type Runable interface {
	Stop() error
	Run()
}

/*
The Beater Server makes sure that all uniquely named clients
are alive.
*/
type BeaterServer struct {
	clients map[string]*timer

	clientInfo []client

	sckt *net.UDPConn

	wg *sync.WaitGroup

	resultsChan chan error

	port string

	shutdown chan struct{}
	running  bool

	dood *dood.DooD
}

/*
Timer runs a routine to keep clients
in check, ensuring that they are alive
*/
type timer struct {
	//Allows for the timer to reset
	//if Inbound is false, then the routine shutsdown
	InboundChan chan bool
	//Outputs a message to be delivered to the client
	OutboundChan chan *net.UDPAddr

	//Outputs the name to restart the service
	restartChan chan string

	clientAddr string

	/*
		Service name that's going to be used to
		recreate the client
	*/
	name string

	//Heartbeat timer
	maxTime time.Duration
}

func NewTimer(at string, name string, restart chan string, outbound chan *net.UDPAddr) *timer {
	t := new(timer)

	t.InboundChan = make(chan bool, 1)
	t.OutboundChan = outbound
	t.clientAddr = at
	t.name = name
	t.maxTime = time.Millisecond * 100
	t.restartChan = restart

	return t
}

func (t *timer) resolveAddr() (addr *net.UDPAddr, running bool) {
	addr, err := net.ResolveUDPAddr("udp", t.clientAddr)
	running = true
	if err != nil {
		logrus.Errorf("action: resolving client address | result: failed | action: re-instantiating client")
		t.restartChan <- t.name
	}
	for err != nil && running {
		<-time.After(t.maxTime * 5)
		addr, err = net.ResolveUDPAddr("udp", t.clientAddr)
		select {
		case <-t.InboundChan:
			running = false
		default:
		}
	}
	return
}

func (t *timer) executeTimer(group *sync.WaitGroup) {
	clientAddr, running := t.resolveAddr()
	retriesLeft := 3

	for running {
		timeout := time.After(t.maxTime)
		t.OutboundChan <- clientAddr
		select {
		case <-timeout:
			retriesLeft--
			if retriesLeft <= 0 {
				clientAddr, running = t.resolveAddr()
			} else {
				logrus.Infof("action: client %s timer | status: client unresponsive | remaining attempts: %d", t.name, retriesLeft)
			}
		case running = <-t.InboundChan:
			if !running {
				logrus.Infof("action: Client %s timer | status: ending", t.name)
			}
			retriesLeft = 3
			<-time.After(t.maxTime / 2)
		}
	}
	group.Done()

}

func NewBeaterServer(clients []string, clientAddrs []string, at string) *BeaterServer {
	client_map := make(map[string]*timer)

	clientInfo := make([]client, 0)
	for index, clientName := range clients {
		clientInfo = append(clientInfo, client{
			clientName,
			clientAddrs[index],
		})
	}

	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+at)
	if err != nil {
		logrus.Fatalf("action: initializing beater server | status: failed | reason: %s", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		logrus.Fatalf("action: initializing beater server | status: failed | reason: %s", err)
	}
	doodServer, err := dood.NewDockerClientDefault()
	if err != nil {
		logrus.Fatalf("action: initializing beater server | status: failed | reason: %s", err)
	}

	return &BeaterServer{
		client_map,
		clientInfo,
		conn,
		new(sync.WaitGroup),
		make(chan error, 1),
		at,
		nil,
		false,
		doodServer,
	}
}

/*
Initiates timers and merge channels
*/
func (b *BeaterServer) initiateTimers(port string, dockerChan chan string) chan *net.UDPAddr {
	timersChan := make(chan *net.UDPAddr, 1)
	for _, value := range b.clientInfo {

		timer := NewTimer(value.Addr+":"+port, value.Name, dockerChan, timersChan)

		b.clients[value.Name] = timer
		go timer.executeTimer(b.wg)

	}
	b.wg.Add(len(b.clientInfo))
	return timersChan
}

/*
Initiates socket reading routine
*/
func (b *BeaterServer) initiateReader() chan []byte {
	readerChan := make(chan []byte, 1)
	go func() {
		var err error
		for err == nil {
			beat, _, err_read := utils.SafeReadFrom(b.sckt)
			err = err_read
			if err == nil {
				readerChan <- beat
			}
		}
	}()
	return readerChan
}

func (b *BeaterServer) parseBeat(beat []byte) {
	if beat[0] != Ok {
		logrus.Errorf("recieved invalid beat response")
		return
	}
	ok := &ok{}
	err := ok.deserialize(beat[1:])
	if err != nil {
		logrus.Errorf("error while deserializing beat response: %s", err)
		return
	}
	if timer, ok := b.clients[ok.whom]; ok {
		timer.InboundChan <- true
	}
}

/*
Runs a routine that is in charge of writing to the socket
heartbeats
*/
func (b *BeaterServer) writeRoutine(channel <-chan *net.UDPAddr) {
	for addr := range channel {
		err := utils.SafeWriteTo(heartbeat{}.serialize(), b.sckt, addr)
		if err != nil {
			logrus.Errorf("action: heartbeat to services | status: %s", err)
		}
	}
	logrus.Info("action: write routine | status: finishing")
}

func (b *BeaterServer) run(port string) (err error) {
	dockerChan := b.dood.StartIncoming()
	defer close(dockerChan)
	timersChan := b.initiateTimers(port, dockerChan)
	defer close(timersChan)
	readerChan := b.initiateReader()
	defer close(readerChan)
	go b.writeRoutine(timersChan)

	shutdown := make(chan struct{}, 1)
	b.shutdown = shutdown
	defer close(shutdown)
	b.running = true
loop:
	for {
		select {
		case beat := <-readerChan:
			b.parseBeat(beat)
		case <-shutdown:
			b.running = false
			break loop

		}
	}

	for _, t := range b.clients {
		t.InboundChan <- false
	}

	b.wg.Wait()

	return
}

/*
1. server starts all timers
2. server loop:
  - Select -> write to client channel
    -> Send to write rutine
    -> listen from socket channel
    -> Update client mappings
    -> listen for shutdown
    -> shutdown if necessary
*/
func (b *BeaterServer) Run() {
	go func() {
		b.resultsChan <- b.run(b.port)
	}()
}

func (b *BeaterServer) Stop() (err error) {
	if b.running {
		err = b.sckt.Close()
		b.dood.Shutdown()
		b.shutdown <- struct{}{}
		err = errors.Join(err, <-b.resultsChan)
		b.running = false
	}

	return err
}
