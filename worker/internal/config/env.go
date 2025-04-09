package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"github.com/notifique/shared/clients"
	wc "github.com/notifique/worker/internal/clients"
	"github.com/notifique/worker/internal/consumers"
	"github.com/notifique/worker/internal/providers"
	"github.com/notifique/worker/internal/sender"
)

const (
	rabbitMqUrl                           = "RABBITMQ_URL"
	sqsBaseEndpoint                       = "SQS_BASE_ENDPOINT"
	sqsRegion                             = "SQS_REGION"
	redisUrl                              = "REDIS_URL"
	consumerQueue                         = "CONSUMER_QUEUE"
	m2mTokenUrl                           = "M2M_TOKEN_URL"
	m2mClientId                           = "M2M_CLIENT_ID"
	m2mClientSecret                       = "M2M_CLIENT_SECRET"
	notificationServiceUrl                = "NOTIFICATION_SERVICE_URL"
	notificationServiceNumRetries         = "NOTIFICATION_SERVICE_NUM_RETRIES"
	notificationServiceBaseDelayInSeconds = "NOTIFICATION_SERVICE_BASE_DELAY_IN_SECONDS"
	notificationServiceMaxDelayInSeconds  = "NOTIFICATION_SERVICE_MAX_DELAY_IN_SECONDS"
	smtpHost                              = "SMTP_HOST"
	smtpPort                              = "SMTP_PORT"
	smtpUsername                          = "SMTP_USERNAME"
	smtpPassword                          = "SMTP_PASSWORD"
	smtpFrom                              = "SMTP_FROM"
	userPoolId                            = "USER_POOL_ID"
	cognitoBaseEndpoint                   = "COGNITO_BASE_ENDPOINT"
	cognitoRegion                         = "COGNITO_REGION"
	sqsMaxNumberOfMessages                = "MAX_NUMBER_OF_MESSAGES"
	sqsWaitTimeSeconds                    = "WAIT_TIME_SECONDS"
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

func (cfg EnvConfig) GetQueue() (consumers.RabbitMQQueue, error) {

	queue, ok := os.LookupEnv(consumerQueue)

	if !ok {
		return "", fmt.Errorf("queue %s not set", consumerQueue)
	}

	return consumers.RabbitMQQueue(queue), nil
}

func (cfg EnvConfig) GetCognitoAuthCfg() (wc.CognitoAuthProviderCfg, error) {

	config := wc.CognitoAuthProviderCfg{}

	if tokenUrl, ok := os.LookupEnv(m2mTokenUrl); ok {
		config.TokenUrl = tokenUrl
	} else {
		return config, fmt.Errorf("token url %s not set", m2mTokenUrl)
	}

	if clientId, ok := os.LookupEnv(m2mClientId); ok {
		config.ClientID = clientId
	} else {
		return config, fmt.Errorf("client id %s not set", m2mClientId)
	}

	if clientSecret, ok := os.LookupEnv(m2mClientSecret); ok {
		config.ClientSecret = clientSecret
	} else {
		return config, fmt.Errorf("client secret %s not set", m2mClientSecret)
	}

	return config, nil
}

func (cfg EnvConfig) GetNotificationServiceClientCfg() (wc.NotificationServiceClientCfg, error) {
	config := wc.NotificationServiceClientCfg{}

	if notificationServiceUrl, ok := os.LookupEnv(notificationServiceUrl); ok {
		config.NotificationServiceUrl = wc.NotificationServiceUrl(notificationServiceUrl)
	} else {
		return config, fmt.Errorf("notification service url %s not set", notificationServiceUrl)
	}

	if numRetries, ok := os.LookupEnv(notificationServiceNumRetries); ok {
		nr, err := strconv.Atoi(numRetries)
		if err != nil {
			return config, fmt.Errorf("error parsing number of retries %s - %w", numRetries, err)
		}
		if nr < 0 {
			return config, fmt.Errorf("number of retries %s is negative", numRetries)
		}
		config.NumRetries = nr
	} else {
		return config, fmt.Errorf("number of retries %s not set", notificationServiceNumRetries)
	}

	if baseDelay, ok := os.LookupEnv(notificationServiceBaseDelayInSeconds); ok {
		bd, err := strconv.Atoi(baseDelay)
		if err != nil {
			return config, fmt.Errorf("error parsing base delay %s - %w", baseDelay, err)
		}
		if bd < 0 {
			return config, fmt.Errorf("base delay %s is negative", baseDelay)
		}
		config.BaseDelay = wc.BaseDelay(time.Duration(bd) * time.Second)
	} else {
		return config, fmt.Errorf("base delay %s not set", notificationServiceBaseDelayInSeconds)
	}

	if maxDelay, ok := os.LookupEnv(notificationServiceMaxDelayInSeconds); ok {
		md, err := strconv.Atoi(maxDelay)
		if err != nil {
			return config, fmt.Errorf("error parsing max delay %s - %w", maxDelay, err)
		}
		if md < 0 {
			return config, fmt.Errorf("max delay %s is negative", maxDelay)
		}
		config.MaxDelay = wc.MaxDelay(time.Duration(md) * time.Second)
	} else {
		return config, fmt.Errorf("max delay %s not set", notificationServiceMaxDelayInSeconds)
	}

	return config, nil
}

func (cfg EnvConfig) GetSMTPConfig() (sender.SMTPConfig, error) {
	config := sender.SMTPConfig{}

	if host, ok := os.LookupEnv(smtpHost); ok {
		config.Host = host
	} else {
		return config, fmt.Errorf("smtp host %s not set", smtpHost)
	}

	if port, ok := os.LookupEnv(smtpPort); ok {
		p, err := strconv.Atoi(port)
		if err != nil {
			return config, fmt.Errorf("error parsing smtp port %s - %w", port, err)
		}
		config.Port = p
	} else {
		return config, fmt.Errorf("smtp port %s not set", smtpPort)
	}

	if username, ok := os.LookupEnv(smtpUsername); ok {
		config.Username = username
	} else {
		return config, fmt.Errorf("smtp username %s not set", smtpUsername)
	}

	if password, ok := os.LookupEnv(smtpPassword); ok {
		config.Password = password
	} else {
		return config, fmt.Errorf("smtp password %s not set", smtpPassword)
	}

	if from, ok := os.LookupEnv(smtpFrom); ok {
		config.From = from
	} else {
		return config, fmt.Errorf("smtp from %s not set", smtpFrom)
	}

	return config, nil
}

func (cfg EnvConfig) GetUserPoolId() (providers.UserPoolID, error) {
	userPoolId, ok := os.LookupEnv(userPoolId)

	if !ok {
		return "", fmt.Errorf("user pool id %s not set", userPoolId)
	}

	return providers.UserPoolID(userPoolId), nil
}

func (cfg EnvConfig) GetCognitoIdentityProviderConfig() providers.CognitoIdentityProviderConfig {
	config := providers.CognitoIdentityProviderConfig{}

	if baseEndpoint, ok := os.LookupEnv(cognitoBaseEndpoint); ok {
		config.BaseEndpoint = &baseEndpoint
	}

	if region, ok := os.LookupEnv(cognitoRegion); ok {
		config.Region = &region
	}

	return config
}

func (cfg EnvConfig) GetSQSQueueCfg() (consumers.SQSQueueCfg, error) {
	queueURL, ok := os.LookupEnv(consumerQueue)

	if !ok {
		return consumers.SQSQueueCfg{}, fmt.Errorf("sqs queue url %s not set", consumerQueue)
	}

	maxNumberOfMessages := int32(10)
	if max, ok := os.LookupEnv(sqsMaxNumberOfMessages); ok {
		m, err := strconv.Atoi(max)
		if err != nil {
			return consumers.SQSQueueCfg{}, fmt.Errorf("error parsing max number of messages %s - %w", max, err)
		}
		maxNumberOfMessages = min(int32(m), maxNumberOfMessages)
	}

	waitTimeSeconds := int32(20)
	if wait, ok := os.LookupEnv(sqsWaitTimeSeconds); ok {
		w, err := strconv.Atoi(wait)
		if err != nil {
			return consumers.SQSQueueCfg{}, fmt.Errorf("error parsing wait time seconds %s - %w", wait, err)
		}
		waitTimeSeconds = min(int32(w), waitTimeSeconds)
	}

	return consumers.SQSQueueCfg{
		QueueURL:            queueURL,
		MaxNumberOfMessages: maxNumberOfMessages,
		WaitTimeSeconds:     waitTimeSeconds,
	}, nil
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
