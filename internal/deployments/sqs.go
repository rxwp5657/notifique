package deployments

import (
	"context"
	"fmt"
	"path"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/notifique/internal/publisher"
)

type SQSDeployer interface {
	Deploy() (publisher.PriorityQueues, error)
}

type SQSPriorityDeployer struct {
	Client *sqs.Client
	Queues publisher.PriorityQueues
}

func (d *SQSPriorityDeployer) Deploy() (urls publisher.PriorityQueues, err error) {

	availableQueues, err := getQueues(d.Client)

	if err != nil {
		return urls, err
	}

	availableQueuesMap := make(map[string]string)

	for _, queueUrl := range availableQueues {
		queueName := path.Base(queueUrl)
		availableQueuesMap[queueName] = queueUrl
	}

	getQueueUrl := func(name *string) (*string, error) {

		if name == nil {
			return nil, nil
		}

		url, ok := availableQueuesMap[*name]

		if !ok {
			url, err := createSQSQueue(d.Client, *name)
			return &url, err
		}

		return &url, nil
	}

	lowUrl, err := getQueueUrl(d.Queues.Low)

	if err != nil {
		return urls, err
	}

	midUrl, err := getQueueUrl(d.Queues.Medium)

	if err != nil {
		return urls, err
	}

	highUrl, err := getQueueUrl(d.Queues.High)

	if err != nil {
		return urls, err
	}

	urls.Low = lowUrl
	urls.Medium = midUrl
	urls.High = highUrl

	return
}

func getQueues(c *sqs.Client) (queueUrls []string, err error) {

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

func createSQSQueue(c *sqs.Client, queueName string) (queueUrl string, err error) {

	queue, err := c.CreateQueue(context.TODO(), &sqs.CreateQueueInput{
		QueueName: &queueName,
	})

	if err != nil {
		return "", fmt.Errorf("failed to create queue - %w", err)
	}

	queueUrl = *queue.QueueUrl

	return
}

func NewSQSPriorityDeployer(c publisher.SQSPriorityConfigurator) (*SQSPriorityDeployer, func(), error) {
	client, err := publisher.NewSQSClient(c)

	if err != nil {
		return nil, nil, nil
	}

	cleanup := func() {}

	deployer := SQSPriorityDeployer{
		Client: client,
		Queues: c.GetPriorityQueues(),
	}

	return &deployer, cleanup, nil
}
