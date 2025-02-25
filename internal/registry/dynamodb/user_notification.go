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
	"github.com/notifique/internal/registry"
	"github.com/notifique/internal/server"
	"github.com/notifique/internal/server/dto"
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

// GetUserNotifications retrieves user notifications from DynamoDB based on the provided filters.
// It returns a paginated list of user notifications.
//
// Parameters:
//   - ctx: The context for managing request deadlines and cancellations.
//   - filters: The filters to apply when querying user notifications.
//
// Returns:
//   - dto.Page[dto.UserNotification]: A paginated list of user notifications.
//   - error: An error if the operation fails.
func (s *Registry) GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) (dto.Page[dto.UserNotification], error) {

	page := dto.Page[dto.UserNotification]{}

	keyExp := expression.
		Key(UserNotificactionsHashKey).
		Equal(expression.Value(filters.UserId))

	builder := expression.
		NewBuilder().
		WithKeyCondition(keyExp)

	topicsFilter := makeInFilter("topic", filters.Topics)

	if topicsFilter != nil {
		builder.WithFilter(*topicsFilter)
	}

	expr, err := builder.Build()

	if err != nil {
		return page, fmt.Errorf("failed to build query - %w", err)
	}

	pageParams, err := makePageFilters(&userNotificationKey{}, filters.PageFilter)

	if err != nil {
		return page, fmt.Errorf("failed to make page params - %w", err)
	}

	queryInput := dynamodb.QueryInput{
		TableName:                 aws.String(UserNotificationsTable),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		FilterExpression:          expr.Filter(),
		ScanIndexForward:          aws.Bool(false),
		Limit:                     pageParams.Limit,
		ExclusiveStartKey:         pageParams.ExclusiveStartKey,
	}

	response, err := s.client.Query(ctx, &queryInput)

	if err != nil {
		return page, fmt.Errorf("failed to get notifications - %w", err)
	}

	var notifications []UserNotification
	err = attributevalue.UnmarshalListOfMaps(response.Items, &notifications)

	if err != nil {
		return page, fmt.Errorf("failed to unmarshall user notifications %w", err)
	}

	var nextToken *string = nil

	if len(response.LastEvaluatedKey) != 0 {
		key := userNotificationKey{}
		encoded, err := marshalNextToken(&key, response.LastEvaluatedKey)

		if err != nil {
			return page, err
		}

		nextToken = &encoded
	}

	result := make([]dto.UserNotification, 0, len(notifications))

	for _, notification := range notifications {
		un := dto.UserNotification{
			Id:        notification.Id,
			Title:     notification.Title,
			Contents:  notification.Contents,
			CreatedAt: notification.CreatedAt,
			Image:     notification.Image,
			ReadAt:    notification.ReadAt,
			Topic:     notification.Topic,
		}

		result = append(result, un)
	}

	page.NextToken = nextToken
	page.PrevToken = filters.NextToken
	page.ResultCount = len(result)
	page.Data = result

	return page, nil
}

func (s *Registry) SetReadStatus(ctx context.Context, userId, notificationId string) error {

	reatAt := time.Now().Format(time.RFC3339Nano)
	update := expression.Set(expression.Name("readAt"), expression.Value(reatAt))
	condEx := expression.AttributeExists(expression.Name(UserNotificactionsHashKey))
	expr, err := expression.NewBuilder().WithUpdate(update).WithCondition(condEx).Build()

	if err != nil {
		return fmt.Errorf("failed to make update query - %w", err)
	}

	notification := UserNotification{UserId: userId, Id: notificationId}
	key, err := notification.GetKey()

	if err != nil {
		return err
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                           aws.String(UserNotificationsTable),
		Key:                                 key,
		ExpressionAttributeNames:            expr.Names(),
		ExpressionAttributeValues:           expr.Values(),
		UpdateExpression:                    expr.Update(),
		ConditionExpression:                 expr.Condition(),
		ReturnValues:                        types.ReturnValueUpdatedNew,
		ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureNone,
	})

	if err != nil {
		target := &types.ConditionalCheckFailedException{}
		if errors.As(err, &target) {
			return server.EntityNotFound{
				Id:   notificationId,
				Type: registry.NotificationType,
			}
		}
		return fmt.Errorf("failed to set the read status - %w", err)
	}

	return nil
}
