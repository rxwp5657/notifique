package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/notifique/internal/publisher"
	storage "github.com/notifique/internal/storage/dynamodb"
)

const (
	postgresUrl         = "POSTGRES_URL"
	dynamoBaseEndpoint  = "DYNAMO_BASE_ENDPOINT"
	dynamoRegion        = "DYNAMO_REGION"
	lowPriorityQueue    = "LOW_PRIORITY_QUEUE"
	mediumPriorityQueue = "MEDIUM_PRIORITY_QUEUE"
	highPriorityQueue   = "HIGH_PRIORITY_QUEUE"
	rabbitMqUrl         = "RABBIT_MQ_URL"
	sqsBaseEndpoint     = "SQS_BASE_ENDPOINT"
	sqsRegion           = "SQS_REGION"
)

type EnvConfig struct{}

func (cfg EnvConfig) GetPostgresUrl() (string, error) {

	url, ok := os.LookupEnv(postgresUrl)

	if !ok {
		return "", fmt.Errorf("postgres env variable %s not set", postgresUrl)
	}

	return url, nil
}

func (cfg EnvConfig) GetDynamoClientConfig() (dcfg storage.DynamoClientConfig) {

	if be, ok := os.LookupEnv(dynamoBaseEndpoint); ok {
		dcfg.BaseEndpoint = &be
	}

	if region, ok := os.LookupEnv(dynamoRegion); ok {
		dcfg.Region = &region
	}

	return
}

func (cfg EnvConfig) GetPriorityQueues() (queues publisher.PriorityQueues) {

	if low, ok := os.LookupEnv(lowPriorityQueue); ok {
		queues.Low = &low
	}

	if medium, ok := os.LookupEnv(mediumPriorityQueue); ok {
		queues.Medium = &medium
	}

	if high, ok := os.LookupEnv(highPriorityQueue); ok {
		queues.High = &high
	}

	return
}

func (cfg EnvConfig) GetRabbitMQUrl() (string, error) {
	url, ok := os.LookupEnv(rabbitMqUrl)

	if !ok {
		return "", fmt.Errorf("rabbitmq url env variable %s not found", rabbitMqUrl)
	}

	return url, nil
}

func (cfg EnvConfig) GetSQSClientConfig() (sqsCfg publisher.SQSClientConfig) {

	if be, ok := os.LookupEnv(sqsBaseEndpoint); ok {
		sqsCfg.BaseEndpoint = &be
	}

	if region, ok := os.LookupEnv(sqsRegion); ok {
		sqsCfg.Region = &region
	}

	return
}

func MakeEnvConfig(envFile string) (*EnvConfig, error) {
	err := godotenv.Load(envFile)

	if err != nil {
		return nil, fmt.Errorf("failed to load env file %s - %w", envFile, err)
	}

	cfg := EnvConfig{}
	return &cfg, nil
}
