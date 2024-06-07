package deployments

import (
	"context"
	"fmt"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/notifique/internal/publisher"
)

const (
	LOW    = "notifique-low"
	MEDIUM = "notifique-medium"
	HIGH   = "notifique-high"
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

func MakeQueues(c *sqs.Client) (publisher.SQSEndpoints, error) {

	queueUrls, err := getQueues(c)
	urls := publisher.SQSEndpoints{}

	if err != nil {
		return urls, err
	}

	existingQueues := make(map[string]string)

	for _, queueUrl := range queueUrls {
		queueName := path.Base(queueUrl)
		existingQueues[queueName] = queueUrl
	}

	queuesToCreate := []string{LOW, MEDIUM, HIGH}

	for _, queue := range queuesToCreate {

		if _, ok := existingQueues[queue]; !ok {
			url, err := createQueue(c, queue)

			if err != nil {
				return urls, fmt.Errorf("failed to create queue %s - %w", queue, err)
			}

			existingQueues[queue] = url
		}
	}

	urls.Low = aws.String(existingQueues[LOW])
	urls.Medium = aws.String(existingQueues[MEDIUM])
	urls.High = aws.String(existingQueues[HIGH])

	return urls, nil
}
