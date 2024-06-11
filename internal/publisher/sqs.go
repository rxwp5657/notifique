package publisher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	c "github.com/notifique/controllers"
)

const (
	LOW_QUEUE_URL_ENV    string = "SQS_LOW"
	MEDIUM_QUEUE_URL_ENV string = "SQS_MEDIUM"
	HIGH_QUEUE_URL_ENV   string = "SQS_HIGH"
)

type SQSAPI interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type SQSEndpoints struct {
	Low    *string
	Medium *string
	High   *string
}

type SQSConfig struct {
	Client SQSAPI
	Urls   SQSEndpoints
}

type SQSPublisher struct {
	client    SQSAPI
	lowUrl    *string
	mediumUrl *string
	highUrl   *string
}

type SQSClientConfig struct {
	BaseEndpoint *string
	Region       *string
}

type Priority string

const (
	LOW    Priority = "LOW"
	MEDIUM Priority = "MEDIUM"
	HIGH   Priority = "HIGH"
)

func (p *SQSPublisher) Publish(ctx context.Context, notification c.Notification, storage c.NotificationStorage) error {

	var url *string = nil

	switch notification.Priority {
	case string(LOW):
		url = p.lowUrl
	case string(MEDIUM):
		url = p.mediumUrl
	case string(HIGH):
		url = p.highUrl
	default:
		return fmt.Errorf("invalid priority")
	}

	if url == nil {
		return fmt.Errorf("url for %s priority not found", notification.Priority)
	}

	messageBody, err := json.Marshal(notification)

	if err != nil {
		return fmt.Errorf("failed to marshall message body - %w", err)
	}

	_, err = p.client.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(string(messageBody)),
		QueueUrl:    url,
	})

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

func MakeSQSClient(clientCfg SQSClientConfig) (client *sqs.Client, err error) {

	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		return nil, fmt.Errorf("failed to load default config - %w", err)
	}

	client = sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		if clientCfg.BaseEndpoint != nil {
			o.BaseEndpoint = clientCfg.BaseEndpoint
		}

		if clientCfg.Region != nil {
			o.Region = *clientCfg.Region
		}
	})

	return
}

func MakeSQSPublisher(cfg SQSConfig) *SQSPublisher {
	return &SQSPublisher{
		client:    cfg.Client,
		lowUrl:    cfg.Urls.Medium,
		mediumUrl: cfg.Urls.Medium,
		highUrl:   cfg.Urls.High,
	}
}
