package registry_test

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ds "github.com/notifique/internal/registry/dynamodb"
	"github.com/notifique/internal/server/dto"
	"github.com/notifique/internal/testutils/containers"
)

type dynamoregistryTester struct {
	ds.Registry
	client    ds.DynamoDBAPI
	container *containers.DynamoContainer
}

func (t *dynamoregistryTester) ClearDB(ctx context.Context) error {

	tables := []string{
		ds.DistListRecipientsTable,
		ds.DistListSummaryTable,
		ds.NotificationsTable,
		ds.NotificationStatusLogTable,
		ds.UserConfigTable,
		ds.UserNotificationsTable,
		ds.NotificationsTemplateTable,
	}

	for _, table := range tables {
		_, err := t.client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
			TableName: aws.String(table),
		})

		if err != nil {
			return fmt.Errorf("failed to delete %s table - %w", table, err)
		}
	}

	err := t.container.CreateTables(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (t *dynamoregistryTester) GetDistributionList(
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

func (t *dynamoregistryTester) DistributionListExists(ctx context.Context,
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

func (t *dynamoregistryTester) getNotification(ctx context.Context,
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

func (t *dynamoregistryTester) GetNotification(ctx context.Context,
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

func (t *dynamoregistryTester) InsertUserNotifications(ctx context.Context,
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

func (t *dynamoregistryTester) GetNotificationTemplate(ctx context.Context, id string) (dto.NotificationTemplateReq, error) {

	templateReq := dto.NotificationTemplateReq{}

	nt := ds.NotificationTemplate{Id: id}

	key, err := nt.GetKey()

	if err != nil {
		return templateReq, err
	}

	resp, err := t.client.GetItem(ctx, &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(ds.NotificationsTemplateTable),
	})

	if err != nil {
		return templateReq, fmt.Errorf("failed to retrieve template - %w", err)
	}

	template := ds.NotificationTemplate{}
	err = attributevalue.UnmarshalMap(resp.Item, &template)

	if err != nil {
		return templateReq, fmt.Errorf("failed to unmarshall template - %w", err)
	}

	templateReq.Name = template.Name
	templateReq.Description = template.Description
	templateReq.TitleTemplate = template.TitleTemplate
	templateReq.ContentsTemplate = template.ContentsTemplate

	for _, v := range template.Variables {
		templateReq.Variables = append(templateReq.Variables, dto.TemplateVariable{
			Name:       v.Name,
			Type:       v.Type,
			Required:   v.Required,
			Validation: v.Validation,
		})
	}

	return templateReq, nil
}

func (t *dynamoregistryTester) TemplateExists(ctx context.Context, id string) (bool, error) {

	key, err := ds.NotificationTemplate{Id: id}.GetKey()

	if err != nil {
		return false, fmt.Errorf("failed to make the template key - %w", err)
	}

	projExp := expression.
		AddNames(expression.NamesList(expression.Name(ds.NotificationHashKey)))

	expr, err := expression.
		NewBuilder().
		WithProjection(projExp).
		Build()

	if err != nil {
		return false, fmt.Errorf("failed to create expression - %w", err)
	}

	resp, err := t.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:                aws.String(ds.NotificationsTemplateTable),
		Key:                      key,
		ProjectionExpression:     expr.Projection(),
		ExpressionAttributeNames: expr.Names(),
	})

	if err != nil {
		return false, fmt.Errorf("failed to query template id - %w", err)
	}

	return len(resp.Item) > 0, nil
}

func NewDynamoRegistryTester(ctx context.Context) (*dynamoregistryTester, closer, error) {

	container, containerCloser, err := containers.NewDynamoContainer(ctx)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create container - %w", err)
	}

	client, err := ds.NewDynamoDBClient(container)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dynamo client - %w", err)
	}

	s := ds.NewDynamoDBRegistry(client)

	closer := func() {
		containerCloser()
	}

	t := dynamoregistryTester{
		Registry:  *s,
		client:    client,
		container: container,
	}

	return &t, closer, nil
}
