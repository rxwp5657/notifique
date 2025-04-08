package deploy

import (
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQAPI interface {
	QueueDeclare(name string, durable bool, autoDelete bool, exclusive bool, noWait bool, args amqp091.Table) (amqp091.Queue, error)
}

func RabbitMQQueue(client RabbitMQAPI, queueName string) error {

	_, err := client.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)

	if err != nil {
		return fmt.Errorf("failed to create queue %s - %w", queueName, err)

	}

	return nil
}
