package deployments

import (
	"context"
	"errors"
	"fmt"
	"time"

	r "github.com/notifique/internal/registry/dynamodb"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func tableExists(client dynamodb.Client, tableName string) (bool, error) {

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

func createTable(client dynamodb.Client, tableName string, input dynamodb.CreateTableInput) error {

	if exists, err := tableExists(client, tableName); exists || err != nil {
		return err
	}

	_, err := client.CreateTable(context.TODO(), &input)

	if err != nil {
		return err
	}

	waiter := dynamodb.NewTableExistsWaiter(&client)

	descTableInput := dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	err = waiter.Wait(context.TODO(), &descTableInput, 5*time.Minute)

	if err != nil {
		return fmt.Errorf("failed to create table %s - %w", tableName, err)
	}

	return nil
}

func createNotificationTable(client dynamodb.Client) error {

	tableName := r.NotificationsTable

	tableInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String(r.NotificationHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String(r.NotificationHashKey),
			KeyType:       types.KeyTypeHash,
		}},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	return createTable(client, tableName, tableInput)
}

func createUserConfigTable(client dynamodb.Client) error {

	tableName := r.UserConfigTable

	tableInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String(r.UserConfigHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String(r.UserConfigHashKey),
			KeyType:       types.KeyTypeHash,
		}},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	return createTable(client, tableName, tableInput)
}

func createUserNotificationTable(client dynamodb.Client) error {

	tableName := r.UserNotificationsTable

	tableInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String(r.UserNotificactionsHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String(r.UserNotificationsSortKey),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String(r.UserNotificactionsHashKey),
			KeyType:       types.KeyTypeHash,
		}, {
			AttributeName: aws.String(r.UserNotificationsSortKey),
			KeyType:       types.KeyTypeRange,
		}},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	return createTable(client, tableName, tableInput)
}

func createDLRecipientsTable(client dynamodb.Client) error {

	tableName := r.DistListRecipientsTable

	tableInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String(r.DistListRecipientHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String(r.DistListRecipientSortKey),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String(r.DistListRecipientHashKey),
			KeyType:       types.KeyTypeHash,
		}, {
			AttributeName: aws.String(r.DistListRecipientSortKey),
			KeyType:       types.KeyTypeRange,
		}},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	return createTable(client, tableName, tableInput)
}

func createDLSummaryTable(client dynamodb.Client) error {

	tableName := r.DistListSummaryTable

	tableInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String(r.DistListSummaryHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String(r.DistListSummaryHashKey),
			KeyType:       types.KeyTypeHash,
		}},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	return createTable(client, tableName, tableInput)
}

func createNotificationStatusLogTable(client dynamodb.Client) error {
	tableName := r.NotificationStatusLogTable

	tableInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String(r.NotificationStatusLogHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String(r.NotificationStatusLogSortKey),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String(r.NotificationStatusLogHashKey),
			KeyType:       types.KeyTypeHash,
		}, {
			AttributeName: aws.String(r.NotificationStatusLogSortKey),
			KeyType:       types.KeyTypeRange,
		}},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	return createTable(client, tableName, tableInput)
}

func createNotificationTemplateTable(client dynamodb.Client) error {

	tableName := r.NotificationsTemplateTable

	tableInput := dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String(r.NotificationTemplateHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String(r.NotificationsTemplateNameGSIHashKey),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String(r.NotificationsTemplateNameGSISortKey),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String(r.NotificationTemplateHashKey),
			KeyType:       types.KeyTypeHash,
		}},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{{
			IndexName: aws.String(r.NotificationsTemplateNameGSI),
			KeySchema: []types.KeySchemaElement{{
				AttributeName: aws.String(r.NotificationsTemplateNameGSIHashKey),
				KeyType:       types.KeyTypeHash,
			}, {
				AttributeName: aws.String(r.NotificationsTemplateNameGSISortKey),
				KeyType:       types.KeyTypeRange,
			}},
			Projection: &types.Projection{
				NonKeyAttributes: []string{
					"description",
				},
				ProjectionType: types.ProjectionTypeInclude,
			},
			ProvisionedThroughput: &types.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(10),
				WriteCapacityUnits: aws.Int64(10),
			},
		}},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	return createTable(client, tableName, tableInput)
}

func CreateTables(client *dynamodb.Client) error {

	if client == nil {
		return fmt.Errorf("client is nil")
	}

	tables := []func(dynamodb.Client) error{
		createNotificationTable,
		createUserConfigTable,
		createUserNotificationTable,
		createDLRecipientsTable,
		createDLSummaryTable,
		createNotificationStatusLogTable,
		createNotificationTemplateTable,
	}

	for _, fn := range tables {
		if err := fn(*client); err != nil {
			return err
		}
	}

	return nil
}
