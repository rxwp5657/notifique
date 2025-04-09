package consumers_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/notifique/shared/clients"
	"github.com/notifique/shared/dto"
	"github.com/notifique/worker/internal/consumers"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQAPI interface {
	consumers.RabbitMQAPI
	PublishWithContext(_ context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

type RabbitMQCfg struct {
	consumers.RabbitMQCfg
	Client *clients.RabbitMQ
}

type RabbitMQ struct {
	*consumers.RabbitMQ
	ch    RabbitMQAPI
	queue string
}

type SQSAPI interface {
	consumers.SQSAPI
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type SQSCfg struct {
	consumers.SQSCfg
	Client *sqs.Client
}

type SQS struct {
	*consumers.SQS
	client   SQSAPI
	queueCfg consumers.SQSQueueCfg
}

func (r *RabbitMQ) Publish(ctx context.Context, n dto.NotificationMsgPayload) error {

	message, err := json.Marshal(n)

	if err != nil {
		return fmt.Errorf("failed to marshal message - %w", err)
	}

	return r.ch.PublishWithContext(
		ctx,
		"",
		r.queue,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         message,
			MessageId:    n.Hash,
		},
	)
}

func (s *SQS) Publish(ctx context.Context, n dto.NotificationMsgPayload) error {

	message, err := json.Marshal(n)

	if err != nil {
		return fmt.Errorf("failed to marshal message - %w", err)
	}

	_, err = s.client.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody:            aws.String(string(message)),
		QueueUrl:               &s.queueCfg.QueueURL,
		MessageDeduplicationId: &n.Hash,
		MessageGroupId:         aws.String("default"),
	})

	return err
}

func NewRabbitMQConsumerTest(ctx context.Context, cfg RabbitMQCfg) (*RabbitMQ, error) {

	c, err := consumers.NewRabbitMQConsumer(cfg.RabbitMQCfg)

	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ consumer - %w", err)
	}

	ct := &RabbitMQ{
		RabbitMQ: c,
		ch:       cfg.Client,
		queue:    string(cfg.Queue),
	}

	return ct, nil
}

func NewSQSConsumerTest(ctx context.Context, cfg SQSCfg) (*SQS, error) {

	c, err := consumers.NewSQSConsumer(cfg.SQSCfg)

	if err != nil {
		return nil, fmt.Errorf("failed to create SQS consumer - %w", err)
	}

	ct := &SQS{
		SQS:      c,
		client:   cfg.Client,
		queueCfg: cfg.QueueCfg,
	}

	return ct, nil
}
