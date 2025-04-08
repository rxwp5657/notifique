package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"github.com/notifique/service/internal/publish"
	"github.com/notifique/shared/clients"
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
	apiVersion          = "API_VERSION"
	expectedHost        = "EXPECTED_HOST"
	requestsPerSecond   = "REQUESTS_PER_SECOND"
	cacheTTLInSeconds   = "CACHE_TTL_IN_SECONDS"
	workerQueue         = "WORKER_QUEUE"
	jwksUrl             = "JWKS_URL"
)

type EnvConfig struct{}

func (cfg EnvConfig) GetPostgresUrl() (string, error) {

	url, ok := os.LookupEnv(postgresUrl)

	if !ok {
		return "", fmt.Errorf("postgres env variable %s not set", postgresUrl)
	}

	return url, nil
}

func (cfg EnvConfig) GetDynamoClientConfig() (dcfg clients.DynamoClientConfig) {

	if be, ok := os.LookupEnv(dynamoBaseEndpoint); ok {
		dcfg.BaseEndpoint = &be
	}

	if region, ok := os.LookupEnv(dynamoRegion); ok {
		dcfg.Region = &region
	}

	return
}

func (cfg EnvConfig) GetPriorityQueues() (queues publish.PriorityQueues) {

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

func (cfg EnvConfig) GetSQSClientConfig() (sqsCfg clients.SQSClientConfig) {

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

func (cfg EnvConfig) GetVersion() (string, error) {
	version, ok := os.LookupEnv(apiVersion)

	if !ok {
		return "", fmt.Errorf("api version env variable %s not found", apiVersion)
	}

	return version, nil
}

func (cfg EnvConfig) GetExpectedHost() *string {

	host, ok := os.LookupEnv(expectedHost)

	if !ok {
		return nil
	}

	return &host
}

func (cfg EnvConfig) GetRequestsPerSecond() (int, error) {

	rps, ok := os.LookupEnv(requestsPerSecond)

	if !ok {
		return 0, fmt.Errorf("requests per second env variable %s not found", requestsPerSecond)
	}

	rpsInt, err := strconv.Atoi(rps)

	if err != nil {
		return 0, fmt.Errorf("failed to parse requests per second to int - %w", err)
	}

	return rpsInt, nil
}

func (cfg EnvConfig) GetTTL() (time.Duration, error) {

	ttl, ok := os.LookupEnv(cacheTTLInSeconds)

	if !ok {
		return 0, fmt.Errorf("cache ttl env variable %s not found", cacheTTLInSeconds)
	}

	ttlInt, err := strconv.Atoi(ttl)

	if err != nil {
		return 0, fmt.Errorf("failed to parse cache ttl to int - %w", err)
	}

	return time.Duration(ttlInt) * time.Second, nil
}

func (cfg EnvConfig) GetJWKSURL() (string, error) {

	jwks, ok := os.LookupEnv(jwksUrl)

	if !ok {
		return "", fmt.Errorf("jwks url env variable %s not found", jwksUrl)
	}

	return jwks, nil
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
