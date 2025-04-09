package publish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	c "github.com/notifique/service/internal/controllers"
	"github.com/notifique/shared/cache"
	"github.com/notifique/shared/dto"
)

type Message struct {
	Id       string
	Payload  []byte
	Priority dto.NotificationPriority
}

type Publisher interface {
	Publish(ctx context.Context, queueName string, message Message) error
}

type PriorityQueues struct {
	Low    *string
	Medium *string
	High   *string
}

type PriorityQueueConfigurator interface {
	GetPriorityQueues() PriorityQueues
}

type Priority struct {
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

func (p *Priority) Publish(ctx context.Context, n dto.NotificationMsgPayload) error {

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

	notificationMessage, err := json.Marshal(n)

	if err != nil {
		return fmt.Errorf("failed to marshall message body - %w", err)
	}

	errorsArr := []error{}

	message := Message{
		Id:       n.Hash,
		Payload:  notificationMessage,
		Priority: n.Priority,
	}

	publishErr := p.publisher.Publish(ctx, *queueUri, message)

	status := dto.Queued

	var errorMsg *string

	if publishErr != nil {
		errorsArr = append(errorsArr, publishErr)
		status = dto.Failed
		errStr := publishErr.Error()
		errorMsg = &errStr
	}

	statusLog := dto.NotificationStatusLog{
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

func NewPriorityPublisher(cfg PriorityPublisherCfg) *Priority {

	return &Priority{
		publisher: cfg.Publisher,
		registry:  cfg.Registry,
		queues:    cfg.QueueConfigurator.GetPriorityQueues(),
		cache:     cfg.Cache,
	}
}
