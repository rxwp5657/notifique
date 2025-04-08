package deployments

import (
	"path"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/notifique/service/internal/publish"
	"github.com/notifique/shared/clients"
	"github.com/notifique/shared/deploy"
)

type SQSDeployer interface {
	Deploy() (publish.PriorityQueues, error)
}

type SQSPriorityDeployer struct {
	Client *sqs.Client
	Queues publish.PriorityQueues
}

func (d *SQSPriorityDeployer) Deploy() (urls publish.PriorityQueues, err error) {

	availableQueues, err := deploy.GetQueues(d.Client)

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
			url, err := deploy.SQSQueue(d.Client, *name)
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

func NewSQSPriorityDeployer(c publish.SQSPriorityConfigurator) (*SQSPriorityDeployer, error) {
	client, err := clients.NewSQSClient(c)

	if err != nil {
		return nil, nil
	}

	deployer := SQSPriorityDeployer{
		Client: client,
		Queues: c.GetPriorityQueues(),
	}

	return &deployer, nil
}
