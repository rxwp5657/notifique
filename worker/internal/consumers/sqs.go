package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/notifique/shared/dto"
)

type SQSAPI interface {
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

type SQSQueueCfg struct {
	QueueURL            string
	MaxNumberOfMessages int32
	WaitTimeSeconds     int32
}

type SQSQueueConfigurator interface {
	GetSQSQueueCfg() (SQSQueueCfg, error)
}

type SQSCfg struct {
	QueueCfg    SQSQueueCfg
	Client      SQSAPI
	MessageChan chan<- dto.NotificationMsg
}

type SQS struct {
	client      SQSAPI
	queueCfg    SQSQueueCfg
	messageChan chan<- dto.NotificationMsg
}

func (c *SQS) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		messages, err := c.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &c.queueCfg.QueueURL,
			MaxNumberOfMessages: c.queueCfg.MaxNumberOfMessages,
			WaitTimeSeconds:     c.queueCfg.WaitTimeSeconds,
		})

		if err != nil {
			err = fmt.Errorf("failed to receive messages - %w", err)
			slog.Error(err.Error())
			continue
		}

		for _, m := range messages.Messages {
			payload := dto.NotificationMsgPayload{}
			err := json.Unmarshal([]byte(*m.Body), &payload)

			if err != nil {
				err = fmt.Errorf("failed to unmarshal message - %w", err)
				slog.Error(err.Error())
				continue
			}

			msg := dto.NotificationMsg{
				MessageId: *m.MessageId,
				Payload:   payload,
				DeleteTag: *m.ReceiptHandle,
			}

			c.messageChan <- msg
		}
	}
}

func (c *SQS) Ack(ctx context.Context, deleteTag string) error {

	_, err := c.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &c.queueCfg.QueueURL,
		ReceiptHandle: &deleteTag,
	})

	if err != nil {
		return fmt.Errorf("failed to ack message - %w", err)
	}

	return nil
}

func NewSQSConsumer(cfg SQSCfg) (*SQS, error) {

	consumer := SQS{
		messageChan: cfg.MessageChan,
		queueCfg:    cfg.QueueCfg,
		client:      cfg.Client,
	}

	return &consumer, nil
}
