package publish

import (
	"context"

	"github.com/notifique/shared/clients"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQURL string

type RabbitMQAPI interface {
	PublishWithContext(_ context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

type RabbitMQ struct {
	ch RabbitMQAPI
}

type RabbitMQPriorityConfigurator interface {
	clients.RabbitMQConfigurator
	PriorityQueueConfigurator
}

func (p *RabbitMQ) Publish(ctx context.Context, queueName string, message Message) error {
	return p.ch.PublishWithContext(
		ctx,
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         message.Payload,
			MessageId:    message.Id,
		},
	)
}

func NewRabbitMQPublisher(p RabbitMQAPI) *RabbitMQ {
	return &RabbitMQ{ch: p}
}
