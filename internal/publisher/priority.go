package publisher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	c "github.com/notifique/controllers"
	conv "github.com/notifique/internal/convertors"
)

type Priority string

const (
	low    Priority = "LOW"
	medium Priority = "MEDIUM"
	high   Priority = "HIGH"
)

type Publisher interface {
	Publish(ctx context.Context, queueName string, message []byte) error
}

type PriorityQueueConfigurator interface {
	GetPriorityQueues() PriorityQueues
}

type PriorityQueues struct {
	Low    *string
	Medium *string
	High   *string
}

type PriorityPublisher struct {
	publisher Publisher
	storage   c.NotificationStorage
	queues    PriorityQueues
}

func (p PriorityPublisher) Publish(ctx context.Context, n c.Notification) error {

	var queueUri *string = nil

	switch n.Priority {
	case string(low):
		queueUri = p.queues.Low
	case string(medium):
		queueUri = p.queues.Medium
	case string(high):
		queueUri = p.queues.High
	default:
		return fmt.Errorf("invalid priority")
	}

	if queueUri == nil {
		return fmt.Errorf("queue for %s priority not found", n.Priority)
	}

	message, err := json.Marshal(n)

	if err != nil {
		return fmt.Errorf("failed to marshall message body - %w", err)
	}

	err = p.publisher.Publish(ctx, *queueUri, message)

	if err != nil {
		errMsg := err.Error()

		statusLogs := conv.MakeStatusLogs(n, c.PublishFailed, &errMsg)
		statuslogErr := p.storage.CreateNotificationStatusLog(ctx, statusLogs...)

		if statuslogErr != nil {
			errs := errors.Join(err, statuslogErr)
			return fmt.Errorf("failed to publish and create notification status log %w - ", errs)
		}

		return fmt.Errorf("failed to publish notification - %w", err)
	}

	statusLogs := conv.MakeStatusLogs(n, c.Published, nil)
	err = p.storage.CreateNotificationStatusLog(ctx, statusLogs...)

	if err != nil {
		return fmt.Errorf("failed to create notification status log - %w", err)
	}

	return nil
}

func NewPriorityPublisher(p Publisher, c PriorityQueueConfigurator, s c.NotificationStorage) *PriorityPublisher {

	return &PriorityPublisher{
		publisher: p,
		storage:   s,
		queues:    c.GetPriorityQueues(),
	}
}
