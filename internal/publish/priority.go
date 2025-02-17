package publish

import (
	"context"
	"encoding/json"
	"fmt"

	c "github.com/notifique/internal/server/controllers"
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
	registry  c.NotificationRegistry
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

	publishErr := p.publisher.Publish(ctx, *queueUri, message)

	status := c.Queued

	var errorMsg *string

	if publishErr != nil {
		status = c.Failed
		errStr := publishErr.Error()
		errorMsg = &errStr
	}

	statusLog := c.NotificationStatusLog{
		NotificationId: n.Id,
		Status:         status,
		ErrorMsg:       errorMsg,
	}

	if logErr := p.registry.UpdateNotificationStatus(ctx, statusLog); logErr != nil {
		if publishErr != nil {
			return fmt.Errorf("publish failed (%v) and status update failed: %w", publishErr, logErr)
		}
		return fmt.Errorf("failed to update notification status: %w", logErr)
	}

	return publishErr
}

func NewPriorityPublisher(p Publisher, c PriorityQueueConfigurator, s c.NotificationRegistry) *PriorityPublisher {

	return &PriorityPublisher{
		publisher: p,
		registry:  s,
		queues:    c.GetPriorityQueues(),
	}
}
