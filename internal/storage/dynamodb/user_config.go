package storage

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	USER_CONFIG_TABLE        = "UserConfig"
	USER_CONFIG_HASH_KEY     = "userId"
	USER_CONFIG_EMAIL_KEY    = "emailConfig"
	USER_CONFIG_SMS_KEY      = "smsConfig"
	USER_CONFIG_INAPP_KEY    = "inAppConfig"
	USER_CONFIG_SNOOZE_UNTIL = "snoozeUntil"
	USER_CONFIG_OPT_IN       = "optIn"
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

func (cfg *userConfig) GetKey() (DynamoDBKey, error) {
	key := make(map[string]types.AttributeValue)

	userId, err := attributevalue.Marshal(cfg.UserId)

	if err != nil {
		return key, fmt.Errorf("failed to make user config key - %w", err)
	}

	key["userId"] = userId

	return key, nil
}
