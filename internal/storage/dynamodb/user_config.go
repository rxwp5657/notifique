package storage

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type channelConfig struct {
	OptIn       bool    `dynamodbav:"optIn"`
	SnoozeUntil *string `dynamodbav:"snoozeUntil"`
}

type userConfig struct {
	UserId      string        `dynamodbav:"userId"`
	EmailConfig channelConfig `dynamodbav:"emailConfig"`
	SMSConfig   channelConfig `dynamodbav:"smsConfig"`
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
