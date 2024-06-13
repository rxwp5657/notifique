package publisher

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	c "github.com/notifique/controllers"
)

type SQSAPI interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type SQSConfig struct {
	Client SQSAPI
	Queues PriorityQueues
}

type SQSPublisher struct {
	client SQSAPI
	queues PriorityQueues
}

type SQSClientConfig struct {
	BaseEndpoint *string
	Region       *string
}

func (p *SQSPublisher) Publish(ctx context.Context, notification c.Notification, storage c.NotificationStorage) error {
	return publishByPriority(ctx, notification, storage, p, p.queues)
}

func (p *SQSPublisher) PublishMsg(ctx context.Context, queue string, message []byte) error {

	_, err := p.client.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(string(message)),
		QueueUrl:    &queue,
	})

	return err
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
		client: cfg.Client,
		queues: cfg.Queues,
	}
}
