package dynamoregistry

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/notifique/internal/server"
	"github.com/notifique/internal/server/dto"
)

const (
	DistListRecipientsTable  = "DistributionListRecipients"
	DistListSummaryTable     = "DistributionListSummary"
	DistListRecipientHashKey = "listName"
	DistListRecipientSortKey = "userId"
	DistListSummaryHashKey   = "name"
)

type DistListRecipient struct {
	DistListName string `dynamodbav:"listName"`
	UserId       string `dynamodbav:"userId"`
}

type DistListSummary struct {
	Name          string `dynamodbav:"name"`
	NumRecipients int    `dynamodbav:"numOfRecipients"`
}

type DistListSummaryKey struct {
	Name string `dynamodbav:"name" json:"name"`
}

func (dl *DistListRecipient) GetKey() (DynamoKey, error) {
	key := make(map[string]types.AttributeValue)

	name, err := attributevalue.Marshal(dl.DistListName)

	if err != nil {
		return key, fmt.Errorf("failed to marshall dl name - %w", err)
	}

	userId, err := attributevalue.Marshal(dl.UserId)

	if err != nil {
		return key, fmt.Errorf("failed to marshall dl userId - %w", err)
	}

	key[DistListRecipientHashKey] = name
	key[DistListRecipientSortKey] = userId

	return key, nil
}

func getSummaryKey(listName string) (DynamoKey, error) {
	key := make(map[string]types.AttributeValue)

	name, err := attributevalue.Marshal(listName)

	if err != nil {
		return key, fmt.Errorf("failed to marshall dl name - %w", err)
	}

	key[DistListSummaryHashKey] = name

	return key, nil
}

func (dl *DistListSummary) GetKey() (DynamoKey, error) {
	return getSummaryKey(dl.Name)
}

func (dl *DistListSummaryKey) GetKey() (DynamoKey, error) {
	return getSummaryKey(dl.Name)
}

func (s *Registry) addRecipients(ctx context.Context, recipients []DistListRecipient) (int, error) {

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

func (s *Registry) queryDistListSummary(ctx context.Context, listName string) (*map[string]types.AttributeValue, error) {
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

func (s *Registry) getDistListSummary(ctx context.Context, listName string) (*DistListSummary, error) {
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

func (s *Registry) distListExists(ctx context.Context, listName string) (bool, error) {

	resp, err := s.queryDistListSummary(ctx, listName)

	if err != nil {
		return false, err
	}

	return len(*resp) != 0, nil
}

func (s *Registry) CreateDistributionList(ctx context.Context, dlReq dto.DistributionList) error {

	exists, err := s.distListExists(ctx, dlReq.Name)

	if err != nil {
		return fmt.Errorf("failed to check for list existence - %w", err)
	}

	if exists {
		return server.DistributionListAlreadyExists{Name: dlReq.Name}
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

func (s *Registry) GetDistributionLists(ctx context.Context, filters dto.PageFilter) (dto.Page[dto.DistributionListSummary], error) {

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
		encoded, err := marshalNextToken(&key, response.LastEvaluatedKey)

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

func (s *Registry) deleteSummary(ctx context.Context, listName string) error {
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

func (s *Registry) deleteRecipients(ctx context.Context, recipients []DistListRecipient) (int, error) {

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

func (s *Registry) DeleteDistributionList(ctx context.Context, listName string) error {

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

func (s *Registry) GetRecipients(ctx context.Context, distlistName string, filters dto.PageFilter) (dto.Page[string], error) {

	page := dto.Page[string]{}

	exists, err := s.distListExists(ctx, distlistName)

	if err != nil {
		return page, fmt.Errorf("failed to check if distribution list exists - %w", err)
	}

	if !exists {
		return page, server.DistributionListNotFound{Name: distlistName}
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
		encoded, err := marshalNextToken(&key, response.LastEvaluatedKey)

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

func (s *Registry) getRecipientsInDL(ctx context.Context, listName string, recipients []string) ([]DistListRecipient, error) {

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

func (s *Registry) updateRecipientCount(ctx context.Context, listName string, numRecipients int) (int, error) {

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

func (s *Registry) getNewRecipients(recipientsInDL []DistListRecipient, toCheck []string) []string {

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

func (s *Registry) AddRecipients(ctx context.Context, listName string, recipients []string) (*dto.DistributionListSummary, error) {

	exists, err := s.distListExists(ctx, listName)

	if err != nil {
		return nil, fmt.Errorf("failed to check if distribution list exists - %w", err)
	}

	if !exists {
		return nil, server.DistributionListNotFound{Name: listName}
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

func (s *Registry) DeleteRecipients(ctx context.Context, listName string, recipients []string) (*dto.DistributionListSummary, error) {

	exists, err := s.distListExists(ctx, listName)

	if err != nil {
		return nil, fmt.Errorf("failed to check if distribution list exists - %w", err)
	}

	if !exists {
		return nil, server.DistributionListNotFound{Name: listName}
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
