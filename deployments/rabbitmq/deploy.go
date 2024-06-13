package deployments

import (
	"fmt"

	"github.com/notifique/internal/publisher"
)

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

func MakeRabbitMQPriorityQueues(client publisher.RabbitMQClient, queues publisher.PriorityQueues) error {

	makeQueueIfSupplied := func(name *string) error {
		if name == nil {
			return nil
		}

		return createQueue(client, *name)
	}

	if err := makeQueueIfSupplied(queues.Low); err != nil {
		return err
	}

	if err := makeQueueIfSupplied(queues.Medium); err != nil {
		return err
	}

	if err := makeQueueIfSupplied(queues.High); err != nil {
		return err
	}

	return nil
}
