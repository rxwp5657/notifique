package deploy

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func GetQueues(c *sqs.Client) (queueUrls []string, err error) {

	paginator := sqs.NewListQueuesPaginator(c, &sqs.ListQueuesInput{})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())

		if err != nil {
			return nil, fmt.Errorf("failed to get queue urls - %w", err)
		}

		queueUrls = append(queueUrls, output.QueueUrls...)
	}

	return
}

func SQSQueue(c *sqs.Client, queueName string) (queueUrl string, err error) {

	queueName = fmt.Sprintf("%s.fifo", queueName)

	queue, err := c.CreateQueue(context.TODO(), &sqs.CreateQueueInput{
		QueueName: &queueName,
		Attributes: map[string]string{
			"FifoQueue":                 "true",
			"ContentBasedDeduplication": "true",
			"VisibilityTimeout":         "300",
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to create queue - %w", err)
	}

	queueUrl = *queue.QueueUrl

	return
}
