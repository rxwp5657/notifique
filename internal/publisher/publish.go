package publisher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	c "github.com/notifique/controllers"
)

type PriorityQueues struct {
	Low    *string
	Medium *string
	High   *string
}

type Publisher interface {
	PublishMsg(ctx context.Context, queue string, message []byte) error
}

type Priority string

const (
	LOW    Priority = "LOW"
	MEDIUM Priority = "MEDIUM"
	HIGH   Priority = "HIGH"
)

func publishByPriority(ctx context.Context, notification c.Notification, storage c.NotificationStorage, publisher Publisher, queues PriorityQueues) error {

	var queueUri *string = nil

	switch notification.Priority {
	case string(LOW):
		queueUri = queues.Low
	case string(MEDIUM):
		queueUri = queues.Medium
	case string(HIGH):
		queueUri = queues.High
	default:
		return fmt.Errorf("invalid priority")
	}

	if queueUri == nil {
		return fmt.Errorf("queue for %s priority not found", notification.Priority)
	}

	message, err := json.Marshal(notification)

	if err != nil {
		return fmt.Errorf("failed to marshall message body - %w", err)
	}

	err = publisher.PublishMsg(ctx, *queueUri, message)

	if err != nil {
		errMsg := err.Error()
		statuslogErr := storage.CreateNotificationStatusLog(ctx, notification.Id, c.PUBLISH_FAILED, &errMsg)

		if statuslogErr != nil {
			errs := errors.Join(err, statuslogErr)
			return fmt.Errorf("failed to publish and create notification status log %w - ", errs)
		}

		return fmt.Errorf("failed to publish notification - %w", err)
	}

	err = storage.CreateNotificationStatusLog(ctx, notification.Id, c.PUBLISHED, nil)

	if err != nil {
		return fmt.Errorf("failed to create notification status log - %w", err)
	}

	return nil
}
