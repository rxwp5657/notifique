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
	Id     string `dynamodbav:"id" json:"id"`
	UserId string `dynamodbav:"userId" json:"userId"`
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

func (n *userNotificationKey) GetKey() (DynamoKey, error) {
	un := UserNotification{
		UserId: n.UserId,
		Id:     n.Id,
	}

	return un.GetKey()
}
