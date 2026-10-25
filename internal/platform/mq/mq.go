package mq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const Exchange = "wayposts"

const (
	DeadLetterExchange = "wayposts.dlx"
	DeadLetterQueue    = "fanout.dlq"
)

func QueueName(shard int) string {
	return fmt.Sprintf("fanout.%d", shard)
}

func Shard(authorID int64, shards int) int {
	if shards <= 0 {
		shards = 1
	}
	shard := int(authorID % int64(shards))
	if shard < 0 {
		shard += shards
	}
	return shard
}

type Publisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := amqp.DialConfig(url, amqp.Config{Heartbeat: 10 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("mq: dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("mq: channel: %w", err)
	}
	if err := ch.ExchangeDeclare(Exchange, "direct", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("mq: declare exchange: %w", err)
	}
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("mq: confirm mode: %w", err)
	}

	return &Publisher{conn: conn, ch: ch}, nil
}

func (p *Publisher) Publish(ctx context.Context, routingKey string, body []byte) error {
	confirmation, err := p.ch.PublishWithDeferredConfirmWithContext(
		ctx,
		Exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("mq: publish: %w", err)
	}
	ok, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("mq: confirm: %w", err)
	}
	if !ok {
		return fmt.Errorf("mq: publish não confirmado")
	}
	return nil
}

func (p *Publisher) Close() {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
}

// EnsureTopology declara as filas sharded no canal do publisher. Sem isso, um
// post publicado antes de o fanout subir seria confirmado pelo exchange e
// descartado por falta de binding.
func (p *Publisher) EnsureTopology(shards int) error {
	return declareFanoutTopology(p.ch, shards)
}

func DeclareFanout(conn *amqp.Connection, shards int) (*amqp.Channel, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("mq: channel: %w", err)
	}
	if err := declareFanoutTopology(ch, shards); err != nil {
		_ = ch.Close()
		return nil, err
	}
	return ch, nil
}

func declareFanoutTopology(ch *amqp.Channel, shards int) error {
	if err := ch.ExchangeDeclare(Exchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("mq: declare exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(DeadLetterExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("mq: declare dlx: %w", err)
	}
	if _, err := ch.QueueDeclare(DeadLetterQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("mq: declare dlq: %w", err)
	}
	if err := ch.QueueBind(DeadLetterQueue, DeadLetterQueue, DeadLetterExchange, false, nil); err != nil {
		return fmt.Errorf("mq: bind dlq: %w", err)
	}

	queueArgs := amqp.Table{
		"x-dead-letter-exchange":    DeadLetterExchange,
		"x-dead-letter-routing-key": DeadLetterQueue,
	}
	for i := 0; i < shards; i++ {
		q := QueueName(i)
		if _, err := ch.QueueDeclare(q, true, false, false, false, queueArgs); err != nil {
			return fmt.Errorf("mq: declare queue %s: %w", q, err)
		}
		if err := ch.QueueBind(q, q, Exchange, false, nil); err != nil {
			return fmt.Errorf("mq: bind queue %s: %w", q, err)
		}
	}
	return nil
}

func Dial(url string) (*amqp.Connection, error) {
	conn, err := amqp.DialConfig(url, amqp.Config{Heartbeat: 10 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("mq: dial: %w", err)
	}
	return conn, nil
}
