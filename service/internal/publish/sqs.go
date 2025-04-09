package publish

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/notifique/shared/clients"
)

type SQSAPI interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type SQS struct {
	client SQSAPI
}

type SQSPriorityConfigurator interface {
	clients.SQSConfigurator
	PriorityQueueConfigurator
}

func (p *SQS) Publish(ctx context.Context, queueUrl string, message Message) error {

	_, err := p.client.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody:            aws.String(string(message.Payload)),
		QueueUrl:               &queueUrl,
		MessageDeduplicationId: &message.Id,
		MessageGroupId:         aws.String(string(message.Priority)),
	})

	return err
}

func NewSQSPublisher(a SQSAPI) *SQS {
	return &SQS{
		client: a,
	}
}
