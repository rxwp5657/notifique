package publisher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	c "github.com/notifique/controllers"
)

type Priority string

const (
	low    Priority = "LOW"
	medium Priority = "MEDIUM"
	high   Priority = "HIGH"
)

type PriorityQueues struct {
	Low    *string
	Medium *string
	High   *string
}

type Publisher interface {
	PublishMsg(ctx context.Context, queueName string, message []byte) error
}

type PriorityQueueConfigurator interface {
	GetPriorityQueues() PriorityQueues
}

func publishByPriority(ctx context.Context, n c.Notification, s c.NotificationStorage, p Publisher, pq PriorityQueues) error {

	var queueUri *string = nil

	switch n.Priority {
	case string(low):
		queueUri = pq.Low
	case string(medium):
		queueUri = pq.Medium
	case string(high):
		queueUri = pq.High
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

	err = p.PublishMsg(ctx, *queueUri, message)

	if err != nil {
		errMsg := err.Error()
		statuslogErr := s.CreateNotificationStatusLog(ctx, n.Id, c.PublishFailed, &errMsg)

		if statuslogErr != nil {
			errs := errors.Join(err, statuslogErr)
			return fmt.Errorf("failed to publish and create notification status log %w - ", errs)
		}

		return fmt.Errorf("failed to publish notification - %w", err)
	}

	err = s.CreateNotificationStatusLog(ctx, n.Id, c.Published, nil)

	if err != nil {
		return fmt.Errorf("failed to create notification status log - %w", err)
	}

	return nil
}
