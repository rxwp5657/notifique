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

func setupContainer(t storageType) (Container, func() error, error) {
	switch t {
	case DYNAMODB:
		container, f, err := setupDynamoDB(context.TODO())
		return container, f, err
	case POSTGRES:
		container, f, err := setupPostgres(context.TODO())
		return container, f, err
	default:
		return nil, nil, fmt.Errorf("invalid option - %s", t)
	}
}
