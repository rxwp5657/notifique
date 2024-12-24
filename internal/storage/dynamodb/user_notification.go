package dynamostorage

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	UserNotificationsTable               = "UserNotifications"
	UserNotificationsCreatedAtIdx        = "createdAtIdx"
	UserNotificactionsHashKey            = "userId"
	UserNotificationsSortKey             = "id"
	UserNotificationsCreatedAtIdxSortKey = "createdAt"
)

type UserNotification struct {
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
	UserId    string `dynamodbav:"userId" json:"userId"`
	CreatedAt string `dynamodbav:"createdAt" json:"createdAt"`
}

func (n *UserNotification) GetKey() (DynamoKey, error) {
	key := make(map[string]types.AttributeValue)

	userId, err := attributevalue.Marshal(n.UserId)

	if err != nil {
		return key, fmt.Errorf("failed to marshall userId - %w", err)
	}

	id, err := attributevalue.Marshal(n.Id)

	if err != nil {
		return key, fmt.Errorf("failed to marshall createdAt - %w", err)
	}

	key["userId"] = userId
	key["id"] = id

	return key, nil
}

func (n *UserNotification) GetSecondaryIdxKey() (DynamoKey, error) {
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

func (n *userNotificationKey) GetKey() (DynamoKey, error) {
	un := UserNotification{UserId: n.UserId, Id: n.UserId}

	return un.GetKey()
}
