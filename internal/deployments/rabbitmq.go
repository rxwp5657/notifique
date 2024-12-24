package deployments

import (
	"fmt"

	"github.com/notifique/internal/publish"
)

type RabbitMQDeployer interface {
	Deploy() error
}

type RabbitMQPriorityDeployer struct {
	Client publish.RabbitMQClient
	Queues publish.PriorityQueues
}

func (d *RabbitMQPriorityDeployer) Deploy() error {

	makeQueueIfSupplied := func(name *string) error {
		if name == nil {
			return nil
		}

		return createRabbitMQQueue(d.Client, *name)
	}

	if err := makeQueueIfSupplied(d.Queues.Low); err != nil {
		return err
	}

	if err := makeQueueIfSupplied(d.Queues.Medium); err != nil {
		return err
	}

	if err := makeQueueIfSupplied(d.Queues.High); err != nil {
		return err
	}

	return nil
}

func createRabbitMQQueue(client publish.RabbitMQClient, name string) error {

	_, err := client.QueueDeclare(
		name,  // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

	if err != nil {
		return fmt.Errorf("failed to create queue %s - %w", name, err)
	}

	return nil
}

func NewRabbitMQPriorityDeployer(c publish.RabbitMQPriorityConfigurator) (*RabbitMQPriorityDeployer, func(), error) {

	client, close, err := publish.NewRabbitMQClient(c)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create rabbitmq client - %w", err)
	}

	deployer := RabbitMQPriorityDeployer{
		Client: *client,
		Queues: c.GetPriorityQueues(),
	}

	return &deployer, func() { close() }, nil
}
