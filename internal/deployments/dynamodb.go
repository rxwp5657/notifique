package deployments

import (
	"context"
	"errors"
	"fmt"
	"time"

	sdb "github.com/notifique/internal/storage/dynamodb"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func tableExists(client *dynamodb.Client, tableName string) (bool, error) {

	_, err := client.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})

	if err == nil {
		return true, nil
	}

	var notFoundEx *types.ResourceNotFoundException

	if errors.As(err, &notFoundEx) {
		return false, nil
	} else {
		return false, err
	}
}

func createTable(client *dynamodb.Client, tableName string, input *dynamodb.CreateTableInput) error {

	if exists, err := tableExists(client, tableName); exists || err != nil {
		return err
	}

	_, err := client.CreateTable(context.TODO(), input)

	if err != nil {
		return err
	}

	waiter := dynamodb.NewTableExistsWaiter(client)

	descTableInput := dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	err = waiter.Wait(context.TODO(), &descTableInput, 5*time.Minute)

	if err != nil {
		return err
	}

	return nil
}

func createNotificationTable(client *dynamodb.Client) error {

	tableName := sdb.NotificationsTable

	tableInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String(sdb.NotificationHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String(sdb.NotificationHashKey),
			KeyType:       types.KeyTypeHash,
		}},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	return createTable(client, tableName, &tableInput)
}

func createUserConfigTable(client *dynamodb.Client) error {

	tableName := sdb.UserConfigTable

	tableInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String(sdb.UserConfigHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String(sdb.UserConfigHashKey),
			KeyType:       types.KeyTypeHash,
		}},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	return createTable(client, tableName, &tableInput)
}

func createUserNotificationTable(client *dynamodb.Client) error {

	tableName := sdb.UserNotificationsTable

	tableInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String(sdb.UserNotificactionsHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String(sdb.UserNotificationsSortKey),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String(sdb.UserNotificationsCreatedAtIdxSortKey),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String(sdb.UserNotificactionsHashKey),
			KeyType:       types.KeyTypeHash,
		}, {
			AttributeName: aws.String(sdb.UserNotificationsSortKey),
			KeyType:       types.KeyTypeRange,
		}},
		LocalSecondaryIndexes: []types.LocalSecondaryIndex{{
			IndexName: aws.String(sdb.UserNotificationsCreatedAtIdx),
			KeySchema: []types.KeySchemaElement{{
				AttributeName: aws.String(sdb.UserNotificactionsHashKey),
				KeyType:       types.KeyTypeHash,
			}, {
				AttributeName: aws.String(sdb.UserNotificationsCreatedAtIdxSortKey),
				KeyType:       types.KeyTypeRange,
			}},
			Projection: &types.Projection{
				ProjectionType: types.ProjectionTypeAll,
			},
		}},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	return createTable(client, tableName, &tableInput)
}

func createDLRecipientsTable(client *dynamodb.Client) error {

	tableName := sdb.DistListRecipientsTable

	tableInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String(sdb.DistListRecipientHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String(sdb.DistListRecipientSortKey),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String(sdb.DistListRecipientHashKey),
			KeyType:       types.KeyTypeHash,
		}, {
			AttributeName: aws.String(sdb.DistListRecipientSortKey),
			KeyType:       types.KeyTypeRange,
		}},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	return createTable(client, tableName, &tableInput)
}

func createDLSummaryTable(client *dynamodb.Client) error {

	tableName := sdb.DistListSummaryTable

	tableInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String(sdb.DistListSummaryHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String(sdb.DistListSummaryHashKey),
			KeyType:       types.KeyTypeHash,
		}},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	return createTable(client, tableName, &tableInput)
}

func createNotificationStatusLogTable(client *dynamodb.Client) error {
	tableName := sdb.NotificationStatusLogTable

	tableInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String(sdb.NotificationStatusLogHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String(sdb.NotificationStatusLogKey),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String(sdb.NotificationStatusLogHashKey),
			KeyType:       types.KeyTypeHash,
		}, {
			AttributeName: aws.String(sdb.NotificationStatusLogKey),
			KeyType:       types.KeyTypeRange,
		}},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	return createTable(client, tableName, &tableInput)
}

func CreateTables(client *dynamodb.Client) error {

	if err := createNotificationTable(client); err != nil {
		return fmt.Errorf("notifications table - %w", err)
	}

	if err := createUserConfigTable(client); err != nil {
		return fmt.Errorf("user config table - %w", err)
	}

	if err := createUserNotificationTable(client); err != nil {
		return fmt.Errorf("user notifications table - %w", err)
	}

	if err := createDLRecipientsTable(client); err != nil {
		return fmt.Errorf("distribution lists table - %w", err)
	}

	if err := createDLSummaryTable(client); err != nil {
		return fmt.Errorf("distribution lists table - %w", err)
	}

	if err := createNotificationStatusLogTable(client); err != nil {
		return fmt.Errorf("notification status log table - %w", err)
	}

	return nil
}
