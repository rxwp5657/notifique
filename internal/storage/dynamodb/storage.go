package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/notifique/dto"
	"github.com/notifique/internal"
)

type DynamoDBStorage struct {
	client *dynamodb.Client
}

type DynamodbPrimaryKey interface {
	GetKey() (DynamoDBKey, error)
}

type DynamoDBPageParams struct {
	Limit             *int32
	ExclusiveStartKey map[string]types.AttributeValue
}

type DynamoDBKey map[string]types.AttributeValue
type DynamoObj map[string]types.AttributeValue

func marshallNextToken[T any](key *T, lastEvaluatedKey DynamoDBKey) (string, error) {
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

func (s *DynamoDBStorage) getUserConfig(ctx context.Context, userId string) (*userConfig, error) {

	tmpConfig := userConfig{UserId: userId}
	key, err := tmpConfig.GetKey()

	if err != nil {
		return nil, err
	}

	resp, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(USER_CONFIG_TABLE),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get the user config - %w", err)
	}

	if len(resp.Item) == 0 {
		return nil, nil
	}

	config := userConfig{}

	err = attributevalue.UnmarshalMap(resp.Item, &config)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall the user config - %w", err)
	}

	return &config, nil
}

func (s *DynamoDBStorage) createUserConfig(ctx context.Context, userId string) (*userConfig, error) {
	config := userConfig{
		UserId:      userId,
		EmailConfig: channelConfig{OptIn: true, SnoozeUntil: nil},
		SMSConfig:   channelConfig{OptIn: true, SnoozeUntil: nil},
		InAppConfig: channelConfig{OptIn: true, SnoozeUntil: nil},
	}

	item, err := attributevalue.MarshalMap(config)

	if err != nil {
		return nil, fmt.Errorf("failed to marshall the user config - %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(USER_CONFIG_TABLE),
		Item:      item,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to store notification - %w", err)
	}

	return &config, nil
}

func (s *DynamoDBStorage) SaveNotification(ctx context.Context, createdBy string, notificationReq dto.NotificationReq) (string, error) {

	id := uuid.NewString()

	notification := notification{
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
		Logs:             []notificationLog{},
	}

	item, err := attributevalue.MarshalMap(notification)

	if err != nil {
		return "", fmt.Errorf("failed to marshall notification - %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(NOTIFICATIONS_TABLE),
		Item:      item,
	})

	if err != nil {
		return "", fmt.Errorf("failed to store notification - %w", err)
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

func makePageFilters[T DynamodbPrimaryKey](key T, filters dto.PageFilter) (DynamoDBPageParams, error) {

	params := DynamoDBPageParams{}

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

func (s *DynamoDBStorage) GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) (dto.Page[dto.UserNotification], error) {

	page := dto.Page[dto.UserNotification]{}

	keyExp := expression.Key(USER_NOTIFICATIONS_HASH_KEY).Equal(expression.Value(filters.UserId))
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
		TableName:                 aws.String(USER_NOTIFICATIONS_TABLE),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		FilterExpression:          expr.Filter(),
		ScanIndexForward:          aws.Bool(false),
		Limit:                     pageParams.Limit,
		ExclusiveStartKey:         pageParams.ExclusiveStartKey,
		IndexName:                 aws.String(USER_NOTIFICATIONS_CREATEDAT_IDX),
	}

	response, err := s.client.Query(ctx, &queryInput)

	if err != nil {
		return page, fmt.Errorf("failed to get notifications - %w", err)
	}

	var notifications []userNotification
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

func (s *DynamoDBStorage) GetUserConfig(ctx context.Context, userId string) (dto.UserConfig, error) {

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

func (s *DynamoDBStorage) SetReadStatus(ctx context.Context, userId, notificationId string) error {

	reatAt := time.Now().Format(time.RFC3339Nano)
	update := expression.Set(expression.Name("readAt"), expression.Value(reatAt))
	condEx := expression.AttributeExists(expression.Name(USER_NOTIFICATIONS_HASH_KEY))
	expr, err := expression.NewBuilder().WithUpdate(update).WithCondition(condEx).Build()

	if err != nil {
		return fmt.Errorf("failed to make update query - %w", err)
	}

	notification := userNotification{UserId: userId, Id: notificationId}
	key, err := notification.GetKey()

	if err != nil {
		return err
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                           aws.String(USER_NOTIFICATIONS_TABLE),
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

func (s *DynamoDBStorage) UpdateUserConfig(ctx context.Context, userId string, config dto.UserConfig) error {

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

	emailFmt := makeKeyFormatter(USER_CONFIG_EMAIL_KEY)
	smsFmt := makeKeyFormatter(USER_CONFIG_SMS_KEY)
	inAppFmt := makeKeyFormatter(USER_CONFIG_INAPP_KEY)

	update := expression.Set(emailFmt(USER_CONFIG_OPT_IN), expression.Value(config.EmailConfig.OptIn))
	update.Set(emailFmt(USER_CONFIG_SNOOZE_UNTIL), expression.Value(config.EmailConfig.SnoozeUntil))
	update.Set(smsFmt(USER_CONFIG_OPT_IN), expression.Value(config.SMSConfig.OptIn))
	update.Set(smsFmt(USER_CONFIG_SNOOZE_UNTIL), expression.Value(config.SMSConfig.SnoozeUntil))
	update.Set(inAppFmt(USER_CONFIG_OPT_IN), expression.Value(config.InAppConfig.OptIn))
	update.Set(inAppFmt(USER_CONFIG_SNOOZE_UNTIL), expression.Value(config.InAppConfig.SnoozeUntil))

	expr, err := expression.NewBuilder().WithUpdate(update).Build()

	if err != nil {
		return fmt.Errorf("failed to make update query - %w", err)
	}

	tmpConfig := userConfig{UserId: userId}
	key, err := tmpConfig.GetKey()

	if err != nil {
		return err
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(USER_CONFIG_TABLE),
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

func (s *DynamoDBStorage) addRecipients(ctx context.Context, recipients []distListRecipient) (int, error) {

	writeReq := make([]types.WriteRequest, 0, len(recipients))

	for _, r := range recipients {
		item, err := attributevalue.MarshalMap(r)

		if err != nil {
			return 0, fmt.Errorf("failed to marshal dl distListRecipient - %w", err)
		}

		writeReq = append(writeReq, types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: item,
			},
		})
	}

	requestItems := map[string][]types.WriteRequest{
		DIST_LIST_RECIPIENTS_TABLE: writeReq,
	}

	_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: requestItems,
	})

	if err != nil {
		return 0, fmt.Errorf("failed to add recipients to dl - %w", err)
	}

	return len(recipients), nil
}

func (s *DynamoDBStorage) queryDistListSummary(ctx context.Context, listName string) (*map[string]types.AttributeValue, error) {
	summary := distListSummary{Name: listName}
	key, err := summary.GetKey()

	if err != nil {
		return nil, fmt.Errorf("failed to get distribution list key - %w", err)
	}

	resp, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(DIST_LIST_SUMMARY_TABLE),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get distribution list - %w", err)
	}

	return &resp.Item, nil
}

func (s *DynamoDBStorage) getDistListSummary(ctx context.Context, listName string) (*distListSummary, error) {
	resp, err := s.queryDistListSummary(ctx, listName)

	if err != nil {
		return nil, err
	}

	summary := distListSummary{}

	err = attributevalue.UnmarshalMap(*resp, &summary)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall distribution list - %w", err)
	}

	return &summary, nil
}

func (s *DynamoDBStorage) distListExists(ctx context.Context, listName string) (bool, error) {

	resp, err := s.queryDistListSummary(ctx, listName)

	if err != nil {
		return false, err
	}

	return len(*resp) != 0, nil
}

func (s *DynamoDBStorage) deleteSummary(ctx context.Context, listName string) error {
	summary := distListSummary{Name: listName}
	key, err := summary.GetKey()

	if err != nil {
		return err
	}

	_, err = s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(DIST_LIST_SUMMARY_TABLE),
		Key:       key,
	})

	return err
}

func (s *DynamoDBStorage) CreateDistributionList(ctx context.Context, dlReq dto.DistributionList) error {

	exists, err := s.distListExists(ctx, dlReq.Name)

	if err != nil {
		return fmt.Errorf("failed to check for list existence - %w", err)
	}

	if exists {
		return internal.DistributionListAlreadyExists{Name: dlReq.Name}
	}

	summary := distListSummary{
		Name:          dlReq.Name,
		NumRecipients: len(dlReq.Recipients),
	}

	marshalled, err := attributevalue.MarshalMap(summary)

	if err != nil {
		return fmt.Errorf("failed to marshall distribution list summary - %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(DIST_LIST_SUMMARY_TABLE),
		Item:      marshalled,
	})

	if err != nil {
		return fmt.Errorf("failed to create summary - %w", err)
	}

	recipients := make([]distListRecipient, 0, len(dlReq.Recipients))

	for _, r := range dlReq.Recipients {
		recipients = append(recipients, distListRecipient{
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

func (s *DynamoDBStorage) GetDistributionLists(ctx context.Context, filters dto.PageFilter) (dto.Page[dto.DistributionListSummary], error) {

	page := dto.Page[dto.DistributionListSummary]{}

	pageParams, err := makePageFilters(&distListSummaryKey{}, filters)

	if err != nil {
		return page, fmt.Errorf("failed to make page params - %w", err)
	}

	scanInput := dynamodb.ScanInput{
		TableName:         aws.String(DIST_LIST_SUMMARY_TABLE),
		Limit:             pageParams.Limit,
		ExclusiveStartKey: pageParams.ExclusiveStartKey,
	}

	response, err := s.client.Scan(ctx, &scanInput)

	if err != nil {
		return page, fmt.Errorf("failed to get the summaries - %w", err)
	}

	var summaries []distListSummary
	err = attributevalue.UnmarshalListOfMaps(response.Items, &summaries)

	if err != nil {
		return page, fmt.Errorf("failed to unmarshall the summaries - %w", err)
	}

	var nextToken *string = nil

	if len(response.LastEvaluatedKey) != 0 {
		key := distListSummaryKey{}
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

func (s *DynamoDBStorage) deleteRecipients(ctx context.Context, recipients []distListRecipient) (int, error) {

	deleteReq := make([]types.WriteRequest, 0, len(recipients))

	for _, distListRecipient := range recipients {
		key, _ := distListRecipient.GetKey()
		deleteReq = append(deleteReq, types.WriteRequest{
			DeleteRequest: &types.DeleteRequest{
				Key: key,
			},
		})
	}

	requestItems := map[string][]types.WriteRequest{
		DIST_LIST_RECIPIENTS_TABLE: deleteReq,
	}

	_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: requestItems,
	})

	if err != nil {
		return 0, fmt.Errorf("failed to delete recipients of dl - %w", err)
	}

	return len(recipients), nil
}

func (s *DynamoDBStorage) DeleteDistributionList(ctx context.Context, listName string) error {

	keyEx := expression.Key(DIST_LIST_RECIPIENT_HASH_KEY).Equal(expression.Value(listName))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()

	if err != nil {
		return fmt.Errorf("failed to create expression - %w", err)
	}

	queryPaginator := dynamodb.NewQueryPaginator(s.client, &dynamodb.QueryInput{
		TableName:                 aws.String(DIST_LIST_RECIPIENTS_TABLE),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
	})

	for queryPaginator.HasMorePages() {
		resp, err := queryPaginator.NextPage(ctx)

		if err != nil {
			return fmt.Errorf("failed to retrieve user page - %w", err)
		}

		var recipients []distListRecipient
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

func (s *DynamoDBStorage) GetRecipients(ctx context.Context, distlistName string, filters dto.PageFilter) (dto.Page[string], error) {

	page := dto.Page[string]{}

	exists, err := s.distListExists(ctx, distlistName)

	if err != nil {
		return page, fmt.Errorf("failed to check if distribution list exists - %w", err)
	}

	if !exists {
		return page, internal.DistributionListNotFound{Name: distlistName}
	}

	keyExp := expression.Key(DIST_LIST_RECIPIENT_HASH_KEY).Equal(expression.Value(distlistName))
	projExp := expression.NamesList(expression.Name(DIST_LIST_RECIPIENT_SORT_KEY))

	builder := expression.NewBuilder()
	expr, err := builder.WithKeyCondition(keyExp).WithProjection(projExp).Build()

	if err != nil {
		return page, fmt.Errorf("failed to build query - %w", err)
	}

	pageParams, err := makePageFilters(&distListRecipient{}, filters)

	if err != nil {
		return page, fmt.Errorf("failed to make page params - %w", err)
	}

	queryInput := dynamodb.QueryInput{
		TableName:                 aws.String(DIST_LIST_RECIPIENTS_TABLE),
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

	var recipients []distListRecipient
	err = attributevalue.UnmarshalListOfMaps(response.Items, &recipients)

	if err != nil {
		return page, fmt.Errorf("failed to unmarshall recipients - %w", err)
	}

	var nextToken *string = nil

	if len(response.LastEvaluatedKey) != 0 {
		key := distListRecipient{}
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

func (s *DynamoDBStorage) getRecipientsInDL(ctx context.Context, listName string, recipients []string) ([]distListRecipient, error) {

	result := make([]distListRecipient, 0)

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
		TableName:                 aws.String(DIST_LIST_RECIPIENTS_TABLE),
		ConsistentRead:            aws.Bool(true),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
	})

	for scanPaginator.HasMorePages() {
		resp, err := scanPaginator.NextPage(ctx)

		if err != nil {
			return []distListRecipient{}, fmt.Errorf("failed to retrieve recipients page - %w", err)
		}

		var page []distListRecipient
		err = attributevalue.UnmarshalListOfMaps(resp.Items, &page)

		if err != nil {
			return []distListRecipient{}, fmt.Errorf("failed to unmarshall recipients page - %w", err)
		}

		result = append(result, page...)
	}

	return result, nil
}

func (s *DynamoDBStorage) updateRecipientCount(ctx context.Context, listName string, numRecipients int) (int, error) {

	summary := distListSummary{Name: listName}
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
		TableName:                 aws.String(DIST_LIST_SUMMARY_TABLE),
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

func (s *DynamoDBStorage) getNewRecipients(recipientsInDL []distListRecipient, toCheck []string) []string {

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

func (s *DynamoDBStorage) AddRecipients(ctx context.Context, listName string, recipients []string) (*dto.DistributionListSummary, error) {

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
	toAdd := make([]distListRecipient, 0, len(recipients))

	for _, r := range newRecipients {
		toAdd = append(toAdd, distListRecipient{
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

func (s *DynamoDBStorage) DeleteRecipients(ctx context.Context, listName string, recipients []string) (*dto.DistributionListSummary, error) {

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

func (s *DynamoDBStorage) CreateUserNotification(ctx context.Context, userId string, un dto.UserNotification) error {

	notification := userNotification{
		Id:        un.Id,
		UserId:    userId,
		Title:     un.Title,
		Contents:  un.Contents,
		CreatedAt: un.CreatedAt,
		Image:     un.Image,
		ReadAt:    un.ReadAt,
		Topic:     un.Topic,
	}

	item, err := attributevalue.MarshalMap(notification)

	if err != nil {
		return fmt.Errorf("failed to marshall user notification - %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(USER_NOTIFICATIONS_TABLE),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to store user notification - %w", err)
	}

	return nil
}

func (s *DynamoDBStorage) DeleteUserNotification(ctx context.Context, userId string, un dto.UserNotification) error {

	notification := userNotification{
		UserId: userId,
		ReadAt: un.ReadAt,
	}

	key, err := notification.GetKey()

	if err != nil {
		return err
	}

	_, err = s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(USER_NOTIFICATIONS_TABLE),
		Key:       key,
	})

	return err
}

func MakeDynamoDBStorage(baseEndpoint *string) DynamoDBStorage {

	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		log.Fatal(err)
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = baseEndpoint
	})

	return DynamoDBStorage{client: client}
}
