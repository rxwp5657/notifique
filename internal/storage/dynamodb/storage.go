package dynamostorage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	c "github.com/notifique/controllers"
	"github.com/notifique/dto"
	"github.com/notifique/internal"
)

type DynamoDBAPI interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)

	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
	TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)

	DeleteTable(ctx context.Context, params *dynamodb.DeleteTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteTableOutput, error)
}

type Storage struct {
	client DynamoDBAPI
}

type DynamoPrimaryKey interface {
	GetKey() (DynamoKey, error)
}

type DynamoPageParams struct {
	Limit             *int32
	ExclusiveStartKey map[string]types.AttributeValue
}

type DynamoClientConfig struct {
	BaseEndpoint *string
	Region       *string
}

type DynamoConfigurator interface {
	GetDynamoClientConfig() DynamoClientConfig
}

type DynamoKey map[string]types.AttributeValue
type DynamoObj map[string]types.AttributeValue
type BatchWriteRequest map[string][]types.WriteRequest

func marshallNextToken[T any](key *T, lastEvaluatedKey DynamoKey) (string, error) {
	err := attributevalue.UnmarshalMap(lastEvaluatedKey, &key)

	if err != nil {
		return "", fmt.Errorf("failed to unmarshall last evaluated key - %w", err)
	}

	jsonMarshalled, err := json.Marshal(key)

	if err != nil {
		return "", fmt.Errorf("failed to json marshall last evaluated key - %w", err)
	}

	return base64.StdEncoding.EncodeToString([]byte(jsonMarshalled)), nil
}

func unmarshallNextToken[T any](nextToken string, key *T) error {
	decoded, err := base64.StdEncoding.DecodeString(nextToken)

	if err != nil {
		return fmt.Errorf("failed to decode base64 nextToken - %w", err)
	}

	err = json.Unmarshal(decoded, &key)

	if err != nil {
		return fmt.Errorf("failed to json unmarshall nextToken - %w", err)
	}

	return nil
}

func MakeBatchWriteRequest[T any](table string, data []T) (BatchWriteRequest, error) {
	requests := make([]types.WriteRequest, 0, len(data))

	for _, d := range data {
		item, err := attributevalue.MarshalMap(d)

		if err != nil {
			return BatchWriteRequest{}, fmt.Errorf("failed to marshall - %w", err)
		}

		requests = append(requests, types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: item,
			},
		})
	}

	batchRequest := BatchWriteRequest{
		table: requests,
	}

	return batchRequest, nil
}

func (s *Storage) getUserConfig(ctx context.Context, userId string) (*UserConfig, error) {

	tmpConfig := UserConfig{UserId: userId}
	key, err := tmpConfig.GetKey()

	if err != nil {
		return nil, err
	}

	resp, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(UserConfigTable),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get the user config - %w", err)
	}

	if len(resp.Item) == 0 {
		return nil, nil
	}

	config := UserConfig{}

	err = attributevalue.UnmarshalMap(resp.Item, &config)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall the user config - %w", err)
	}

	return &config, nil
}

func (s *Storage) createUserConfig(ctx context.Context, userId string) (*UserConfig, error) {
	config := UserConfig{
		UserId:      userId,
		EmailConfig: ChannelConfig{OptIn: true, SnoozeUntil: nil},
		SMSConfig:   ChannelConfig{OptIn: true, SnoozeUntil: nil},
		InAppConfig: ChannelConfig{OptIn: true, SnoozeUntil: nil},
	}

	item, err := attributevalue.MarshalMap(config)

	if err != nil {
		return nil, fmt.Errorf("failed to marshall the user config - %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(UserConfigTable),
		Item:      item,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to store notification - %w", err)
	}

	return &config, nil
}

func (s *Storage) SaveNotification(ctx context.Context, createdBy string, notificationReq dto.NotificationReq) (string, error) {

	id := uuid.NewString()

	notification := Notification{
		Id:               id,
		CreatedBy:        createdBy,
		CreatedAt:        time.Now().Format(time.RFC3339Nano),
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

func makeInFilter(expName string, values []string) *expression.ConditionBuilder {

	if len(values) == 0 {
		return nil
	}

	filters := make([]expression.OperandBuilder, 0, len(values))

	for _, v := range values {
		filters = append(filters, expression.Value(v))
	}

	first := filters[0]
	rest := make([]expression.OperandBuilder, 0)

	if len(filters) > 1 {
		rest = filters[1:]
	}

	cond := expression.In(expression.Name(expName), first, rest...)
	return &cond
}

func makePageFilters[T DynamoPrimaryKey](key T, filters dto.PageFilter) (DynamoPageParams, error) {

	params := DynamoPageParams{}

	params.Limit = aws.Int32(internal.PageSize)

	if filters.MaxResults != nil {
		limit := int32(*filters.MaxResults)
		params.Limit = &limit
	}

	if filters.NextToken != nil {
		err := unmarshallNextToken(*filters.NextToken, &key)

		if err != nil {
			return params, fmt.Errorf("failed to unmarshall token - %w", err)
		}

		dynamoDBKey, err := key.GetKey()

		if err != nil {
			return params, fmt.Errorf("failed to get model key - %w", err)
		}

		params.ExclusiveStartKey = dynamoDBKey
	}

	return params, nil
}

func (s *Storage) GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) (dto.Page[dto.UserNotification], error) {

	page := dto.Page[dto.UserNotification]{}

	keyExp := expression.Key(UserNotificactionsHashKey).Equal(expression.Value(filters.UserId))
	builder := expression.NewBuilder().WithKeyCondition(keyExp)

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
		encoded, err := marshallNextToken(&key, response.LastEvaluatedKey)

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

func (s *Storage) GetUserConfig(ctx context.Context, userId string) (dto.UserConfig, error) {

	config, err := s.getUserConfig(ctx, userId)

	if err != nil {
		return dto.UserConfig{}, err
	}

	if config == nil {
		config, err = s.createUserConfig(ctx, userId)

		if err != nil {
			return dto.UserConfig{}, err
		}
	}

	cfg := dto.UserConfig{
		EmailConfig: dto.ChannelConfig{
			OptIn:       config.EmailConfig.OptIn,
			SnoozeUntil: config.EmailConfig.SnoozeUntil,
		},
		SMSConfig: dto.ChannelConfig{
			OptIn:       config.SMSConfig.OptIn,
			SnoozeUntil: config.SMSConfig.SnoozeUntil,
		},
		InAppConfig: dto.ChannelConfig{
			OptIn:       config.InAppConfig.OptIn,
			SnoozeUntil: config.InAppConfig.SnoozeUntil,
		},
	}

	return cfg, nil
}

func (s *Storage) SetReadStatus(ctx context.Context, userId, notificationId string) error {

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
			return internal.NotificationNotFound{NotificationId: notificationId}
		}
		return fmt.Errorf("failed to set the read status - %w", err)
	}

	return nil
}

func (s *Storage) UpdateUserConfig(ctx context.Context, userId string, config dto.UserConfig) error {

	usrCfg, err := s.getUserConfig(ctx, userId)

	if err != nil {
		return err
	}

	if usrCfg == nil {
		_, err := s.createUserConfig(ctx, userId)

		if err != nil {
			return err
		}
	}

	makeKeyFormatter := func(key string) func(string) expression.NameBuilder {
		return func(subKey string) expression.NameBuilder {
			return expression.Name(fmt.Sprintf("%v.%v", key, subKey))
		}
	}

	emailFmt := makeKeyFormatter(UserConfigEmailKey)
	smsFmt := makeKeyFormatter(UserConfigSmsKey)
	inAppFmt := makeKeyFormatter(UserConfigInAppKey)

	update := expression.Set(emailFmt(UserConfigOptIn), expression.Value(config.EmailConfig.OptIn))
	update.Set(emailFmt(UserConfigSnoozeUntil), expression.Value(config.EmailConfig.SnoozeUntil))
	update.Set(smsFmt(UserConfigOptIn), expression.Value(config.SMSConfig.OptIn))
	update.Set(smsFmt(UserConfigSnoozeUntil), expression.Value(config.SMSConfig.SnoozeUntil))
	update.Set(inAppFmt(UserConfigOptIn), expression.Value(config.InAppConfig.OptIn))
	update.Set(inAppFmt(UserConfigSnoozeUntil), expression.Value(config.InAppConfig.SnoozeUntil))

	expr, err := expression.NewBuilder().WithUpdate(update).Build()

	if err != nil {
		return fmt.Errorf("failed to make update query - %w", err)
	}

	tmpConfig := UserConfig{UserId: userId}
	key, err := tmpConfig.GetKey()

	if err != nil {
		return err
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(UserConfigTable),
		Key:                       key,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
		ConditionExpression:       expr.Condition(),
		ReturnValues:              types.ReturnValueUpdatedNew,
	})

	if err != nil {
		return fmt.Errorf("failed to set the read status - %w", err)
	}

	return nil
}

func (s *Storage) addRecipients(ctx context.Context, recipients []DistListRecipient) (int, error) {

	requestItems, err := MakeBatchWriteRequest(DistListRecipientsTable, recipients)

	if err != nil {
		return 0, fmt.Errorf("failed create batch request for DL - %w", err)
	}

	_, err = s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: requestItems,
	})

	if err != nil {
		return 0, fmt.Errorf("failed to add recipients to dl - %w", err)
	}

	return len(recipients), nil
}

func (s *Storage) queryDistListSummary(ctx context.Context, listName string) (*map[string]types.AttributeValue, error) {
	summary := DistListSummary{Name: listName}
	key, err := summary.GetKey()

	if err != nil {
		return nil, fmt.Errorf("failed to get distribution list key - %w", err)
	}

	resp, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(DistListSummaryTable),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get distribution list - %w", err)
	}

	return &resp.Item, nil
}

func (s *Storage) getDistListSummary(ctx context.Context, listName string) (*DistListSummary, error) {
	resp, err := s.queryDistListSummary(ctx, listName)

	if err != nil {
		return nil, err
	}

	summary := DistListSummary{}

	err = attributevalue.UnmarshalMap(*resp, &summary)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall distribution list - %w", err)
	}

	return &summary, nil
}

func (s *Storage) distListExists(ctx context.Context, listName string) (bool, error) {

	resp, err := s.queryDistListSummary(ctx, listName)

	if err != nil {
		return false, err
	}

	return len(*resp) != 0, nil
}

func (s *Storage) deleteSummary(ctx context.Context, listName string) error {
	summary := DistListSummary{Name: listName}
	key, err := summary.GetKey()

	if err != nil {
		return err
	}

	_, err = s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(DistListSummaryTable),
		Key:       key,
	})

	return err
}

func (s *Storage) CreateDistributionList(ctx context.Context, dlReq dto.DistributionList) error {

	exists, err := s.distListExists(ctx, dlReq.Name)

	if err != nil {
		return fmt.Errorf("failed to check for list existence - %w", err)
	}

	if exists {
		return internal.DistributionListAlreadyExists{Name: dlReq.Name}
	}

	summary := DistListSummary{
		Name:          dlReq.Name,
		NumRecipients: len(dlReq.Recipients),
	}

	marshalled, err := attributevalue.MarshalMap(summary)

	if err != nil {
		return fmt.Errorf("failed to marshall distribution list summary - %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(DistListSummaryTable),
		Item:      marshalled,
	})

	if err != nil {
		return fmt.Errorf("failed to create summary - %w", err)
	}

	recipients := make([]DistListRecipient, 0, len(dlReq.Recipients))

	for _, r := range dlReq.Recipients {
		recipients = append(recipients, DistListRecipient{
			DistListName: dlReq.Name,
			UserId:       r,
		})
	}

	if len(recipients) == 0 {
		return nil
	}

	_, recipientsErr := s.addRecipients(ctx, recipients)

	if recipientsErr != nil {
		recipientsErr = fmt.Errorf("failed to add recipients to list - %w", err)
		summaryError := s.deleteSummary(ctx, dlReq.Name)

		if summaryError != nil {
			summaryError = fmt.Errorf("failed to delete dist list summary - %w", summaryError)
		}

		err = errors.Join(recipientsErr, summaryError)
	}

	return err
}

func (s *Storage) GetDistributionLists(ctx context.Context, filters dto.PageFilter) (dto.Page[dto.DistributionListSummary], error) {

	page := dto.Page[dto.DistributionListSummary]{}

	pageParams, err := makePageFilters(&DistListSummaryKey{}, filters)

	if err != nil {
		return page, fmt.Errorf("failed to make page params - %w", err)
	}

	scanInput := dynamodb.ScanInput{
		TableName:         aws.String(DistListSummaryTable),
		Limit:             pageParams.Limit,
		ExclusiveStartKey: pageParams.ExclusiveStartKey,
	}

	response, err := s.client.Scan(ctx, &scanInput)

	if err != nil {
		return page, fmt.Errorf("failed to get the summaries - %w", err)
	}

	var summaries []DistListSummary
	err = attributevalue.UnmarshalListOfMaps(response.Items, &summaries)

	if err != nil {
		return page, fmt.Errorf("failed to unmarshall the summaries - %w", err)
	}

	var nextToken *string = nil

	if len(response.LastEvaluatedKey) != 0 {
		key := DistListSummaryKey{}
		encoded, err := marshallNextToken(&key, response.LastEvaluatedKey)

		if err != nil {
			return page, err
		}

		nextToken = &encoded
	}

	result := make([]dto.DistributionListSummary, 0, len(summaries))

	for _, summary := range summaries {
		s := dto.DistributionListSummary{
			Name:               summary.Name,
			NumberOfRecipients: summary.NumRecipients,
		}

		result = append(result, s)
	}

	page.PrevToken = filters.NextToken
	page.NextToken = nextToken
	page.ResultCount = len(summaries)
	page.Data = result

	return page, nil
}

func (s *Storage) deleteRecipients(ctx context.Context, recipients []DistListRecipient) (int, error) {

	if len(recipients) == 0 {
		return 0, nil
	}

	deleteReq := make([]types.WriteRequest, 0, len(recipients))

	for _, DistListRecipient := range recipients {
		key, _ := DistListRecipient.GetKey()
		deleteReq = append(deleteReq, types.WriteRequest{
			DeleteRequest: &types.DeleteRequest{
				Key: key,
			},
		})
	}

	requestItems := map[string][]types.WriteRequest{
		DistListRecipientsTable: deleteReq,
	}

	_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: requestItems,
	})

	if err != nil {
		return 0, fmt.Errorf("failed to delete recipients of dl - %w", err)
	}

	return len(recipients), nil
}

func (s *Storage) DeleteDistributionList(ctx context.Context, listName string) error {

	keyEx := expression.Key(DistListRecipientHashKey).Equal(expression.Value(listName))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()

	if err != nil {
		return fmt.Errorf("failed to create expression - %w", err)
	}

	queryPaginator := dynamodb.NewQueryPaginator(s.client, &dynamodb.QueryInput{
		TableName:                 aws.String(DistListRecipientsTable),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
	})

	for queryPaginator.HasMorePages() {
		resp, err := queryPaginator.NextPage(ctx)

		if err != nil {
			return fmt.Errorf("failed to retrieve user page - %w", err)
		}

		var recipients []DistListRecipient
		err = attributevalue.UnmarshalListOfMaps(resp.Items, &recipients)

		if err != nil {
			return fmt.Errorf("failed to unmarshal distribution list - %w", err)
		}

		_, err = s.deleteRecipients(ctx, recipients)

		if err != nil {
			return err
		}
	}

	err = s.deleteSummary(ctx, listName)

	if err != nil {
		return fmt.Errorf("failed to delete summary - %w", err)
	}

	return nil
}

func (s *Storage) GetRecipients(ctx context.Context, distlistName string, filters dto.PageFilter) (dto.Page[string], error) {

	page := dto.Page[string]{}

	exists, err := s.distListExists(ctx, distlistName)

	if err != nil {
		return page, fmt.Errorf("failed to check if distribution list exists - %w", err)
	}

	if !exists {
		return page, internal.DistributionListNotFound{Name: distlistName}
	}

	keyExp := expression.Key(DistListRecipientHashKey).Equal(expression.Value(distlistName))
	projExp := expression.NamesList(expression.Name(DistListRecipientSortKey))

	builder := expression.NewBuilder()
	expr, err := builder.WithKeyCondition(keyExp).WithProjection(projExp).Build()

	if err != nil {
		return page, fmt.Errorf("failed to build query - %w", err)
	}

	pageParams, err := makePageFilters(&DistListRecipient{}, filters)

	if err != nil {
		return page, fmt.Errorf("failed to make page params - %w", err)
	}

	queryInput := dynamodb.QueryInput{
		TableName:                 aws.String(DistListRecipientsTable),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		ProjectionExpression:      expr.Projection(),
		Limit:                     pageParams.Limit,
		ExclusiveStartKey:         pageParams.ExclusiveStartKey,
	}

	response, err := s.client.Query(ctx, &queryInput)

	if err != nil {
		return page, fmt.Errorf("failed to get recipients - %w", err)
	}

	var recipients []DistListRecipient
	err = attributevalue.UnmarshalListOfMaps(response.Items, &recipients)

	if err != nil {
		return page, fmt.Errorf("failed to unmarshall recipients - %w", err)
	}

	var nextToken *string = nil

	if len(response.LastEvaluatedKey) != 0 {
		key := DistListRecipient{}
		encoded, err := marshallNextToken(&key, response.LastEvaluatedKey)

		if err != nil {
			return page, err
		}

		nextToken = &encoded
	}

	result := make([]string, 0, len(recipients))

	for _, r := range recipients {
		result = append(result, r.UserId)
	}

	page.NextToken = nextToken
	page.PrevToken = filters.NextToken
	page.ResultCount = len(recipients)

	page.Data = result

	return page, nil
}

func (s *Storage) getRecipientsInDL(ctx context.Context, listName string, recipients []string) ([]DistListRecipient, error) {

	result := make([]DistListRecipient, 0)

	if len(recipients) == 0 {
		return result, nil
	}

	listFilter := expression.Equal(expression.Name("listName"), expression.Value(listName))
	userFilter := makeInFilter("userId", recipients)
	filterEx := listFilter.And(*userFilter)

	expr, err := expression.NewBuilder().WithFilter(filterEx).Build()

	if err != nil {
		return result, fmt.Errorf("failed to make expression - %w", err)
	}

	scanPaginator := dynamodb.NewScanPaginator(s.client, &dynamodb.ScanInput{
		TableName:                 aws.String(DistListRecipientsTable),
		ConsistentRead:            aws.Bool(true),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
	})

	for scanPaginator.HasMorePages() {
		resp, err := scanPaginator.NextPage(ctx)

		if err != nil {
			return []DistListRecipient{}, fmt.Errorf("failed to retrieve recipients page - %w", err)
		}

		var page []DistListRecipient
		err = attributevalue.UnmarshalListOfMaps(resp.Items, &page)

		if err != nil {
			return []DistListRecipient{}, fmt.Errorf("failed to unmarshall recipients page - %w", err)
		}

		result = append(result, page...)
	}

	return result, nil
}

func (s *Storage) updateRecipientCount(ctx context.Context, listName string, numRecipients int) (int, error) {

	summary := DistListSummary{Name: listName}
	key, err := summary.GetKey()

	if err != nil {
		return 0, fmt.Errorf("failed to build summary key")
	}

	update := expression.Add(expression.Name("numOfRecipients"), expression.Value(numRecipients))
	exp, err := expression.NewBuilder().WithUpdate(update).Build()

	if err != nil {
		return 0, fmt.Errorf("failed to build update expression - %w", err)
	}

	resp, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(DistListSummaryTable),
		Key:                       key,
		ExpressionAttributeNames:  exp.Names(),
		ExpressionAttributeValues: exp.Values(),
		UpdateExpression:          exp.Update(),
		ReturnValues:              types.ReturnValueUpdatedNew,
	})

	if err != nil {
		return 0, fmt.Errorf("failed to update summary count - %w", err)
	}

	var attrMap map[string]int

	err = attributevalue.UnmarshalMap(resp.Attributes, &attrMap)

	if err != nil {
		return 0, fmt.Errorf("failed to unmarshall summary count update - %w", err)
	}

	return attrMap["numOfRecipients"], nil
}

func (s *Storage) getNewRecipients(recipientsInDL []DistListRecipient, toCheck []string) []string {

	newRecipients := make([]string, 0)
	recipientSet := make(map[string]struct{})

	for _, r := range recipientsInDL {
		recipientSet[r.UserId] = struct{}{}
	}

	for _, r := range toCheck {
		if _, ok := recipientSet[r]; !ok {
			newRecipients = append(newRecipients, r)
		}
	}

	return newRecipients
}

func (s *Storage) AddRecipients(ctx context.Context, listName string, recipients []string) (*dto.DistributionListSummary, error) {

	exists, err := s.distListExists(ctx, listName)

	if err != nil {
		return nil, fmt.Errorf("failed to check if distribution list exists - %w", err)
	}

	if !exists {
		return nil, internal.DistributionListNotFound{Name: listName}
	}

	recipientsInDL, err := s.getRecipientsInDL(ctx, listName, recipients)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve recipients - %w", err)
	}

	newRecipients := s.getNewRecipients(recipientsInDL, recipients)
	toAdd := make([]DistListRecipient, 0, len(recipients))

	for _, r := range newRecipients {
		toAdd = append(toAdd, DistListRecipient{
			DistListName: listName,
			UserId:       r,
		})
	}

	if len(toAdd) == 0 {
		summary, err := s.getDistListSummary(ctx, listName)

		if err != nil {
			return nil, fmt.Errorf("failed to get dist list summary - %w", err)
		}

		s := dto.DistributionListSummary{
			Name:               listName,
			NumberOfRecipients: summary.NumRecipients,
		}

		return &s, nil
	}

	_, err = s.addRecipients(ctx, toAdd)

	if err != nil {
		return nil, err
	}

	count, err := s.updateRecipientCount(ctx, listName, len(newRecipients))

	if err != nil {
		return nil, fmt.Errorf("failed to update summary count - %w", err)
	}

	summary := dto.DistributionListSummary{
		Name:               listName,
		NumberOfRecipients: count,
	}

	return &summary, nil
}

func (s *Storage) DeleteRecipients(ctx context.Context, listName string, recipients []string) (*dto.DistributionListSummary, error) {

	exists, err := s.distListExists(ctx, listName)

	if err != nil {
		return nil, fmt.Errorf("failed to check if distribution list exists - %w", err)
	}

	if !exists {
		return nil, internal.DistributionListNotFound{Name: listName}
	}

	toRemove, err := s.getRecipientsInDL(ctx, listName, recipients)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve recipients - %w", err)
	}

	if len(toRemove) == 0 {
		summary, err := s.getDistListSummary(ctx, listName)

		if err != nil {
			return nil, fmt.Errorf("failed to get dist list summary - %w", err)
		}

		s := dto.DistributionListSummary{
			Name:               listName,
			NumberOfRecipients: summary.NumRecipients,
		}

		return &s, nil
	}

	_, err = s.deleteRecipients(ctx, toRemove)

	if err != nil {
		return nil, err
	}

	count, err := s.updateRecipientCount(ctx, listName, -len(toRemove))

	if err != nil {
		return nil, fmt.Errorf("failed to update summary count - %w", err)
	}

	summary := dto.DistributionListSummary{
		Name:               listName,
		NumberOfRecipients: count,
	}

	return &summary, nil
}

func (s *Storage) UpdateNotificationStatus(ctx context.Context, statusLog c.NotificationStatusLog) error {

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

func NewDynamoDBClient(c DynamoConfigurator) (client *dynamodb.Client, err error) {

	clientCfg := c.GetDynamoClientConfig()

	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		return client, fmt.Errorf("failed to load default config - %w", err)
	}

	client = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if clientCfg.BaseEndpoint != nil {
			o.BaseEndpoint = clientCfg.BaseEndpoint
		}

		if clientCfg.Region != nil {
			o.Region = *clientCfg.Region
		}
	})

	return
}

func NewDynamoDBStorage(a DynamoDBAPI) *Storage {
	return &Storage{client: a}
}
