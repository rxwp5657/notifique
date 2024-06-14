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

type SQSPublisher struct {
	client SQSAPI
	queues PriorityQueues
}

type SQSClientConfig struct {
	BaseEndpoint *string
	Region       *string
}

type SQSConfigurator interface {
	GetSQSClientConfig() SQSClientConfig
}

func (p *SQSPublisher) Publish(ctx context.Context, n c.Notification, s c.NotificationStorage) error {
	return publishByPriority(ctx, n, s, p, p.queues)
}

func (p *SQSPublisher) PublishMsg(ctx context.Context, q string, m []byte) error {

	_, err := p.client.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(string(m)),
		QueueUrl:    &q,
	})

	return err
}

func MakeSQSClient(c SQSConfigurator) (client *sqs.Client, err error) {

	clientCfg := c.GetSQSClientConfig()

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

func MakeSQSPublisher(a SQSAPI, c PriorityQueueConfigurator) *SQSPublisher {
	return &SQSPublisher{
		client: a,
		queues: c.GetPriorityQueues(),
	}
}
