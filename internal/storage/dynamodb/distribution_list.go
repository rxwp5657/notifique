package storage

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	DistListRecipientsTable  = "DistributionListRecipients"
	DistListSummaryTable     = "DistributionListSummary"
	DistListRecipientHashKey = "listName"
	DistListRecipientSortKey = "userId"
	DistListSummaryHashKey   = "name"
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

func (dl *distListRecipient) GetKey() (DynamoKey, error) {
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

func (dl *distListSummary) GetKey() (DynamoKey, error) {
	return getSummaryKey(dl.Name)
}

func (dl *distListSummaryKey) GetKey() (DynamoKey, error) {
	return getSummaryKey(dl.Name)
}
