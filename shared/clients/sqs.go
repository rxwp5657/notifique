package clients

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSClientConfig struct {
	BaseEndpoint *string
	Region       *string
}

type SQSConfigurator interface {
	GetSQSClientConfig() SQSClientConfig
}

func NewSQSClient(c SQSConfigurator) (*sqs.Client, error) {

	clientCfg := c.GetSQSClientConfig()

	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		return nil, fmt.Errorf("failed to load default config - %w", err)
	}

	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		if clientCfg.BaseEndpoint != nil {
			o.BaseEndpoint = clientCfg.BaseEndpoint
		}

		if clientCfg.Region != nil {
			o.Region = *clientCfg.Region
		}
	})

	return client, nil
}
