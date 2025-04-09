package containers_test

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/notifique/shared/clients"
	sc "github.com/notifique/shared/containers"
	"github.com/notifique/shared/deploy"
	"github.com/notifique/worker/internal/consumers"
)

const testQueue = "notifique-test"

type RabbitMQConsumerContainer struct {
	container *sc.RabbitMQ
	Queue     consumers.RabbitMQQueue
	Client    *clients.RabbitMQ
}

type SQSConsumerContainer struct {
	container *sc.SQS
	Client    *sqs.Client
	QueueCfg  consumers.SQSQueueCfg
}

func NewRabbitMQConsumerContainer(ctx context.Context) (*RabbitMQConsumerContainer, func(), error) {

	container, closeContainer, err := sc.NewRabbitMQContainer(ctx)

	if err != nil {
		return nil, nil, nil
	}

	client, closeClient, err := clients.NewRabbitMQClient(&container)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create RabbitMQ client - %w", err)
	}

	consumer := RabbitMQConsumerContainer{
		container: &container,
		Queue:     testQueue,
		Client:    client,
	}

	err = deploy.RabbitMQQueue(client, testQueue)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to deploy test queue - %w", err)
	}

	close := func() {
		closeContainer()
		closeClient()
	}

	return &consumer, close, nil
}

func NewSQSConsumerContainer(ctx context.Context) (*SQSConsumerContainer, func(), error) {

	container, closeContainer, err := sc.NewSQSContainer(ctx)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create SQS container - %w", err)
	}

	client, err := clients.NewSQSClient(&container)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create SQS client - %w", err)
	}

	queueURL, err := deploy.SQSQueue(client, testQueue)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to deploy test queue - %w", err)
	}

	queueCfg := consumers.SQSQueueCfg{
		QueueURL:            queueURL,
		MaxNumberOfMessages: 1,
		WaitTimeSeconds:     1,
	}

	consumer := SQSConsumerContainer{
		container: &container,
		Client:    client,
		QueueCfg:  queueCfg,
	}

	return &consumer, closeContainer, nil
}
