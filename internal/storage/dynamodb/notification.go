package storage

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

const (
	NotificationsTable           = "Notifications"
	NotificationHashKey          = "id"
	NotificationStatusLogTable   = "NotificationStatusLogs"
	NotificationStatusLogHashKey = "notificationId"
	NotificationStatusLogSortKey = "statusDate"
)

type notificationStatusLog struct {
	NotificationId string  `dynamodbav:"notificationId"`
	Status         string  `dynamodbav:"status"`
	StatusDate     string  `dynamodbav:"statusDate"`
	Error          *string `dynamodbav:"errorMsg"`
}

type notification struct {
	Id               string   `dynamodbav:"id"`
	CreatedBy        string   `dynamodbav:"createdBy"`
	CreatedAt        string   `dynamodbav:"createdAt"`
	Title            string   `dynamodbav:"title"`
	Contents         string   `dynamodbav:"contents"`
	Image            *string  `dynamodbav:"image"`
	Topic            string   `dynamodbav:"topic"`
	Priority         string   `dynamodbav:"priority"`
	DistributionList *string  `dynamodbav:"distributionList"`
	Recipients       []string `dynamodbav:"recipients"`
	Channels         []string `dynamodbav:"channels"`
	Status           string   `dynamodbav:"status"`
}

func (n notification) GetKey() (DynamoKey, error) {
	key := make(DynamoKey)

	notificationId, err := attributevalue.Marshal(n.Id)

	if err != nil {
		return key, fmt.Errorf("failed to make notification key - %w", err)
	}

	key["id"] = notificationId

	return key, nil
}
