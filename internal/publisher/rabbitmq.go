package publisher

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	c "github.com/notifique/controllers"
)

type RabbitMQURL string

type RabbitMQAPI interface {
	PublishWithContext(_ context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

type RabbitMQClient struct {
	*amqp.Channel
	conn  *amqp.Connection
	Close func() error
}

type RabbitMQPublisher struct {
	ch RabbitMQAPI
}

type RabbitMQPriorityPublisher struct {
	publisher RabbitMQPublisher
	queues    PriorityQueues
}

type RabbitMQConfigurator interface {
	GetRabbitMQUrl() (string, error)
}

func (p *RabbitMQPriorityPublisher) Publish(ctx context.Context, n c.Notification, s c.NotificationStorage) error {
	return publishByPriority(ctx, n, s, &p.publisher, p.queues)
}

func (p *RabbitMQPublisher) PublishMsg(ctx context.Context, queueName string, message []byte) error {
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

func MakeRabbitMQClient(c RabbitMQConfigurator) (*RabbitMQClient, error) {

	url, err := c.GetRabbitMQUrl()

	if err != nil {
		return nil, err
	}

	conn, err := amqp.Dial(string(url))

	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq - %w", err)
	}

	ch, err := conn.Channel()

	if err != nil {
		return nil, fmt.Errorf("failed to create channel - %w", err)
	}

	close := func() error {

		if err := ch.Close(); err != nil {
			return fmt.Errorf("failed to close rabbitmq channel - %w", err)
		}

		if err := conn.Close(); err != nil {
			return fmt.Errorf("failed to close rabbitmq connection - %w", err)
		}

		return nil
	}

	client := RabbitMQClient{Channel: ch, conn: conn, Close: close}

	return &client, nil
}

func MakeRabbitMQPriorityPub(p RabbitMQAPI, c PriorityQueueConfigurator) *RabbitMQPriorityPublisher {

	pub := RabbitMQPublisher{ch: p}

	ppub := RabbitMQPriorityPublisher{
		publisher: pub,
		queues:    c.GetPriorityQueues(),
	}

	return &ppub
}
