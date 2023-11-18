package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware/id"
	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

const MaxMessageSize = 8192
const ControlRoutingKey = "control"

var ErrMiddleware = errors.New("rabbitMQ channel closed")

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

	controlCount map[string]int
}

func Dial(url string) (*Middleware, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Middleware{
		conn:         conn,
		ch:           ch,
		controlCount: make(map[string]int),
	}, nil
}

func (m *Middleware) SetExpectedControlCount(queue string, count int) {
	m.controlCount[queue] = count
}

func (m *Middleware) ExchangeDeclare(name string) (string, error) {
	return name, m.ch.ExchangeDeclare(
		name,     // name
		"direct", // type
		true,     // durable
		false,    // delete when unused
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
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

func (m *Middleware) QueueBind(queue, exchange string, routingKeys []string) error {
	for _, key := range routingKeys {
		err := m.ch.QueueBind(
			queue,
			key,
			exchange,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("Failed to bind exchange to queue with routing key '%s': %w", key, err)
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
			cc int
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
				log.Infof("action: new_client | result: success | queue: %q | client: %x", name, clientId)
				ch := make(chan Delivery)
				pair.cc = cc
				pair.ch = ch
				pairs[clientId] = pair
				ret <- Client{clientId, ch}
			}
			if strings.HasPrefix(d.RoutingKey, ControlRoutingKey) {
				pair.cc--
				pairs[clientId] = pair
				if len(msg) > 0 && msg[0] > 1 {
					// A shared queue must have the same
					// name as the exchange it's bound to.
					if err := m.SharedQueueEOF(ctx, name, clientId, msg[0]-1); err != nil {
						log.Errorf("action: propagate_eof | result: failure | queue: %q | client: %x | error: %s", name, clientId, err)
						return
					}
				}
				// TODO: store EOF state
				if err := m.Ack(d.DeliveryTag); err != nil {
					log.Errorf("action: ack_eof | result: failure | queue: %q | client: %x | error: %s", name, clientId, err)
					return
				}
				if pair.cc <= 0 {
					log.Infof("action: EOF | result: success | queue: %q | client: %x", name, clientId)
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
	}(m.controlCount[name])

	return ret, nil
}

func (m *Middleware) Publish(ctx context.Context, exchange, key string, body []byte) error {
	return m.ch.PublishWithContext(
		ctx,
		exchange, // exchange
		key,      // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType: "application/octet-stream",
			Body:        body,
		},
	)
}

func (m *Middleware) WaitReady(ctx context.Context, name string, workers int) error {
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
		return err
	}

	for d := range msgs {
		workers--
		if workers <= 0 {
			return m.ch.Ack(d.DeliveryTag, true)
		}
	}
	return ErrMiddleware
}

func (m *Middleware) Ready(ctx context.Context, exchange string) error {
	return m.Publish(ctx, exchange, ControlRoutingKey, nil)
}

func (m *Middleware) EOF(ctx context.Context, exchange, clientId string) error {
	log.Infof("sending EOF into exchange %q", exchange)
	return m.Publish(ctx, exchange, ControlRoutingKey, []byte(clientId))
}

func (m *Middleware) TopicEOF(ctx context.Context, exchange, topic, clientId string) error {
	log.Infof("sending EOF into exchange %q for topic %q", exchange, topic)
	rKey := ControlRoutingKey + "." + topic
	return m.Publish(ctx, exchange, rKey, []byte(clientId))
}

func (m *Middleware) SharedQueueEOF(ctx context.Context, name, clientId string, eof byte) error {
	log.Infof("sending EOF(%d) into queue %q", eof, name)
	return m.Publish(ctx, name, ControlRoutingKey, append([]byte(clientId), eof))
}

func (m *Middleware) Close() {
	// the corresponding Channel is closed along with the Connection
	m.conn.Close()
	log.Info("closed rabbitMQ Connection")
}
