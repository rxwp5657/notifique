package deployments

import (
	"context"
	"fmt"
	"path"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/notifique/internal/publisher"
)

func getQueues(c *sqs.Client) (queueUrls []string, err error) {

	paginator := sqs.NewListQueuesPaginator(c, &sqs.ListQueuesInput{})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())

		if err != nil {
			return nil, fmt.Errorf("failed to get queue urls")
		}

		queueUrls = append(queueUrls, output.QueueUrls...)
	}

	return
}

func createQueue(c *sqs.Client, queueName string) (queueUrl string, err error) {

	queue, err := c.CreateQueue(context.TODO(), &sqs.CreateQueueInput{
		QueueName: &queueName,
	})

	if err != nil {
		return "", fmt.Errorf("failed to create queue - %w", err)
	}

	queueUrl = *queue.QueueUrl

	return
}

func MakePriorityQueues(c *sqs.Client, queueNames publisher.PriorityQueues) (urls publisher.PriorityQueues, err error) {

	queueUrls, err := getQueues(c)

	if err != nil {
		return urls, err
	}

	existingQueues := make(map[string]string)

	for _, queueUrl := range queueUrls {
		queueName := path.Base(queueUrl)
		existingQueues[queueName] = queueUrl
	}

	createQueueIfNotExists := func(name *string) (*string, error) {

		if name == nil {
			return nil, nil
		}

		url, ok := existingQueues[*name]

		if !ok {
			url, err := createQueue(c, *name)
			return &url, err
		}

		return &url, nil
	}

	low, err := createQueueIfNotExists(queueNames.Low)

	if err != nil {
		return urls, fmt.Errorf("failed to create low priority queue - %w", err)
	}

	medium, err := createQueueIfNotExists(queueNames.Medium)

	if err != nil {
		return urls, fmt.Errorf("failed to create medium priority queue - %w", err)
	}

	high, err := createQueueIfNotExists(queueNames.High)

	if err != nil {
		return urls, fmt.Errorf("failed to create high priority queue - %w", err)
	}

	urls.Low = low
	urls.Medium = medium
	urls.High = high

	return
}
