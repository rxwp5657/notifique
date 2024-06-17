package deployments

import (
	"fmt"
	"log"

	"github.com/notifique/internal/publisher"
)

type RabbitMQDeployer interface {
	Deploy() error
}

type RabbitMQPriorityDeployer struct {
	Client publisher.RabbitMQClient
	Queues publisher.PriorityQueues
}

func (d *RabbitMQPriorityDeployer) Deploy() error {

	makeQueueIfSupplied := func(name *string) error {
		if name == nil {
			return nil
		}

		return createQueue(d.Client, *name)
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

func createQueue(client publisher.RabbitMQClient, name string) error {

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

func MakeRabbitMQPriorityDeployer(c publisher.RabbitMQPriorityConfigurator) (*RabbitMQPriorityDeployer, func(), error) {

	client, err := publisher.MakeRabbitMQClient(c)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create rabbitmq client - %w", err)
	}

	cleanup := func() {
		log.Fatal(client.Close())
	}

	deployer := RabbitMQPriorityDeployer{
		Client: *client,
		Queues: c.GetPriorityQueues(),
	}

	return &deployer, cleanup, nil
}
