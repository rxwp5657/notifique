package clients

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// DynamoClientConfig holds the configuration settings for connecting to a DynamoDB instance.
// BaseEndpoint is an optional field that specifies the custom endpoint URL for the DynamoDB service.
// Region is an optional field that specifies the AWS region where the DynamoDB instance is hosted.
type DynamoClientConfig struct {
	BaseEndpoint *string
	Region       *string
}

// DynamoConfigurator is an interface that defines a method for retrieving
// the configuration for a DynamoDB client. Implementations of this interface
// should provide the necessary details to configure and connect to a DynamoDB
// instance.
type DynamoConfigurator interface {
	GetDynamoClientConfig() DynamoClientConfig
}

func NewDynamoDBClient(c DynamoConfigurator) (client *dynamodb.Client, err error) {

	clientCfg := c.GetDynamoClientConfig()

	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		return client, fmt.Errorf("failed to load default config - %w", err)
	}

	client = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if clientCfg.BaseEndpoint != nil {
			o.BaseEndpoint = clientCfg.BaseEndpoint
		}

		if clientCfg.Region != nil {
			o.Region = *clientCfg.Region
		}
	})

	return
}
