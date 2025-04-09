package dynamoregistry

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/notifique/service/internal"
	"github.com/notifique/service/internal/registry"
	sdto "github.com/notifique/shared/dto"
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

// Registry represents a registry system that uses DynamoDB as the backend.
// It contains a client that implements the DynamoDBAPI interface, which
// provides methods to interact with DynamoDB.
type Registry struct {
	client DynamoDBAPI
}

// DynamoPrimaryKey is an interface that defines a method for obtaining a DynamoDB key.
// Implementers of this interface should provide their own logic for generating and returning
// a DynamoKey, along with any potential errors that might occur during the process.
type DynamoPrimaryKey interface {
	GetKey() (DynamoKey, error)
}

// DynamoPageParams represents the parameters for paginating through DynamoDB results.
// Limit specifies the maximum number of items to evaluate (not necessarily the number of matching items).
// ExclusiveStartKey is the primary key of the first item that this operation will evaluate.
// Use the value that was returned for LastEvaluatedKey in the previous operation.
type DynamoPageParams struct {
	Limit             *int32
	ExclusiveStartKey map[string]types.AttributeValue
}

type DynamoKey map[string]types.AttributeValue
type DynamoObj map[string]types.AttributeValue
type BatchWriteRequest map[string][]types.WriteRequest

func marshalNextToken[T any](key *T, lastEvaluatedKey DynamoKey) (string, error) {
	err := attributevalue.UnmarshalMap(lastEvaluatedKey, &key)

	if err != nil {
		return "", fmt.Errorf("failed to unmarshall last evaluated key - %w", err)
	}

	return registry.MarshalKey(key)
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

func makePageFilters[T DynamoPrimaryKey](key T, filters sdto.PageFilter) (DynamoPageParams, error) {

	params := DynamoPageParams{}

	params.Limit = aws.Int32(internal.PageSize)

	if filters.MaxResults != nil {
		limit := int32(*filters.MaxResults)
		params.Limit = &limit
	}

	if filters.NextToken != nil {
		err := registry.UnmarshalKey(*filters.NextToken, &key)

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

func NewDynamoDBRegistry(a DynamoDBAPI) *Registry {
	return &Registry{client: a}
}
