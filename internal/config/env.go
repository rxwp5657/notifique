package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/notifique/internal/publisher"
	storage "github.com/notifique/internal/storage/dynamodb"
)

const (
	POSTGRES_URL          = "POSTGRES_URL"
	DYNAMO_BASE_ENDPOINT  = "DYNAMO_BASE_ENDPOINT"
	DYNAMO_REGION         = "DYNAMO_REGION"
	LOW_PRIORITY_QUEUE    = "LOW_PRIORITY_QUEUE"
	MEDIUM_PRIORITY_QUEUE = "MEDIUM_PRIORITY_QUEUE"
	HIGH_PRIORITY_QUEUE   = "HIGH_PRIORITY_QUEUE"
	RABBITMQ_URL          = "RABBITMQ_URL"
	SQS_BASE_ENDPOINT     = "SQS_BASE_ENDPOINT"
	SQS_REGION            = "SQS_REGION"
)

type EnvConfig struct{}

func (cfg EnvConfig) GetPostgresUrl() (string, error) {

	url, ok := os.LookupEnv(POSTGRES_URL)

	if !ok {
		return "", fmt.Errorf("postgres env variable %s not set", POSTGRES_URL)
	}

	return url, nil
}

func (cfg EnvConfig) GetDynamoClientConfig() (dcfg storage.DynamoClientConfig) {

	if be, ok := os.LookupEnv(DYNAMO_BASE_ENDPOINT); ok {
		dcfg.BaseEndpoint = &be
	}

	if region, ok := os.LookupEnv(DYNAMO_REGION); ok {
		dcfg.Region = &region
	}

	return
}

func (cfg EnvConfig) GetPriorityQueues() (queues publisher.PriorityQueues) {

	if low, ok := os.LookupEnv(LOW_PRIORITY_QUEUE); ok {
		queues.Low = &low
	}

	if medium, ok := os.LookupEnv(MEDIUM_PRIORITY_QUEUE); ok {
		queues.Medium = &medium
	}

	if high, ok := os.LookupEnv(HIGH_PRIORITY_QUEUE); ok {
		queues.High = &high
	}

	return
}

func (cfg EnvConfig) GetRabbitMQUrl() (string, error) {
	url, ok := os.LookupEnv(RABBITMQ_URL)

	if !ok {
		return "", fmt.Errorf("rabbitmq url env variable %s not found", RABBITMQ_URL)
	}

	return url, nil
}

func (cfg EnvConfig) GetSQSClientConfig() (sqsCfg publisher.SQSClientConfig) {

	if be, ok := os.LookupEnv(SQS_BASE_ENDPOINT); ok {
		sqsCfg.BaseEndpoint = &be
	}

	if region, ok := os.LookupEnv(SQS_REGION); ok {
		sqsCfg.Region = &region
	}

	return
}

func MakeEnvConfig(envFile string) (cfg *EnvConfig, err error) {
	err = godotenv.Load(envFile)

	if err != nil {
		return cfg, fmt.Errorf("failed to load env file %s - %w", envFile, err)
	}

	return
}
