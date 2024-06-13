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

type RabbitMQPriorityPublisherConfg struct {
	Publisher RabbitMQAPI
	Queues    PriorityQueues
}

func (p *RabbitMQPriorityPublisher) Publish(ctx context.Context, notification c.Notification, storage c.NotificationStorage) error {
	return publishByPriority(ctx, notification, storage, &p.publisher, p.queues)
}

func (p *RabbitMQPublisher) PublishMsg(ctx context.Context, queue string, message []byte) error {
	return p.ch.PublishWithContext(
		ctx,
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         message,
		},
	)
}

func MakeRabbitMQClient(url RabbitMQURL) (client RabbitMQClient, err error) {

	conn, err := amqp.Dial(string(url))

	if err != nil {
		return client, fmt.Errorf("failed to connect to rabbitmq - %w", err)
	}

	ch, err := conn.Channel()

	if err != nil {
		return client, fmt.Errorf("failed to create channel - %w", err)
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

	client = RabbitMQClient{Channel: ch, conn: conn, Close: close}

	return client, nil
}

func MakeRabbitMQPriorityPub(cfg RabbitMQPriorityPublisherConfg) RabbitMQPriorityPublisher {
	pub := RabbitMQPublisher{ch: cfg.Publisher}
	ppub := RabbitMQPriorityPublisher{publisher: pub, queues: cfg.Queues}

	return ppub
}
