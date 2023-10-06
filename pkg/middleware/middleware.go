package middleware

import (
	"binary"
	"bytes"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"

	amqp "github.com/rabbitmq/ampq091-go"
)

type Middleware struct {
	conn *amqp.Connection
	ch   *amqp.Channel
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
		err = m.ch.QueueBind(
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

func (m *Middleware) Consume(ctx context.Context, name string) (<-chan []byte, error) {
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
				if d.RoutingKey != "control" {
					ch <- d.Body
					break
				}
				if err := m.propagateEOF(ctx, d); err != nil {
					log.Error(err)
				}
				return
			}
		}
	}()

	return ch, nil
}

func (m *Middleware) propagateEOF(ctx context.Context, d amqp.Delivery) error {
	if len(d.Body) == 0 {
		return nil
	}
	n := binary.Uvarint(d.Body)
	if n < 2 {
		return nil
	}
	body := binary.PutUvarint(d.Body, n-1)
	return m.PublishWithContext(ctx, d.Exchange, d.RoutingKey, body)
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

func KeyFrom(origin, destiny string) string {
	// TODO: use a hash to be able to shard ahead of time
	return origin + "." + destiny
}