package middleware

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

const ControlRoutingKey = "control"

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
	var durable, exclusive bool
	if name == "" {
		exclusive = true
	} else {
		durable = true
	}
	q, err := m.ch.QueueDeclare(
		name,      // name
		durable,   // durable
		false,     // delete when unused
		exclusive, // exclusive
		false,     // no-wait
		nil,       // arguments
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

func (m *Middleware) ConsumeWithContext(ctx context.Context, name string) (<-chan []byte, error) {
	msgs, err := m.ch.ConsumeWithContext(
		ctx,
		name,  // queue
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return nil, err
	}

	ch := make(chan []byte)
	go func(cc int) {
		defer close(ch)
		for d := range msgs {
			if d.RoutingKey == ControlRoutingKey {
				logrus.Info("recieved control message")
				cc--
				// The shared queue needs to have the same name
				// as the exchange it's bound to.
				if len(d.Body) > 0 && d.Body[0] > 1 {
					m.SharedQueueEOF(ctx, name, d.Body[0]-1)
				}
				if cc <= 0 {
					return
				}
			} else {
				ch <- d.Body
			}
		}
		logrus.Error("rabbitmq channel closed")
	}(m.controlCount[name])

	return ch, nil
}

func (m *Middleware) PublishWithContext(ctx context.Context, exchange, key string, body []byte) error {
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

func (m *Middleware) Control(ctx context.Context, exchange string) error {
	return m.PublishWithContext(ctx, exchange, ControlRoutingKey, nil)
}

func (m *Middleware) EOF(ctx context.Context, exchange string) error {
	return m.Control(ctx, exchange)
}

func (m *Middleware) SharedQueueEOF(ctx context.Context, name string, eof byte) error {
	return m.PublishWithContext(ctx, name, ControlRoutingKey, []byte{eof})
}

func (m *Middleware) Close() {
	m.ch.Close()
	m.conn.Close()
}
