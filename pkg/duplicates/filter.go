package duplicates

import (
	"context"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
)

type DuplicateFilterConfig struct {
	Ctx        context.Context
	Mid        *middleware.Middleware
	StreamName string
	StateFile  string
}

/*
Simple Lock made from channels
*/
type ChanLock struct {
	lock chan struct{}
	ctx  context.Context
}

func NewLock(ctx context.Context) *ChanLock {
	return &ChanLock{
		lock: make(chan struct{}, 1),
		ctx:  ctx,
	}
}

func (l *ChanLock) Lock() (err error) {
	select {
	case l.lock <- struct{}{}:
	case <-l.ctx.Done():
		err = l.ctx.Err()
	}
	return
}

func (l *ChanLock) Release() {
	<-l.lock
}

/*
Executes a routine that acts as an intermediary for a client
What it does:

 1. Reads from the middleware channel
 2. Tries to acquire the lock
 4. Sends the delivery to the worker
 5. Waits for the worker to ack the delivery
 3. Stores the state
 6. ACKs the delivery
 7. Frees the lock
*/
type deliveryRoutine struct {
	mid         *middleware.Middleware
	clientId    string
	Ch          <-chan middleware.Delivery
	lock        *ChanLock
	workerChan  chan middleware.Delivery
	closeChan   chan<- string
	ackChan     chan uint64
	fileName    string
	lastMessage []byte
}

func newDeliveryRoutine(clientId string, closeChan chan<- string, mid *middleware.Middleware, ch <-chan middleware.Delivery, lock *ChanLock, stateFile string) *deliveryRoutine {
	return &deliveryRoutine{
		mid:        mid,
		clientId:   clientId,
		Ch:         ch,
		lock:       lock,
		workerChan: make(chan middleware.Delivery),
		closeChan:  closeChan,
		ackChan:    make(chan uint64),
		fileName:   stateFile,
	}
}

func (dr *deliveryRoutine) ackFunc() func(uint64) {
	return func(tag uint64) {
		dr.ackChan <- tag
	}
}

func (dr deliveryRoutine) isDuplicate(body []byte) (dup bool) {
	dup = len(body) == len(dr.lastMessage)
	for i := 0; dup && i < len(body); i++ {
		dup = body[i] == dr.lastMessage[i]
	}
	return
}

func (dr *deliveryRoutine) runDeliveryRoutine() <-chan middleware.Delivery {

	go func() {
		defer func() {
			dr.closeChan <- dr.clientId
		}()
		defer close(dr.workerChan)
		defer close(dr.ackChan)
		for delivery := range dr.Ch {
			if dr.isDuplicate(delivery.Msg) {
				//Ack para que no envie otra vez otro repetido
				dr.mid.Ack(delivery.Tag)
				continue
			}
			if err := dr.lock.Lock(); err != nil {
				break
			}
			dr.workerChan <- delivery
			tag := <-dr.ackChan
			//Pensar si no es conveniente poner esto en el contexto y pedir por un guardado del estado
			//general aca
			state.WriteFile(dr.fileName, delivery.Msg)
			dr.mid.Ack(tag)
			dr.lastMessage = delivery.Msg
			dr.lock.Release()
		}
	}()

	return dr.workerChan
}

/*
TODO: Filter last k messages
Filters duplicates messages
*/
type DuplicateFilter struct {
	config        *DuplicateFilterConfig
	ch            <-chan middleware.Client
	mid           *middleware.Middleware
	lock          *ChanLock
	closeChan     chan string
	activeClients map[string]*deliveryRoutine
}

func FilterAt(config *DuplicateFilterConfig) (df *DuplicateFilter, err error) {
	df = new(DuplicateFilter)
	channel, err := config.Mid.Consume(config.Ctx, config.StreamName)
	if err == nil {
		df.config = config
		df.ch = channel
		df.mid = config.Mid
		df.lock = NewLock(config.Ctx)
		df.closeChan = make(chan string)
		df.activeClients = make(map[string]*deliveryRoutine)
	}
	return
}

func (df *DuplicateFilter) waitForRoutines() {
	for {
		select {
		case routine := <-df.closeChan:
			delete(df.activeClients, routine)
		default:
			break
		}
	}
}

/*
Closes the middleware connection and waits for every routine to
finish
*/
func (df *DuplicateFilter) Shutdown() {
	df.mid.Close()
	for len(df.activeClients) > 0 {
		df.waitForRoutines()
	}
}

func (df *DuplicateFilter) GetClient() (string, <-chan middleware.Delivery, func(uint64), bool) {
	queue, ok := <-df.ch
	df.waitForRoutines()
	//TODO: Pensa bien que es lo que queres que vaya aca
	//	1. El archivo puede ser unico por cliente (Pero eso implicaria guardarse el ultimo de cada cliente)
	//	2. El archivo puede ser el mismo para todos
	//	3. Necesito conocer que clientes estan activos
	dr := newDeliveryRoutine(queue.Id, df.closeChan, df.mid, queue.Ch, df.lock, df.config.StateFile)

	return queue.Id, dr.runDeliveryRoutine(), dr.ackFunc(), ok
}
