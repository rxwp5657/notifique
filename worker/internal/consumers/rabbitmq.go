package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/google/uuid"
	"github.com/notifique/shared/dto"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQAPI interface {
	Ack(tag uint64, multiple bool) error
	Cancel(consumer string, noWait bool) error
	Consume(queue string, consumer string, autoAck bool, exclusive bool, noLocal bool, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
}

type RabbitMQ struct {
	id               string
	ch               RabbitMQAPI
	messageChan      <-chan amqp.Delivery
	notificationChan chan<- dto.NotificationMsg
}

type RabbitMQCfg struct {
	Client           RabbitMQAPI
	Queue            string
	NotificationChan chan<- dto.NotificationMsg
}

func (r *RabbitMQ) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			r.ch.Cancel(r.id, false)
			return
		case delivery := <-r.messageChan:
			payload := dto.NotificationMsgPayload{}
			err := json.Unmarshal(delivery.Body, &payload)

			if err != nil {
				slog.Error("failed to unmarshal dto", "reason", err)
				continue
			}

			msg := dto.NotificationMsg{
				MessageId: delivery.MessageId,
				DeleteTag: strconv.FormatUint(delivery.DeliveryTag, 10),
				Payload:   payload,
			}

			r.notificationChan <- msg
		}
	}
}

func (r *RabbitMQ) Ack(ctx context.Context, deleteTag string) error {

	tag, err := strconv.ParseUint(deleteTag, 10, 64)

	if err != nil {
		return fmt.Errorf("failed to parse message id - %w", err)
	}

	err = r.ch.Ack(tag, false)

	if err != nil {
		return fmt.Errorf("failed to ack message - %w", err)
	}

	return nil
}

func NewRabbitMQConsumer(cfg RabbitMQCfg) (*RabbitMQ, error) {

	consumerId := uuid.NewString()

	messageChan, err := cfg.Client.Consume(
		cfg.Queue,
		consumerId,
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to make message consumer channel - %w", err)
	}

	consumer := &RabbitMQ{
		id:               consumerId,
		ch:               cfg.Client,
		messageChan:      messageChan,
		notificationChan: cfg.NotificationChan,
	}

	return consumer, nil
}
