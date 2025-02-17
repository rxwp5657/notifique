package dynamoregistry

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	c "github.com/notifique/internal/server/controllers"
	"github.com/notifique/internal/server/dto"
)

const (
	NotificationsTable           = "Notifications"
	NotificationHashKey          = "id"
	NotificationStatusLogTable   = "NotificationStatusLogs"
	NotificationStatusLogHashKey = "notificationId"
	NotificationStatusLogSortKey = "statusDate"
)

type NotificationStatusLog struct {
	NotificationId string  `dynamodbav:"notificationId"`
	Status         string  `dynamodbav:"status"`
	StatusDate     string  `dynamodbav:"statusDate"`
	Error          *string `dynamodbav:"errorMsg"`
}

type Notification struct {
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

func (n Notification) GetKey() (DynamoKey, error) {
	key := make(DynamoKey)

	notificationId, err := attributevalue.Marshal(n.Id)

	if err != nil {
		return key, fmt.Errorf("failed to make notification key - %w", err)
	}

	key["id"] = notificationId

	return key, nil
}

func (s *Registry) SaveNotification(ctx context.Context, createdBy string, notificationReq dto.NotificationReq) (string, error) {

	if createdBy == "" {
		return "", fmt.Errorf("creator id cannot be empty")
	}

	id := uuid.NewString()

	notification := Notification{
		Id:               id,
		CreatedBy:        createdBy,
		CreatedAt:        time.Now().Format(time.RFC3339),
		Title:            notificationReq.Title,
		Contents:         notificationReq.Contents,
		Image:            notificationReq.Image,
		Topic:            notificationReq.Topic,
		Priority:         notificationReq.Priority,
		DistributionList: notificationReq.DistributionList,
		Recipients:       notificationReq.Recipients,
		Channels:         notificationReq.Channels,
		Status:           string(c.Created),
	}

	item, err := attributevalue.MarshalMap(notification)

	if err != nil {
		return "", fmt.Errorf("failed to marshall notification - %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(NotificationsTable),
		Item:      item,
	})

	if err != nil {
		return "", fmt.Errorf("failed to store notification - %w", err)
	}

	log := c.NotificationStatusLog{
		NotificationId: id,
		Status:         c.Created,
		ErrorMsg:       nil,
	}

	err = s.UpdateNotificationStatus(ctx, log)

	if err != nil {
		return id, fmt.Errorf("failed to store notification status logs - %w", err)
	}

	return id, nil
}

func (s *Registry) UpdateNotificationStatus(ctx context.Context, statusLog c.NotificationStatusLog) error {

	update := expression.Set(expression.Name("status"), expression.Value((statusLog.Status)))
	condEx := expression.AttributeExists(expression.Name(NotificationHashKey))
	expr, err := expression.NewBuilder().WithUpdate(update).WithCondition(condEx).Build()

	if err != nil {
		return fmt.Errorf("failed to make update query - %w", err)
	}

	n := Notification{Id: statusLog.NotificationId}
	notificationKey, err := n.GetKey()

	if err != nil {
		return err
	}

	log := NotificationStatusLog{
		NotificationId: statusLog.NotificationId,
		Status:         string(statusLog.Status),
		StatusDate:     time.Now().Format(time.RFC3339Nano),
		Error:          statusLog.ErrorMsg,
	}

	item, err := attributevalue.MarshalMap(log)

	if err != nil {
		return fmt.Errorf("failed to marshal notification status log - %w", err)
	}

	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{{
			Put: &types.Put{
				TableName: aws.String(NotificationStatusLogTable),
				Item:      item,
			}}, {
			Update: &types.Update{
				TableName:                           aws.String(NotificationsTable),
				Key:                                 notificationKey,
				ExpressionAttributeNames:            expr.Names(),
				ExpressionAttributeValues:           expr.Values(),
				UpdateExpression:                    expr.Update(),
				ConditionExpression:                 expr.Condition(),
				ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureNone,
			},
		}},
	})

	if err != nil {
		return fmt.Errorf("failed to insert notification status log or update notification status - %w", err)
	}

	return nil
}
