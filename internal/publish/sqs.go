package publish

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSAPI interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type SQSPublisher struct {
	client SQSAPI
}

type SQSClientConfig struct {
	BaseEndpoint *string
	Region       *string
}

type SQSConfigurator interface {
	GetSQSClientConfig() SQSClientConfig
}

type SQSPriorityConfigurator interface {
	SQSConfigurator
	PriorityQueueConfigurator
}

func (p *SQSPublisher) Publish(ctx context.Context, queueUrl string, message []byte) error {

	_, err := p.client.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(string(message)),
		QueueUrl:    &queueUrl,
	})

	return err
}

func NewSQSClient(c SQSConfigurator) (client *sqs.Client, err error) {

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

func NewSQSPublisher(a SQSAPI) *SQSPublisher {
	return &SQSPublisher{
		client: a,
	}
}
