package storage_test

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/notifique/dto"
	ds "github.com/notifique/internal/storage/dynamodb"
	"github.com/notifique/test/containers"
)

type DynamoStorageTester struct {
	ds.DynamoDBStorage
	client    ds.DynamoDBAPI
	container *containers.DynamoContainer
}

func (t *DynamoStorageTester) ClearDB(ctx context.Context) error {

	_, err := t.client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(ds.DistListRecipientsTable),
	})

	if err != nil {
		return fmt.Errorf("failed to delete distribution list recipients table - %w", err)
	}

	_, err = t.client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(ds.DistListSummaryTable),
	})

	if err != nil {
		return fmt.Errorf("failed to delete distribution list summary table - %w", err)
	}

	_, err = t.client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(ds.NotificationsTable),
	})

	if err != nil {
		return fmt.Errorf("failed to delete notifications table - %w", err)
	}

	_, err = t.client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(ds.NotificationStatusLogTable),
	})

	if err != nil {
		return fmt.Errorf("failed to delete notification status log table - %w", err)
	}

	_, err = t.client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(ds.UserConfigTable),
	})

	if err != nil {
		return fmt.Errorf("failed to delete user config table - %w", err)
	}

	_, err = t.client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(ds.UserNotificationsTable),
	})

	if err != nil {
		return fmt.Errorf("failed to delete user notification table - %w", err)
	}

	err = t.container.CreateTables(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (t *DynamoStorageTester) GetDistributionList(
	ctx context.Context,
	dlName string) (dto.DistributionList, error) {

	d := dto.DistributionList{
		Name: dlName,
	}

	filter := dto.PageFilter{}
	recipients := []string{}

	for {
		page, err := t.GetRecipients(ctx, dlName, filter)

		if err != nil {
			return d, fmt.Errorf("failed to get recipients - %w", err)
		}

		recipients = append(recipients, page.Data...)

		if page.NextToken != nil {
			filter.NextToken = page.NextToken
			continue
		} else {
			break
		}
	}

	d.Recipients = recipients

	return d, nil
}

func (t *DynamoStorageTester) DistributionListExists(ctx context.Context,
	dlName string) (bool, error) {

	dlKey := ds.DistListSummaryKey{Name: dlName}

	key, err := dlKey.GetKey()

	if err != nil {
		return false, fmt.Errorf("failed to make dl key - %w", err)
	}

	resp, err := t.client.GetItem(ctx, &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(ds.DistListSummaryTable),
	})

	if err != nil {
		return false, fmt.Errorf("failed to retrieve summary - %w", err)
	}

	summary := ds.DistListSummary{}

	err = attributevalue.UnmarshalMap(resp.Item, &summary)

	if err != nil {
		return false, fmt.Errorf("failed to unmarshall summary - %w", err)
	}

	return summary.Name != "", nil
}

func (t *DynamoStorageTester) getNotification(ctx context.Context,
	notificationId string) (ds.Notification, error) {

	n := ds.Notification{Id: notificationId}
	k, err := n.GetKey()

	if err != nil {
		return n, fmt.Errorf("failed to make notification key - %w", err)
	}

	resp, err := t.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(ds.NotificationsTable),
		Key:       k,
	})

	if err != nil {
		return n, fmt.Errorf("failed to retrieve notification - %w", err)
	}

	err = attributevalue.UnmarshalMap(resp.Item, &n)

	if err != nil {
		return n, fmt.Errorf("failed to unmarshall notification - %w", err)
	}

	return n, nil
}

func (t *DynamoStorageTester) GetNotification(ctx context.Context,
	notificationId string) (dto.NotificationReq, error) {

	n, err := t.getNotification(ctx, notificationId)

	if err != nil {
		return dto.NotificationReq{}, err
	}

	req := dto.NotificationReq{
		Title:            n.Title,
		Contents:         n.Contents,
		Image:            n.Image,
		Topic:            n.Topic,
		Priority:         n.Priority,
		DistributionList: n.DistributionList,
		Recipients:       n.Recipients,
		Channels:         n.Channels,
	}

	return req, nil
}

func (t *DynamoStorageTester) GetNotificationStatus(ctx context.Context,
	notificationId string) (string, error) {

	n, err := t.getNotification(ctx, notificationId)

	if err != nil {
		return "", err
	}

	return n.Status, nil
}

func (t *DynamoStorageTester) InsertUserNotifications(ctx context.Context,
	userId string, un []dto.UserNotification) error {

	notifications := make([]ds.UserNotification, 0, len(un))

	for _, n := range un {
		userNotification := ds.UserNotification{
			Id:        n.Id,
			UserId:    userId,
			Title:     n.Title,
			Contents:  n.Contents,
			CreatedAt: n.CreatedAt,
			Image:     n.Image,
			ReadAt:    n.ReadAt,
			Topic:     n.Topic,
		}

		notifications = append(notifications, userNotification)
	}

	requestItems, err := ds.MakeBatchWriteRequest(ds.UserNotificationsTable, notifications)

	if err != nil {
		return fmt.Errorf("failed to make batch request - %w", err)
	}

	_, err = t.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: requestItems,
	})

	if err != nil {
		return fmt.Errorf("failed to batch write item - %w", err)
	}

	return err
}

func NewDynamoStorageTester(ctx context.Context) (*DynamoStorageTester, closer, error) {

	container, containerCloser, err := containers.NewDynamoContainer(ctx)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create container - %w", err)
	}

	client, err := ds.NewDynamoDBClient(container)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dynamo client - %w", err)
	}

	s := ds.NewDynamoDBStorage(client)

	closer := func() {
		containerCloser()
	}

	t := DynamoStorageTester{
		DynamoDBStorage: *s,
		client:          client,
		container:       container,
	}

	return &t, closer, nil
}
