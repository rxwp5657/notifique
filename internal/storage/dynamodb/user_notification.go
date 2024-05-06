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

type userNotificationKey struct {
	UserId    string `dynamodbav:"userId"`
	CreatedAt string `dynamodbav:"createdAt"`
}

func (n *userNotification) GetKey() (DynamoDBKey, error) {
	key := make(map[string]types.AttributeValue)

	id, err := attributevalue.Marshal(n.UserId)

	if err != nil {
		return key, fmt.Errorf("failed to marshall userId - %w", err)
	}

	createdAt, err := attributevalue.Marshal(n.CreatedAt)

	if err != nil {
		return key, fmt.Errorf("failed to marshall createdAt - %w", err)
	}

	key["userId"] = id
	key["createdAt"] = createdAt

	return key, nil
}

func (n *userNotificationKey) GetKey() (DynamoDBKey, error) {
	un := userNotification{UserId: n.UserId, CreatedAt: n.CreatedAt}

	return un.GetKey()
}
