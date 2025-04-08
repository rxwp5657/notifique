package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/notifique/shared/clients"
)

const (
	rabbitMqUrl     = "RABBITMQ_URL"
	sqsBaseEndpoint = "SQS_BASE_ENDPOINT"
	sqsRegion       = "SQS_REGION"
	redisUrl        = "REDIS_URL"
	consumerQueue   = "CONSUMER_QUEUE"
)

type EnvConfig struct{}

func (cfg EnvConfig) GetRabbitMQUrl() (string, error) {
	url, ok := os.LookupEnv(rabbitMqUrl)

	if !ok {
		return "", fmt.Errorf("rabbitmq url env variable %s not found", rabbitMqUrl)
	}

	return url, nil
}

func (cfg EnvConfig) GetSQSClientConfig() (sqsCfg clients.SQSClientConfig) {

	if be, ok := os.LookupEnv(sqsBaseEndpoint); ok {
		sqsCfg.BaseEndpoint = &be
	}

	if region, ok := os.LookupEnv(sqsRegion); ok {
		sqsCfg.Region = &region
	}

	return
}

func (cfg EnvConfig) GetRedisUrl() (string, error) {

	url, ok := os.LookupEnv(redisUrl)

	if !ok {
		return "", fmt.Errorf("redis url %s not set", redisUrl)
	}

	return url, nil
}

func (cfg EnvConfig) GetQueue() (string, error) {

	queue, ok := os.LookupEnv(consumerQueue)

	if !ok {
		return "", fmt.Errorf("queue %s not set", consumerQueue)
	}

	return queue, nil
}

func NewEnvConfig(envFile *string) (*EnvConfig, error) {

	if envFile == nil {
		cfg := EnvConfig{}
		return &cfg, nil
	}

	err := godotenv.Load(*envFile)

	if err != nil {
		return nil, fmt.Errorf("failed to load env file %s - %w", *envFile, err)
	}

	cfg := EnvConfig{}
	return &cfg, nil
}
