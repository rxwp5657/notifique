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
	"github.com/notifique/service/internal"
	"github.com/notifique/service/internal/dto"
	"github.com/notifique/service/internal/registry"
	sdto "github.com/notifique/shared/dto"
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
func (r *Registry) GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) (sdto.Page[dto.UserNotification], error) {

	page := sdto.Page[dto.UserNotification]{}

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

	response, err := r.client.Query(ctx, &queryInput)

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

func (r *Registry) SetReadStatus(ctx context.Context, userId, notificationId string) error {

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

	_, err = r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
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
			return internal.EntityNotFound{
				Id:   notificationId,
				Type: registry.NotificationType,
			}
		}
		return fmt.Errorf("failed to set the read status - %w", err)
	}

	return nil
}

func (r *Registry) CreateNotifications(ctx context.Context, notifications []sdto.UserNotificationReq) ([]dto.UserNotification, error) {

	userNotifications := make([]dto.UserNotification, 0, len(notifications))

	if len(notifications) == 0 {
		return userNotifications, nil
	}

	var items []UserNotification

	for _, n := range notifications {

		id, err := uuid.NewV7()

		if err != nil {
			return userNotifications, fmt.Errorf("failed to generate uuid - %w", err)
		}

		item := UserNotification{
			Id:        id.String(),
			UserId:    n.UserId,
			Title:     n.Title,
			Contents:  n.Contents,
			CreatedAt: time.Now().Format(time.RFC3339Nano),
			Image:     n.Image,
			ReadAt:    nil,
			Topic:     n.Topic,
		}

		items = append(items, item)

		userNotification := dto.UserNotification{
			Id:        item.Id,
			Title:     item.Title,
			Contents:  item.Contents,
			CreatedAt: item.CreatedAt,
			Image:     item.Image,
			ReadAt:    item.ReadAt,
			Topic:     item.Topic,
		}

		userNotifications = append(userNotifications, userNotification)
	}

	reqItems, err := MakeBatchWriteRequest(UserNotificationsTable, items)

	if err != nil {
		return userNotifications, fmt.Errorf("failed to make batch write request - %w", err)
	}

	response, err := r.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: reqItems,
	})

	if err != nil {
		return userNotifications, fmt.Errorf("failed to create user notifications - %w", err)
	}

	if len(response.UnprocessedItems) != 0 {
		return userNotifications, fmt.Errorf("failed to process all items - %v", response.UnprocessedItems)
	}

	return userNotifications, nil
}
