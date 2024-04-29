package storage

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type userNotification struct {
	Id        string  `dynamodbav:"id"`
	UserId    string  `dynamodbav:"userId"`
	Title     string  `dynamodbav:"title"`
	Contents  string  `dynamodbav:"contents"`
	CreatedAt string  `dynamodbav:"createdAt"`
	Image     *string `dynamodbav:"image"`
	ReadAt    *string `dynamodbav:"readAt"`
	Topic     string  `dynamodbav:"topic"`
}

func (n *userNotification) GetKey() (DynamoDBKey, error) {
	key := make(map[string]types.AttributeValue)

	id, err := attributevalue.Marshal(n.Id)

	if err != nil {
		return key, fmt.Errorf("failed to make user config key - %w", err)
	}

	key["id"] = id

	return key, nil
}
