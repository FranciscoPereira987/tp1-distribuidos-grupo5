package middleware

import (
	"context"
	"fmt"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Middleware struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	countEOF int
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
		conn: conn,
		ch:   ch,
	}, nil
}

func (m *Middleware) SetExpectedEOFCount(count int) {
	m.countEOF = count
}

func (m *Middleware) ExchangeDeclare(name, kind string) (string, error) {
	return name, m.ch.ExchangeDeclare(
		name,  // name
		kind,  // type
		true,  // durable
		false, // delete when unused
		false, // internal
		false, // no-wait
		nil,   // arguments
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
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case d := <-msgs:
				if strings.HasSuffix(d.RoutingKey, "control") {
					m.countEOF--
					if m.countEOF <= 0 {
						return
					}
				}
				ch <- d.Body
			}
		}
	}()

	return ch, nil
}

func (m *Middleware) EOF(ctx context.Context, exchange string) error {
	return m.PublishWithContext(ctx, exchange, "control", []byte{})
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

func (m *Middleware) Close() {
	m.ch.Close()
	m.conn.Close()
}
