package config

import (
	"fmt"
	"os"
	"strconv"

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
	rabbitMqUrl         = "RABBITMQ_URL"
	sqsBaseEndpoint     = "SQS_BASE_ENDPOINT"
	sqsRegion           = "SQS_REGION"
	brokerCapacity      = "BROKER_CAPACITY"
	redisUrl            = "REDIS_URL"
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

func (cfg EnvConfig) GetBrokerChannelSize() (int, error) {

	chSizeStr, ok := os.LookupEnv(brokerCapacity)

	if !ok {
		return 0, fmt.Errorf("broker capacity %s not set", brokerCapacity)
	}

	chSize, err := strconv.Atoi(chSizeStr)

	if err != nil {
		return 0, fmt.Errorf("failed to parse capacity to int - %w", err)
	}

	return chSize, nil
}

func (cfg EnvConfig) GetRedisUrl() (string, error) {

	url, ok := os.LookupEnv(redisUrl)

	if !ok {
		return "", fmt.Errorf("redis url %s not set", redisUrl)
	}

	return url, nil
}

func NewEnvConfig(envFile string) (*EnvConfig, error) {
	err := godotenv.Load(envFile)

	if err != nil {
		return nil, fmt.Errorf("failed to load env file %s - %w", envFile, err)
	}

	cfg := EnvConfig{}
	return &cfg, nil
}
