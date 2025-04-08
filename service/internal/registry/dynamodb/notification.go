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
	NotificationsTable                        = "Notifications"
	NotificationHashKey                       = "id"
	NotificationStatusLogTable                = "NotificationStatusLogs"
	NotificationStatusLogHashKey              = "notificationId"
	NotificationStatusLogSortKey              = "statusDate"
	RecipientNotificationStatusLogTable       = "RecipientNotificationStatusLogs"
	RecipientNotificationStatusLogHashKey     = "notificationId"
	RecipientNotificationStatusLogSortKey     = "userId-channel"
	RecipientNotificationLatestStatusLogTable = "RecipientNotificationLatestStatusLogs"
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
	Image            *string           `dynamodbav:"image"`
	Topic            string            `dynamodbav:"topic"`
	Priority         string            `dynamodbav:"priority"`
	DistributionList *string           `dynamodbav:"distributionList"`
	Recipients       []string          `dynamodbav:"recipients"`
	Channels         []string          `dynamodbav:"channels"`
	Status           string            `dynamodbav:"status"`
	ContentsType     string            `dynamodbav:"contentType"`
}

type notificationSummary struct {
	Id           string `dynamodbav:"id"`
	Topic        string `dynamodbav:"topic"`
	CreatedAt    string `dynamodbav:"createdAt"`
	CreatedBy    string `dynamodbav:"createdBy"`
	Priority     string `dynamodbav:"priority"`
	Status       string `dynamodbav:"status"`
	ContentsType string `dynamodbav:"contentType"`
}

type recipientNotificationStatusLog struct {
	NotificationId string  `dynamodbav:"notificationId"`
	UserIdChannel  string  `dynamodbav:"userId-channel"`
	UserId         string  `dynamodbav:"userId"`
	Status         string  `dynamodbav:"status"`
	Channel        string  `dynamodbav:"channel"`
	StatusDate     string  `dynamodbav:"statusDate"`
	Error          *string `dynamodbav:"errorMsg"`
}

type notificationKey struct {
	Id string `dynamodbav:"id"`
}

type recipientNotificationLatestStatusKey struct {
	NotificationId string `dynamodbav:"notificationId"`
	UserIdChannel  string `dynamodbav:"userId-channel"`
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

func (n notificationKey) GetKey() (DynamoKey, error) {
	return Notification{Id: n.Id}.GetKey()
}

func (n recipientNotificationLatestStatusKey) GetKey() (DynamoKey, error) {
	key := make(DynamoKey)
	notificationId, err := attributevalue.Marshal(n.NotificationId)

	if err != nil {
		return key, fmt.Errorf("failed to make notification key - %w", err)
	}

	sortKey, err := attributevalue.Marshal(n.UserIdChannel)

	if err != nil {
		return key, fmt.Errorf("failed to make sort key - %w", err)
	}

	key[RecipientNotificationStatusLogHashKey] = notificationId
	key[RecipientNotificationStatusLogSortKey] = sortKey

	return key, nil
}

func (r *Registry) SaveNotification(ctx context.Context, createdBy string, notificationReq sdto.NotificationReq) (string, error) {

	if createdBy == "" {
		return "", fmt.Errorf("creator id cannot be empty")
	}

	id := uuid.NewString()

	channels := sdto.NotificationChannel("").
		ToStrSlice(notificationReq.Channels)

	contentsType := dto.Raw

	if notificationReq.RawContents == nil {
		contentsType = dto.Template
	}

	notification := Notification{
		Id:               id,
		CreatedBy:        createdBy,
		CreatedAt:        time.Now().Format(time.RFC3339),
		Image:            notificationReq.Image,
		Topic:            notificationReq.Topic,
		Priority:         string(notificationReq.Priority),
		DistributionList: notificationReq.DistributionList,
		Recipients:       notificationReq.Recipients,
		Channels:         channels,
		Status:           string(sdto.Created),
		ContentsType:     string(contentsType),
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

	log := sdto.NotificationStatusLog{
		NotificationId: id,
		Status:         sdto.Created,
		ErrorMsg:       nil,
	}

	err = r.UpdateNotificationStatus(ctx, log)

	if err != nil {
		return id, fmt.Errorf("failed to store notification status logs - %w", err)
	}

	return id, nil
}

func (r *Registry) notificationExists(ctx context.Context, notificationId string) (bool, error) {

	key, err := Notification{Id: notificationId}.GetKey()

	if err != nil {
		return false, fmt.Errorf("failed to make notification key - %w", err)
	}

	projExp := expression.
		ProjectionBuilder(expression.NamesList(expression.Name(NotificationHashKey)))

	expr, err := expression.
		NewBuilder().
		WithProjection(projExp).
		Build()

	if err != nil {
		return false, fmt.Errorf("failed to build expression - %w", err)
	}

	resp, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:                aws.String(NotificationsTable),
		Key:                      key,
		ProjectionExpression:     expr.Projection(),
		ExpressionAttributeNames: expr.Names(),
	})

	if err != nil {
		return false, fmt.Errorf("failed to check if notification exists - %w", err)
	}

	return len(resp.Item) != 0, nil
}

func (r *Registry) UpdateNotificationStatus(ctx context.Context, statusLog sdto.NotificationStatusLog) error {

	exists, err := r.notificationExists(ctx, statusLog.NotificationId)

	if err != nil {
		return fmt.Errorf("failed to check if notification exists - %w", err)
	}

	if !exists {
		return internal.EntityNotFound{Id: statusLog.NotificationId, Type: registry.NotificationType}
	}

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

func (r *Registry) GetNotificationStatus(ctx context.Context, id string) (sdto.NotificationStatus, error) {

	var status sdto.NotificationStatus

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
		return status, internal.EntityNotFound{Id: id, Type: registry.NotificationType}
	}

	tmp := struct {
		Status string `dynamodbav:"status"`
	}{}

	err = attributevalue.UnmarshalMap(resp.Item, &tmp)

	if err != nil {
		return status, fmt.Errorf("failed to unmarshal status - %w", err)
	}

	status = sdto.NotificationStatus(tmp.Status)

	return status, nil
}

func (r *Registry) DeleteNotification(ctx context.Context, id string) error {

	status, err := r.GetNotificationStatus(ctx, id)

	if err != nil && errors.As(err, &internal.EntityNotFound{}) {
		return nil
	} else if err != nil {
		return err
	}

	canDelete := registry.IsDeletableStatus(status)

	if !canDelete {
		return internal.InvalidNotificationStatus{
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

func (r *Registry) GetNotifications(ctx context.Context, filters sdto.PageFilter) (sdto.Page[dto.NotificationSummary], error) {

	page := sdto.Page[dto.NotificationSummary]{}

	projExp := expression.
		ProjectionBuilder(expression.NamesList(
			expression.Name("id"),
			expression.Name("topic"),
			expression.Name("createdAt"),
			expression.Name("createdBy"),
			expression.Name("priority"),
			expression.Name("status"),
			expression.Name("contentType"),
		))

	expr, err := expression.
		NewBuilder().
		WithProjection(projExp).
		Build()

	if err != nil {
		return page, fmt.Errorf("failed to build expression - %w", err)
	}

	pageParams, err := makePageFilters(notificationKey{}, filters)

	if err != nil {
		return page, fmt.Errorf("failed to make page params - %w", err)
	}

	input := &dynamodb.ScanInput{
		TableName:                aws.String(NotificationsTable),
		ProjectionExpression:     expr.Projection(),
		ExpressionAttributeNames: expr.Names(),
		Limit:                    pageParams.Limit,
		ExclusiveStartKey:        pageParams.ExclusiveStartKey,
	}

	resp, err := r.client.Scan(ctx, input)

	if err != nil {
		return page, fmt.Errorf("failed to retrieve notifications - %w", err)
	}

	var notificationsSummaries []notificationSummary
	err = attributevalue.UnmarshalListOfMaps(resp.Items, &notificationsSummaries)

	if err != nil {
		return page, fmt.Errorf("failed to unmarshall the notifications list - %w", err)
	}

	if len(resp.LastEvaluatedKey) != 0 {
		key := notificationKey{}
		encoded, err := marshalNextToken(&key, resp.LastEvaluatedKey)

		if err != nil {
			return page, fmt.Errorf("failed to encode next token - %w", err)
		}

		page.NextToken = &encoded
	}

	notifications := make([]dto.NotificationSummary, 0, len(notificationsSummaries))

	for _, n := range notificationsSummaries {
		notifications = append(notifications, dto.NotificationSummary{
			Id:           n.Id,
			Topic:        n.Topic,
			CreatedAt:    n.CreatedAt,
			CreatedBy:    n.CreatedBy,
			Priority:     sdto.NotificationPriority(n.Priority),
			Status:       sdto.NotificationStatus(n.Status),
			ContentsType: dto.NotificationContentsType(n.ContentsType),
		})
	}

	page.PrevToken = filters.NextToken
	page.ResultCount = len(notifications)
	page.Data = notifications

	return page, nil
}

func (r *Registry) GetNotification(ctx context.Context, notificationId string) (dto.NotificationResp, error) {

	notificationResp := dto.NotificationResp{}

	key, err := Notification{Id: notificationId}.GetKey()

	if err != nil {
		return notificationResp, fmt.Errorf("failed to make notification key - %w", err)
	}

	resp, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(NotificationsTable),
		Key:       key,
	})

	if err != nil {
		return notificationResp, fmt.Errorf("failed to retrieve notification - %w", err)
	}

	if len(resp.Item) == 0 {
		return notificationResp, internal.EntityNotFound{Id: notificationId, Type: registry.NotificationType}
	}

	var notification Notification
	err = attributevalue.UnmarshalMap(resp.Item, &notification)

	if err != nil {
		return notificationResp, fmt.Errorf("failed to unmarshal notification - %w", err)
	}

	channels := make([]sdto.NotificationChannel, 0, len(notification.Channels))

	for _, c := range notification.Channels {
		channels = append(channels, sdto.NotificationChannel(c))
	}

	notificationResp.NotificationReq = sdto.NotificationReq{
		Image:            notification.Image,
		Topic:            notification.Topic,
		Priority:         sdto.NotificationPriority(notification.Priority),
		DistributionList: notification.DistributionList,
		Recipients:       notification.Recipients,
		Channels:         channels,
	}

	if notification.ContentsType == string(dto.Raw) {
		notificationResp.NotificationReq.RawContents = &sdto.RawContents{
			Title:    notification.RawContents.Title,
			Contents: notification.RawContents.Contents,
		}
	} else {
		numVariables := len(notification.TemplateContents.Variables)
		variables := make([]sdto.TemplateVariableContents, 0, numVariables)

		for _, v := range notification.TemplateContents.Variables {
			variables = append(variables, sdto.TemplateVariableContents{
				Name:  v.Name,
				Value: v.Value,
			})
		}

		notificationResp.NotificationReq.TemplateContents = &sdto.TemplateContents{
			Id:        notification.TemplateContents.Id,
			Variables: variables,
		}
	}

	return notificationResp, nil
}

func makeUserIdChannel(userId, channel string) string {
	return fmt.Sprintf("%s-%s", userId, channel)
}

func (r *Registry) UpsertRecipientNotificationStatuses(ctx context.Context, notificationId string, statuses []sdto.RecipientNotificationStatus) error {

	exists, err := r.notificationExists(ctx, notificationId)

	if err != nil {
		return fmt.Errorf("failed to check if notification exists - %w", err)
	}

	if !exists {
		return internal.EntityNotFound{Id: notificationId, Type: registry.NotificationType}
	}

	if len(statuses) == 0 {
		return nil
	}

	dynamoStatuses := make([]recipientNotificationStatusLog, 0, len(statuses))

	for _, status := range statuses {
		dynamoStatuses = append(dynamoStatuses, recipientNotificationStatusLog{
			NotificationId: notificationId,
			UserId:         status.UserId,
			Channel:        status.Channel,
			UserIdChannel:  makeUserIdChannel(status.UserId, status.Channel),
			Status:         status.Status,
			StatusDate:     time.Now().Format(time.RFC3339Nano),
			Error:          status.ErrMsg,
		})
	}

	reqItems, err := MakeBatchWriteRequest(RecipientNotificationStatusLogTable, dynamoStatuses)

	if err != nil {
		return fmt.Errorf("failed to make batch write request - %w", err)
	}

	_, err = r.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: reqItems,
	})

	if err != nil {
		return fmt.Errorf("failed to batch write user notification statuses - %w", err)
	}

	updates := make([]types.TransactWriteItem, 0, len(statuses))

	for _, status := range dynamoStatuses {

		item, err := attributevalue.MarshalMap(status)

		if err != nil {
			return fmt.Errorf("failed to marshall user notification status log - %w", err)
		}

		put := types.Put{
			TableName: aws.String(RecipientNotificationLatestStatusLogTable),
			Item:      item,
		}

		updates = append(updates, types.TransactWriteItem{
			Put: &put,
		})
	}

	_, err = r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: updates,
	})

	if err != nil {
		return fmt.Errorf("failed to update latest user notification statuses - %w", err)
	}

	return nil
}

func (r *Registry) GetRecipientNotificationStatuses(ctx context.Context, notificationId string, filters sdto.NotificationRecipientStatusFilters) (sdto.Page[sdto.RecipientNotificationStatus], error) {

	page := sdto.Page[sdto.RecipientNotificationStatus]{}

	projExp := expression.
		ProjectionBuilder(expression.NamesList(
			expression.Name("notificationId"),
			expression.Name("userId"),
			expression.Name("status"),
			expression.Name("errorMsg"),
			expression.Name("channel"),
		))

	keyExpr := expression.KeyEqual(
		expression.Key(RecipientNotificationStatusLogHashKey),
		expression.Value(notificationId))

	builder := expression.NewBuilder().
		WithKeyCondition(keyExpr).
		WithProjection(projExp)

	channelFilters := makeInFilter("channel", filters.Channels)
	statusFilters := makeInFilter("status", filters.Statuses)

	if channelFilters != nil && statusFilters != nil {
		builder = builder.WithFilter(expression.And(*channelFilters, *statusFilters))
	} else if channelFilters != nil {
		builder = builder.WithFilter(*channelFilters)
	} else if statusFilters != nil {
		builder = builder.WithFilter(*statusFilters)
	}

	expr, err := builder.Build()

	if err != nil {
		return page, fmt.Errorf("failed to build expression - %w", err)
	}

	pageParams, err := makePageFilters(recipientNotificationLatestStatusKey{}, filters.PageFilter)

	if err != nil {
		return page, fmt.Errorf("failed to make page params - %w", err)
	}

	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(RecipientNotificationLatestStatusLogTable),
		ProjectionExpression:      expr.Projection(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		FilterExpression:          expr.Filter(),
		Limit:                     pageParams.Limit,
		ExclusiveStartKey:         pageParams.ExclusiveStartKey,
	}

	response, err := r.client.Query(ctx, queryInput)

	if err != nil {
		return page, fmt.Errorf("failed to get user notification statuses - %w", err)
	}

	var statuses []recipientNotificationStatusLog
	err = attributevalue.UnmarshalListOfMaps(response.Items, &statuses)

	if err != nil {
		return page, fmt.Errorf("failed to unmarshall user notification statuses - %w", err)
	}

	var nextToken *string = nil

	if len(response.LastEvaluatedKey) != 0 {
		key := recipientNotificationLatestStatusKey{}
		encoded, err := marshalNextToken(&key, response.LastEvaluatedKey)
		if err != nil {
			return page, fmt.Errorf("failed to encode next token - %w", err)
		}
		nextToken = &encoded
	}

	userStatuses := make([]sdto.RecipientNotificationStatus, 0, len(statuses))

	for _, s := range statuses {
		userStatuses = append(userStatuses, sdto.RecipientNotificationStatus{
			UserId:  s.UserId,
			Status:  s.Status,
			ErrMsg:  s.Error,
			Channel: s.Channel,
		})
	}

	page.NextToken = nextToken
	page.PrevToken = filters.NextToken
	page.ResultCount = len(userStatuses)
	page.Data = userStatuses

	return page, nil
}
