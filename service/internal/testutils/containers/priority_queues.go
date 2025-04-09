package containers

import (
	"context"
	"fmt"

	"github.com/notifique/service/internal/publish"
	"github.com/notifique/service/pkg/deployments"
	"github.com/notifique/shared/clients"
	scontainers "github.com/notifique/shared/containers"
)

const (
	LowPriorityQueue    = "notifique-low"
	MediumPriorityQueue = "notifique-medium"
	HighPriorityQueue   = "notifique-high"
)

func NewPriorityQueueConfig() (queues publish.PriorityQueues) {

	low := LowPriorityQueue
	medium := MediumPriorityQueue
	high := HighPriorityQueue

	queues.Low = &low
	queues.Medium = &medium
	queues.High = &high

	return
}

type RabbitMQPriority struct {
	Container scontainers.RabbitMQ
	Client    clients.RabbitMQ
	Queues    publish.PriorityQueues
}

func (rc *RabbitMQPriority) GetRabbitMQUrl() (string, error) {
	return rc.Container.URI, nil
}

func (rc *RabbitMQPriority) GetPriorityQueues() publish.PriorityQueues {
	return rc.Queues
}

type SQSPriority struct {
	scontainers.SQS
	Queues publish.PriorityQueues
}

func (sc *SQSPriority) GetPriorityQueues() publish.PriorityQueues {
	return sc.Queues
}

func NewRabbitMQPriorityContainer(ctx context.Context) (*RabbitMQPriority, func(), error) {

	queues := NewPriorityQueueConfig()
	container, close, err := scontainers.NewRabbitMQContainer(ctx)

	if err != nil {
		return nil, nil, nil
	}

	pc := RabbitMQPriority{
		Container: container,
		Queues:    queues,
	}

	deployer, closeDeployer, err := deployments.NewRabbitMQPriorityDeployer(&pc)

	if err != nil {
		return nil, close, fmt.Errorf("failed to make rabbitmq deployer - %w", err)
	}

	defer closeDeployer()

	deployer.Deploy()

	return &pc, close, nil
}

func NewSQSPriorityContainer(ctx context.Context) (*SQSPriority, func(), error) {
	queues := NewPriorityQueueConfig()
	container, close, err := scontainers.NewSQSContainer(ctx)

	if err != nil {
		return nil, nil, err
	}

	pc := SQSPriority{
		SQS:    container,
		Queues: queues,
	}

	deployer, err := deployments.NewSQSPriorityDeployer(&pc)

	if err != nil {
		return nil, close, err
	}

	deployedQueues, err := deployer.Deploy()

	if err != nil {
		return nil, close, err
	}

	pc.Queues = deployedQueues

	return &pc, close, nil
}
