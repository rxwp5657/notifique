package dynamoregistry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/notifique/internal/registry"
	"github.com/notifique/internal/server"
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

type rawContents struct {
	Title    string `dynamodbav:"title"`
	Contents string `dynamodbav:"contents"`
}

type templateVariableContents struct {
	Name  string `dynamodbav:"name"`
	Value string `dynamodbav:"value"`
}

type templateContents struct {
	Id        string                     `dynamodbav:"id"`
	Variables []templateVariableContents `dynamodbav:"variables"`
}

type Notification struct {
	Id               string            `dynamodbav:"id"`
	RawContents      *rawContents      `dynamodbav:"rawContents"`
	TemplateContents *templateContents `dynamodbav:"templateContents"`
	CreatedBy        string            `dynamodbav:"createdBy"`
	CreatedAt        string            `dynamodbav:"createdAt"`
	Title            string            `dynamodbav:"title"`
	Contents         string            `dynamodbav:"contents"`
	Image            *string           `dynamodbav:"image"`
	Topic            string            `dynamodbav:"topic"`
	Priority         string            `dynamodbav:"priority"`
	DistributionList *string           `dynamodbav:"distributionList"`
	Recipients       []string          `dynamodbav:"recipients"`
	Channels         []string          `dynamodbav:"channels"`
	Status           string            `dynamodbav:"status"`
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

func (r *Registry) SaveNotification(ctx context.Context, createdBy string, notificationReq dto.NotificationReq) (string, error) {

	if createdBy == "" {
		return "", fmt.Errorf("creator id cannot be empty")
	}

	id := uuid.NewString()

	notification := Notification{
		Id:               id,
		CreatedBy:        createdBy,
		CreatedAt:        time.Now().Format(time.RFC3339),
		Image:            notificationReq.Image,
		Topic:            notificationReq.Topic,
		Priority:         notificationReq.Priority,
		DistributionList: notificationReq.DistributionList,
		Recipients:       notificationReq.Recipients,
		Channels:         notificationReq.Channels,
		Status:           string(c.Created),
	}

	if notificationReq.RawContents != nil {
		notification.RawContents = &rawContents{
			Title:    notificationReq.RawContents.Title,
			Contents: notificationReq.RawContents.Contents,
		}
	} else {
		numVariables := len(notificationReq.TemplateContents.Variables)
		variables := make([]templateVariableContents, 0, numVariables)

		for _, v := range notificationReq.TemplateContents.Variables {
			variables = append(variables, templateVariableContents{
				Name:  v.Name,
				Value: v.Value,
			})
		}

		notification.TemplateContents = &templateContents{
			Id:        notificationReq.TemplateContents.Id,
			Variables: variables,
		}
	}

	item, err := attributevalue.MarshalMap(notification)

	if err != nil {
		return "", fmt.Errorf("failed to marshall notification - %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
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

	err = r.UpdateNotificationStatus(ctx, log)

	if err != nil {
		return id, fmt.Errorf("failed to store notification status logs - %w", err)
	}

	return id, nil
}

func (r *Registry) UpdateNotificationStatus(ctx context.Context, statusLog c.NotificationStatusLog) error {

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

	_, err = r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
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

func (r *Registry) GetNotificationStatus(ctx context.Context, id string) (c.NotificationStatus, error) {

	var status c.NotificationStatus

	key, err := Notification{Id: id}.GetKey()

	if err != nil {
		return status, fmt.Errorf("failed to make notification key - %w", err)
	}

	projExp := expression.
		ProjectionBuilder(expression.NamesList(expression.Name("status")))

	expr, err := expression.
		NewBuilder().
		WithProjection(projExp).
		Build()

	if err != nil {
		return status, fmt.Errorf("failed to build expression - %w", err)
	}

	resp, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:                aws.String(NotificationsTable),
		Key:                      key,
		ProjectionExpression:     expr.Projection(),
		ExpressionAttributeNames: expr.Names(),
	})

	if err != nil {
		return status, fmt.Errorf("failed to retrieve the notification status - %w", err)
	}

	if len(resp.Item) == 0 {
		return status, server.EntityNotFound{Id: id, Type: registry.NotificationType}
	}

	tmp := struct {
		Status string `dynamodbav:"status"`
	}{}

	err = attributevalue.UnmarshalMap(resp.Item, &tmp)

	if err != nil {
		return status, fmt.Errorf("failed to unmarshal status - %w", err)
	}

	status = c.NotificationStatus(tmp.Status)

	return status, nil
}

func (r *Registry) DeleteNotification(ctx context.Context, id string) error {

	status, err := r.GetNotificationStatus(ctx, id)

	if err != nil && errors.As(err, &server.EntityNotFound{}) {
		return nil
	} else if err != nil {
		return err
	}

	canDelete := registry.IsDeletableStatus(status)

	if !canDelete {
		return server.InvalidNotificationStatus{
			Id:     id,
			Status: string(status),
		}
	}

	key, err := Notification{Id: id}.GetKey()

	if err != nil {
		return fmt.Errorf("failed to make notification key - %w", err)
	}

	_, err = r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(NotificationsTable),
		Key:       key,
	})

	if err != nil {
		return fmt.Errorf("failed to delete notification - %w", err)
	}

	return nil
}
