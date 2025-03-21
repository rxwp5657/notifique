package publish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/notifique/internal/cache"
	c "github.com/notifique/internal/controllers"
	"github.com/notifique/internal/dto"
)

type Publisher interface {
	Publish(ctx context.Context, queueName, messageId string, message []byte) error
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
	cache     cache.Cache
	registry  c.NotificationRegistry
	queues    PriorityQueues
}

type PriorityPublisherCfg struct {
	Publisher         Publisher
	Cache             cache.Cache
	Registry          c.NotificationRegistry
	QueueConfigurator PriorityQueueConfigurator
}

func (p *PriorityPublisher) Publish(ctx context.Context, n c.NotificationMsg) error {

	var queueUri *string = nil

	switch n.Priority {
	case dto.Low:
		queueUri = p.queues.Low
	case dto.Medium:
		queueUri = p.queues.Medium
	case dto.High:
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

	errorsArr := []error{}

	publishErr := p.publisher.Publish(ctx, *queueUri, n.Hash, message)

	status := dto.Queued

	var errorMsg *string

	if publishErr != nil {
		errorsArr = append(errorsArr, publishErr)
		status = dto.Failed
		errStr := publishErr.Error()
		errorMsg = &errStr
	}

	statusLog := c.NotificationStatusLog{
		NotificationId: n.Id,
		Status:         status,
		ErrorMsg:       errorMsg,
	}

	if cacheErr := c.UpdateNotificationStatus(ctx, p.cache, statusLog); cacheErr != nil {
		errorsArr = append(errorsArr, fmt.Errorf("failed to update cache - %w", cacheErr))
	}

	if registryErr := p.registry.UpdateNotificationStatus(ctx, statusLog); registryErr != nil {
		errorsArr = append(errorsArr, fmt.Errorf("failed to update notification status - %w", registryErr))
	}

	return errors.Join(errorsArr...)
}

func NewPriorityPublisher(cfg PriorityPublisherCfg) *PriorityPublisher {

	return &PriorityPublisher{
		publisher: cfg.Publisher,
		registry:  cfg.Registry,
		queues:    cfg.QueueConfigurator.GetPriorityQueues(),
		cache:     cfg.Cache,
	}
}
