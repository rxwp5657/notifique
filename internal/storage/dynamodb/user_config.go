package storage

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	UserConfigTable       = "UserConfig"
	UserConfigHashKey     = "userId"
	UserConfigEmailKey    = "emailConfig"
	UserConfigSmsKey      = "smsConfig"
	UserConfigInAppKey    = "inAppConfig"
	UserConfigSnoozeUntil = "snoozeUntil"
	UserConfigOptIn       = "optIn"
)

type channelConfig struct {
	OptIn       bool    `dynamodbav:"optIn"`
	SnoozeUntil *string `dynamodbav:"snoozeUntil"`
}

type userConfig struct {
	UserId      string        `dynamodbav:"userId"`
	EmailConfig channelConfig `dynamodbav:"emailConfig"`
	SMSConfig   channelConfig `dynamodbav:"smsConfig"`
	InAppConfig channelConfig `dynamodbav:"inAppConfig"`
}

func (cfg *userConfig) GetKey() (DynamoKey, error) {
	key := make(map[string]types.AttributeValue)

	userId, err := attributevalue.Marshal(cfg.UserId)

	if err != nil {
		return key, fmt.Errorf("failed to make user config key - %w", err)
	}

	key["userId"] = userId

	return key, nil
}
