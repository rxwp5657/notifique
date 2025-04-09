package clients

import (
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	*amqp.Channel
	conn *amqp.Connection
}

type RabbitMQConfigurator interface {
	GetRabbitMQUrl() (string, error)
}

func NewRabbitMQClient(c RabbitMQConfigurator) (*RabbitMQ, func(), error) {

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

	client := RabbitMQ{Channel: ch, conn: conn}

	return &client, close, nil
}
