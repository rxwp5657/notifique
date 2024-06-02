package test

import (
	"context"
	"fmt"
)

type storageType string

type Container interface {
	GetURI() string
}

const (
	DYNAMODB storageType = "DYNAMODB"
	POSTGRES storageType = "POSTGTES"
)

func setupContainer(t storageType) (Container, error) {
	switch t {
	case DYNAMODB:
		container, err := setupDynamoDB(context.TODO())
		return container, err
	case POSTGRES:
		container, err := setupPostgres(context.TODO())
		return container, err
	default:
		return nil, fmt.Errorf("invalid option - %s", t)
	}
}
