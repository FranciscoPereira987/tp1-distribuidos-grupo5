package middleware

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware/id"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

const MaxMessageSize = 8192
const EofRoutingKey = "eof"
const Workdir = "middleware"

var (
	ErrMiddleware = errors.New("rabbitMQ channel closed")
	ErrNack       = errors.New("server nack'ed delivery")
)

type Client struct {
	Id string
	Ch <-chan Delivery
}

type Delivery struct {
	Msg []byte
	Tag uint64
}

type Middleware struct {
	conn *amqp.Connection
	ch   *amqp.Channel

	eofCount map[string]int
}

type Confirmer interface {
	Confirm(context.Context) error
	AddWithContext(context.Context, *amqp.DeferredConfirmation) error
}

type DeferredConfirmer struct {
	newConfirms  chan<- *amqp.DeferredConfirmation
	waitConfirms chan<- struct{}
	errs         <-chan error
}

func Dial(url string) (*Middleware, error) {
	if err := os.MkdirAll(Workdir, 0755); err != nil {
		return nil, err
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := ch.Confirm(false); err != nil {
		conn.Close()
		return nil, err
	}

	return &Middleware{
		conn:     conn,
		ch:       ch,
		eofCount: make(map[string]int),
	}, nil
}

func (m *Middleware) NewDeferredConfirmer(ctx context.Context) DeferredConfirmer {
	newConfirms := make(chan *amqp.DeferredConfirmation)
	waitConfirms := make(chan struct{})
	errs := make(chan error)

	go func() {
		var confirmations []*amqp.DeferredConfirmation
		for {
			select {
			case dc := <-newConfirms:
				confirmations = append(confirmations, dc)
			case <-waitConfirms:
				for i, dc := range confirmations {
					if ack, err := dc.WaitContext(ctx); err != nil {
						errs <- err
						return
					} else if !ack {
						errs <- fmt.Errorf("%w: tag=%d", ErrNack, dc.DeliveryTag)
						return
					}
					confirmations[i] = nil
				}
				errs <- nil
				confirmations = confirmations[:0]
			case <-ctx.Done():
				log.Info("context cancelled, stopping confirmer")
				return
			}
		}
	}()

	return DeferredConfirmer{newConfirms, waitConfirms, errs}
}

func (c DeferredConfirmer) Confirm(ctx context.Context) error {
	select {
	case c.waitConfirms <- struct{}{}:
	case <-ctx.Done():
		return context.Cause(ctx)
	}

	select {
	case err := <-c.errs:
		return err
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

func (c DeferredConfirmer) AddWithContext(ctx context.Context, dc *amqp.DeferredConfirmation) error {
	select {
	case c.newConfirms <- dc:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

func (m *Middleware) SetExpectedEofCount(queue string, count int) {
	m.eofCount[queue] = count
}

// Declares a `fanout' exchange. These are meant for EOF or `fanout' messages.
// If you need to send a message directly to a queue use the default exchange
// with the desired queue's name as routing key.
func (m *Middleware) ExchangeDeclare(name string) (string, error) {
	return name, m.ch.ExchangeDeclare(
		name,     // name
		"fanout", // type
		true,     // durable
		false,    // delete when unused
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
}

func QueueName(worker, id string) (string, error) {
	if worker == "" {
		return "", fmt.Errorf("middleware.QueueName: %w for 'worker'", utils.ErrMissingConfig)
	} else if id == "" {
		return "", fmt.Errorf("middleware.QueueName: %w for 'id'", utils.ErrMissingConfig)
	}
	return worker + "." + id, nil
}

func (m *Middleware) QueueDeclare(name string) (string, error) {
	q, err := m.ch.QueueDeclare(
		name,  // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return "", err
	}

	return q.Name, err
}

// Bind the specified `fanout' exchanges to the given queue.
func (m *Middleware) QueueBind(queue string, exchanges ...string) error {
	for _, exchange := range exchanges {
		err := m.ch.QueueBind(
			queue,    // queue
			"",       // routing key
			exchange, // exchange
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("Failed to bind %q exchange to %q queue: %w", exchange, queue, err)
		}
	}
	return nil
}

func (m *Middleware) Ack(tag uint64) error {
	return m.ch.Ack(tag, false)
}

func (m *Middleware) Consume(ctx context.Context, name string) (<-chan Client, error) {
	msgs, err := m.ch.ConsumeWithContext(
		ctx,
		name,  // queue
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return nil, err
	}

	ret := make(chan Client)
	go func(cc int) {
		pairs := make(map[string]struct {
			sm *state.StateManager
			ch chan<- Delivery
		})
		defer func() {
			close(ret)
			for _, p := range pairs {
				close(p.ch)
			}
		}()

		for d := range msgs {
			clientId, msg := string(d.Body[:id.Len]), d.Body[id.Len:]
			pair, ok := pairs[clientId]
			if !ok {
				ch := make(chan Delivery)
				dir := filepath.Join(Workdir, name, hex.EncodeToString(d.Body[:id.Len]))
				pair.sm = state.NewStateManager(dir)
				pair.ch = ch
				pairs[clientId] = pair
				if err := os.MkdirAll(dir, 0755); err != nil {
					log.Errorf("action: new_client | result: failure | queue: %q | client: %x | error: %s", name, clientId, err)
					return
				} else {
					log.Infof("action: new_client | result: success | queue: %q | client: %x", name, clientId)
					pair.sm.RecoverState()
				}
				ret <- Client{clientId, ch}
			}
			if d.RoutingKey == EofRoutingKey {
				pair.sm.State[string(msg)] = true
				if err := pair.sm.DumpState(); err != nil {
					log.Errorf("action: store_eof | result: failure | queue: %q | worker: %s | error: %s", name, clientId, err)
					return
				}
				if err := m.Ack(d.DeliveryTag); err != nil {
					log.Errorf("action: ack_eof | result: failure | queue: %q | client: %x | error: %s", name, clientId, err)
					return
				}
				if len(pair.sm.State) >= cc {
					log.Infof("action: EOF | result: success | queue: %q | client: %x", name, clientId)
					os.RemoveAll(filepath.Join(Workdir, name, hex.EncodeToString([]byte(clientId))))
					close(pair.ch)
					delete(pairs, clientId)
				} else {
					log.Infof("action: EOF | result: in_progress | queue: %q | client: %x", name, clientId)
				}
			} else {
				pair.ch <- Delivery{msg, d.DeliveryTag}
			}
		}
		log.Error(ErrMiddleware)
	}(m.eofCount[name])

	return ret, nil
}

func (m *Middleware) Publish(ctx context.Context, c Confirmer, exchange, key string, body []byte) error {
	dc, err := m.ch.PublishWithDeferredConfirmWithContext(
		ctx,
		exchange, // exchange
		key,      // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/octet-stream",
			Body:         body,
		},
	)
	if err != nil {
		return err
	}

	return c.AddWithContext(ctx, dc)
}

func (m *Middleware) WaitReady(ctx context.Context, name string, workers int) error {
	msgs, err := m.ch.ConsumeWithContext(
		ctx,
		name,    // queue
		"ready", // consumer
		false,   // auto-ack
		false,   // exclusive
		false,   // no-local
		false,   // no-wait
		nil,     // args
	)
	if err != nil {
		return err
	}

	for d := range msgs {
		workers--
		if workers <= 0 {
			if err := m.ch.Ack(d.DeliveryTag, true); err != nil {
				return err
			}
		}
		if workers == 0 {
			if err := m.ch.Cancel("ready", false); err != nil {
				return err
			}
		}
	}

	if workers > 0 {
		return ErrMiddleware
	} else {
		return nil
	}
}

type BasicConfirmer struct {
	dc *amqp.DeferredConfirmation
}

func (bc BasicConfirmer) Confirm(ctx context.Context) error {
	ack, err := bc.dc.WaitContext(ctx)
	if err == nil && !ack {
		err = fmt.Errorf("%w: tag=%d", ErrNack, bc.dc.DeliveryTag)
	}
	return err
}

func (bc *BasicConfirmer) AddWithContext(ctx context.Context, dc *amqp.DeferredConfirmation) error {
	bc.dc = dc
	return nil
}

func (bc *BasicConfirmer) Publish(ctx context.Context, m *Middleware, exchange, key string, body []byte) error {
	if err := m.Publish(ctx, bc, exchange, key, body); err != nil {
		return err
	}
	return bc.Confirm(ctx)
}

func (m *Middleware) Ready(ctx context.Context, key string) error {
	var bc BasicConfirmer
	return bc.Publish(ctx, m, "", key, nil)
}

func (m *Middleware) EOF(ctx context.Context, exchange, workerId, clientId string) error {
	var bc BasicConfirmer
	log.Infof("sending EOF into exchange %q", exchange)
	msg := append([]byte(clientId), workerId...)
	return bc.Publish(ctx, m, exchange, EofRoutingKey, msg)
}

func (m *Middleware) Close() {
	// the corresponding Channel is closed along with the Connection
	m.conn.Close()
	log.Info("closed rabbitMQ Connection")
}
