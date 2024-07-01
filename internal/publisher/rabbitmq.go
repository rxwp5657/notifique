package publisher

import (
	"context"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQURL string

type RabbitMQAPI interface {
	PublishWithContext(_ context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

type RabbitMQClient struct {
	*amqp.Channel
	conn *amqp.Connection
}

type RabbitMQPublisher struct {
	ch RabbitMQAPI
}

type RabbitMQConfigurator interface {
	GetRabbitMQUrl() (string, error)
}

type RabbitMQPriorityConfigurator interface {
	RabbitMQConfigurator
	PriorityQueueConfigurator
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, queueName string, message []byte) error {
	return p.ch.PublishWithContext(
		ctx,
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         message,
		},
	)
}

func NewRabbitMQClient(c RabbitMQConfigurator) (*RabbitMQClient, func(), error) {

	url, err := c.GetRabbitMQUrl()

	if err != nil {
		return nil, nil, err
	}

	conn, err := amqp.Dial(string(url))

	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to rabbitmq - %w", err)
	}

	ch, err := conn.Channel()

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create channel - %w", err)
	}

	close := func() {

		if err := ch.Close(); err != nil {
			slog.Error("failed to close rabbitmq channel", "reason", err)
		}

		if err := conn.Close(); err != nil {
			slog.Error("failed to close rabbitmq connection", "reason", err)
		}
	}

	client := RabbitMQClient{Channel: ch, conn: conn}

	return &client, close, nil
}

func NewRabbitMQPublisher(p RabbitMQAPI) *RabbitMQPublisher {
	return &RabbitMQPublisher{ch: p}
}
