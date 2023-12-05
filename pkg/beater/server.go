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

	addr *net.UDPAddr
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
	clientAddr string

	/*
		Service name that's going to be used to
		recreate the client
	*/
	Name string

	//Heartbeat timer
	maxTime time.Duration
}

type clientChecker struct {
	clients  chan *timer
	outbound chan *net.UDPAddr
	Inbound  chan bool
	revive   chan string
	Shutdown chan struct{}
}

func NewClientChecker(outbound chan *net.UDPAddr, revive chan string, toCheck ...*timer) *clientChecker {
	clients := make(chan *timer, len(toCheck))
	for _, client := range toCheck {
		clients <- client
	}
	return &clientChecker{
		clients,
		outbound,
		make(chan bool, 1),
		revive,
		make(chan struct{}, 1),
	}
}

func (cc *clientChecker) RunChecker(group *sync.WaitGroup) {
	go func() {
		group.Add(1)
	loop:
		for {
			select {
			case client := <-cc.clients:
				alive := client.executeTimer(cc.outbound, cc.Inbound, cc.Shutdown)
				if !alive {
					logrus.Infof("action: reviving %s", client.Name)
					select {
					case cc.revive <- client.Name:
						logrus.Infof("action: reviving %s | status: request sent", client.Name)
					case <-cc.Shutdown:
						break loop
					}
				}
				cc.clients <- client
			case <-cc.Shutdown:
				logrus.Info("Recieved shutdown")
				break loop
			}
		}
		logrus.Info("action: checker | status: finishing")
		group.Done()
	}()

}

func NewTimer(at string, name string) *timer {
	t := new(timer)
	t.clientAddr = at
	t.Name = name
	t.maxTime = time.Millisecond * 20

	return t
}

func (t *timer) resolveAddr() (addr *net.UDPAddr, resolved bool) {
	addr, err := net.ResolveUDPAddr("udp", t.clientAddr)
	if err != nil {
		resolved = false
	}
	return
}

func (t *timer) executeTimer(outbound chan<- *net.UDPAddr, inbound <-chan bool, shutdown chan struct{}) (ok bool) {
	clientAddr, resolved := t.resolveAddr()
	if !resolved {
		return resolved
	}
	select {
	case outbound <- clientAddr:
	case <-shutdown:
		shutdown <- struct{}{}
		return true
	}
	retries := 3
	for retries > 0 {
		time := time.After(t.maxTime)
		select {
		case <-inbound:
			return true
		case <-time:
			retries--
		case <-shutdown:
			shutdown <- struct{}{}
			return true
		}
	}
	return false
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
		addr,
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
func (b *BeaterServer) initiateTimers(port string, dockerChan chan string) (*clientChecker, chan *net.UDPAddr) {
	timers := []*timer{}
	timersChan := make(chan *net.UDPAddr, 1)
	for _, value := range b.clientInfo {

		timer := NewTimer(value.Addr+":"+port, value.Name)

		b.clients[value.Name] = timer
		timers = append(timers, timer)
	}
	return NewClientChecker(timersChan, dockerChan, timers...), timersChan
}

/*
Initiates socket reading routine
*/
func (b *BeaterServer) initiateReader(close <-chan struct{}) chan []byte {
	readerChan := make(chan []byte, 1)
	b.wg.Add(1)
	go func() {
		var err error
		for err == nil {
			beat, _, err_read := utils.SafeReadFrom(b.sckt)
			err = err_read
			if err == nil {
				select {
				case <-close:
					logrus.Info("action: reader | status: ending")
					b.wg.Done()
					return
				case readerChan <- beat:
				}
			}
		}
		logrus.Info("action: reader | status: ending")
		b.wg.Done()
	}()
	return readerChan
}

func (b *BeaterServer) parseBeat(beat []byte, timerChan chan bool) {
	if len(beat) == 0 || beat[0] != Ok {
		return
	}
	ok := &ok{}
	err := ok.deserialize(beat[1:])
	if err != nil {
		logrus.Errorf("error while deserializing beat response: %s", err)
		return
	}
	if _, ok := b.clients[ok.whom]; ok {
		timerChan <- true
	}
}

/*
Runs a routine that is in charge of writing to the socket
heartbeats
*/
func (b *BeaterServer) writeRoutine(channel <-chan *net.UDPAddr, closeChan <-chan struct{}) {
	b.wg.Add(1)
loop:
	for {
		select {
		case <-closeChan:
			break loop
		case addr, more := <-channel:
			if !more {
				break loop
			}
			err := utils.SafeWriteTo(heartbeat{}.serialize(), b.sckt, addr)
			if err != nil {
				logrus.Errorf("action: heartbeat to services | status: %s", err)
				break loop
			}
		}
	}
	logrus.Info("action: write routine | status: finishing")
	b.wg.Done()
}

func (b *BeaterServer) run(port string) (err error) {
	dockerChan, closeDockerChan := b.dood.StartIncoming()
	checker, timersChan := b.initiateTimers(port, dockerChan)
	checker.RunChecker(b.wg)
	readerCloseChan := make(chan struct{}, 1)
	readerChan := b.initiateReader(readerCloseChan)
	writerCloseChan := make(chan struct{}, 1)
	go b.writeRoutine(timersChan, writerCloseChan)
	shutdown := make(chan struct{}, 1)
	defer close(shutdown)
	defer close(dockerChan)
	defer close(readerChan)
	defer close(timersChan)
	b.shutdown = shutdown
	b.running = true
loop:
	for {
		select {
		case beat := <-readerChan:
			b.parseBeat(beat, checker.Inbound)
		case <-shutdown:
			b.running = false
			break loop

		}
	}
	logrus.Info("action: stopping beater server | status: stopping workers, dockerChan")
	closeDockerChan <- struct{}{}
	logrus.Info("action: stopping beater server | status: stopping workers, checker")
	checker.Shutdown <- struct{}{}
	logrus.Info("action: stopping beater server | status: stopping workers, reader")
	readerCloseChan <- struct{}{}
	logrus.Info("action: stopping beater server | status: stopping workers, writer")
	writerCloseChan <- struct{}{}
	logrus.Info("action: stopping beater server | status: stopping workers | info: reader and writer told to stop")
	logrus.Info("action: stopping beater server | status: waiting workers")
	b.wg.Wait()
	logrus.Info("action: stopping beater server | status: workers stoped")
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
		logrus.Info("action: stoping beater server")
		b.shutdown <- struct{}{}
		err = b.sckt.Close()
		logrus.Info("action: stoping beater server | status: waiting on results chan")
		err = errors.Join(err, <-b.resultsChan)
		b.dood.Shutdown()
		b.running = false
		logrus.Info("action: stoping beater server | status: stopped")
	}

	return err
}
