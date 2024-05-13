package storage

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type distListRecipient struct {
	DistListName string `dynamodbav:"listName"`
	UserId       string `dynamodbav:"userId"`
}

type distListSummary struct {
	Name          string `dynamodbav:"name"`
	NumRecipients int    `dynamodbav:"numOfRecipients"`
}

type distListSummaryKey struct {
	Name string `dynamodbav:"name" json:"name"`
}

func (dl *distListRecipient) GetKey() (DynamoDBKey, error) {
	key := make(map[string]types.AttributeValue)

	name, err := attributevalue.Marshal(dl.DistListName)

	if err != nil {
		return key, fmt.Errorf("failed to marshall dl name - %w", err)
	}

	userId, err := attributevalue.Marshal(dl.UserId)

	if err != nil {
		return key, fmt.Errorf("failed to marshall dl userId - %w", err)
	}

	key["name"] = name
	key["userId"] = userId

	return key, nil
}

func getSummaryKey(listName string) (DynamoDBKey, error) {
	key := make(map[string]types.AttributeValue)

	name, err := attributevalue.Marshal(listName)

	if err != nil {
		return key, fmt.Errorf("failed to marshall dl name - %w", err)
	}

	key["name"] = name

	return key, nil
}

func (dl *distListSummary) GetKey() (DynamoDBKey, error) {
	return getSummaryKey(dl.Name)
}

func (dl *distListSummaryKey) GetKey() (DynamoDBKey, error) {
	return getSummaryKey(dl.Name)
}
