package storage

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type distributionList struct {
	Name   string `dynamodbav:"name"`
	UserId string `dynamodbav:"userId"`
}

func (dl *distributionList) GetKey() (DynamoDBKey, error) {
	key := make(map[string]types.AttributeValue)

	name, err := attributevalue.Marshal(dl.Name)

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
