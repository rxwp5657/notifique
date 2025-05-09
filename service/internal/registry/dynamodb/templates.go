package dynamoregistry

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"

	"github.com/notifique/service/internal"
	"github.com/notifique/service/internal/dto"
	"github.com/notifique/service/internal/registry"
	sdto "github.com/notifique/shared/dto"
)

const (
	NotificationsTemplateTable          = "NotificationsTemplate"
	NotificationTemplateHashKey         = "id"
	NotificationsTemplateNameGSI        = "TemplateNameIdx"
	NotificationsTemplateNameGSIHashKey = "hashKey"
	NotificationsTemplateNameGSISortKey = "name"
	// There's no clear partition key for templates, making it impossible
	// to use query. Also, SCAN is not an option either because it affects
	// user experience as results will be returned randomly. We are
	// sacrificing scalability and performance by using a single partition
	// key.
	NotificationsTemplateSyntheticKey = "TEMPLATE"
)

type TemplateVariable struct {
	Name       string  `dynamodbav:"name"`
	Type       string  `dynamodbav:"type"`
	Required   bool    `dynamodbav:"required"`
	Validation *string `dynamodbav:"validation"`
}

type NotificationTemplate struct {
	Id               string             `dynamodbav:"id"`
	Name             string             `dynamodbav:"name"`
	IsHtml           bool               `dynamodbav:"isHtml"`
	TitleTemplate    string             `dynamodbav:"titleTemplate"`
	ContentsTemplate string             `dynamodbav:"contentsTemplate"`
	Description      string             `dynamodbav:"description"`
	CreatedBy        string             `dynamodbav:"createdBy"`
	CreatedAt        string             `dynamodbav:"createdAt"`
	UpdatedAt        *string            `dynamodbav:"updatedAt"`
	UpdatedBy        *string            `dynamodbav:"updatedBy"`
	HashKey          string             `dynamodbav:"hashKey"`
	Variables        []TemplateVariable `dynamodbav:"variables"`
}

type notificationTemplateGSINameKey struct {
	Id      string `dynamodbav:"id"`
	HashKey string `dynamodbav:"hashKey"`
	Name    string `dynamodbav:"name"`
}

func (nt NotificationTemplate) GetKey() (DynamoKey, error) {
	key := make(DynamoKey)

	templateId, err := attributevalue.Marshal(nt.Id)

	if err != nil {
		return key, fmt.Errorf("failed to make notification template key - %w", err)
	}

	key["id"] = templateId

	return key, nil
}

func (nt NotificationTemplate) GetGSINameKey() (DynamoKey, error) {
	key, err := nt.GetKey()

	if err != nil {
		return key, fmt.Errorf("failed to make notification template key - %w", err)
	}

	name, err := attributevalue.Marshal(nt.Name)

	if err != nil {
		return key, fmt.Errorf("failed to make notification template key - %w", err)
	}

	hashKey, err := attributevalue.Marshal(nt.HashKey)

	if err != nil {
		return key, fmt.Errorf("failed to make notification template key - %w", err)
	}

	key["name"] = name
	key["hashKey"] = hashKey

	return key, nil
}

func (ntsk *notificationTemplateGSINameKey) GetKey() (DynamoKey, error) {

	nt := NotificationTemplate{
		Id:      ntsk.Id,
		Name:    ntsk.Name,
		HashKey: ntsk.HashKey,
	}

	return nt.GetGSINameKey()
}

func (r *Registry) SaveTemplate(ctx context.Context, createdBy string, ntr dto.NotificationTemplateReq) (dto.NotificationTemplateCreatedResp, error) {

	resp := dto.NotificationTemplateCreatedResp{}

	id, err := uuid.NewV7()

	if err != nil {
		return resp, fmt.Errorf("failed to create id")
	}

	nt := NotificationTemplate{
		Id:               id.String(),
		Name:             ntr.Name,
		IsHtml:           ntr.IsHtml,
		TitleTemplate:    ntr.TitleTemplate,
		ContentsTemplate: ntr.ContentsTemplate,
		Description:      ntr.Description,
		HashKey:          NotificationsTemplateSyntheticKey,
		CreatedBy:        createdBy,
		CreatedAt:        time.Now().Format(time.RFC3339),
	}

	nt.Variables = make([]TemplateVariable, 0, len(ntr.Variables))

	for _, v := range ntr.Variables {
		nt.Variables = append(nt.Variables, TemplateVariable{
			Name:       v.Name,
			Type:       v.Type,
			Required:   v.Required,
			Validation: v.Validation,
		})
	}

	item, err := attributevalue.MarshalMap(nt)

	if err != nil {
		return resp, fmt.Errorf("failed to marshal notification template - %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(NotificationsTemplateTable),
		Item:      item,
	})

	if err != nil {
		return resp, fmt.Errorf("failed to put notification template - %w", err)
	}

	resp.Id = nt.Id
	resp.Name = nt.Name
	resp.CreatedAt = nt.CreatedAt

	return resp, nil
}

func (r *Registry) GetTemplates(ctx context.Context, filters dto.NotificationTemplateFilters) (sdto.Page[dto.NotificationTemplateInfoResp], error) {

	page := sdto.Page[dto.NotificationTemplateInfoResp]{}

	pageParams, err := makePageFilters(&notificationTemplateGSINameKey{}, filters.PageFilter)

	if err != nil {
		return page, fmt.Errorf("failed to make page params - %w", err)
	}

	keyExp := expression.
		Key(NotificationsTemplateNameGSIHashKey).
		Equal(expression.Value(NotificationsTemplateSyntheticKey))

	if filters.TemplateName != nil {
		sortKeyExp := expression.
			Key(NotificationsTemplateNameGSISortKey).
			BeginsWith(*filters.TemplateName)

		keyExp = keyExp.And(sortKeyExp)
	}

	expr, err := expression.
		NewBuilder().
		WithKeyCondition(keyExp).
		Build()

	if err != nil {
		return page, fmt.Errorf("failed to build expression - %w", err)
	}

	queryInput := dynamodb.QueryInput{
		TableName:                 aws.String(NotificationsTemplateTable),
		IndexName:                 aws.String(NotificationsTemplateNameGSI),
		ExclusiveStartKey:         pageParams.ExclusiveStartKey,
		Limit:                     pageParams.Limit,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	response, err := r.client.Query(ctx, &queryInput)

	if err != nil {
		return page, fmt.Errorf("failed to get notification templates - %w", err)
	}

	templates := []NotificationTemplate{}
	err = attributevalue.UnmarshalListOfMaps(response.Items, &templates)

	if err != nil {
		return page, fmt.Errorf("failed to unmarshal the notification templates - %w", err)
	}

	var nextToken *string = nil

	if len(response.LastEvaluatedKey) != 0 {
		key := notificationTemplateGSINameKey{}
		encoded, err := marshalNextToken(&key, response.LastEvaluatedKey)

		if err != nil {
			return page, err
		}

		nextToken = &encoded
	}

	items := make([]dto.NotificationTemplateInfoResp, 0, len(templates))

	for _, t := range templates {
		items = append(items, dto.NotificationTemplateInfoResp{
			Id:          t.Id,
			Name:        t.Name,
			Description: t.Description,
		})
	}

	page.NextToken = nextToken
	page.PrevToken = filters.NextToken
	page.ResultCount = len(items)
	page.Data = items

	return page, nil
}

func (r *Registry) GetTemplateDetails(ctx context.Context, templateId string) (sdto.NotificationTemplateDetails, error) {
	details := sdto.NotificationTemplateDetails{}

	key, err := NotificationTemplate{Id: templateId}.GetKey()

	if err != nil {
		return details, fmt.Errorf("failed to create key - %w", err)
	}

	response, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(NotificationsTemplateTable),
		Key:       key,
	})

	if err != nil {
		return details, fmt.Errorf("failed to get template - %w", err)
	}

	if len(response.Item) == 0 {
		return details, internal.EntityNotFound{
			Id:   templateId,
			Type: registry.NotificationTemplateType,
		}
	}

	template := NotificationTemplate{}
	err = attributevalue.UnmarshalMap(response.Item, &template)

	if err != nil {
		return details, fmt.Errorf("failed to unmarshal template - %w", err)
	}

	variables := make([]sdto.TemplateVariable, 0, len(template.Variables))
	for _, v := range template.Variables {
		variables = append(variables, sdto.TemplateVariable{
			Name:       v.Name,
			Type:       v.Type,
			Required:   v.Required,
			Validation: v.Validation,
		})
	}

	details.Id = template.Id
	details.Name = template.Name
	details.IsHtml = template.IsHtml
	details.Description = template.Description
	details.TitleTemplate = template.TitleTemplate
	details.ContentsTemplate = template.ContentsTemplate
	details.Variables = variables
	details.CreatedAt = template.CreatedAt
	details.CreatedBy = template.CreatedBy
	details.UpdatedAt = template.UpdatedAt
	details.UpdatedBy = template.UpdatedBy

	return details, nil
}

func (r *Registry) DeleteTemplate(ctx context.Context, id string) error {

	key, err := NotificationTemplate{Id: id}.GetKey()

	if err != nil {
		return fmt.Errorf("failed to make template key - %w", err)
	}

	_, err = r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(NotificationsTemplateTable),
		Key:       key,
	})

	if err != nil {
		return fmt.Errorf("failed to delete template - %w", err)
	}

	return nil
}

func (r *Registry) GetTemplateVariables(ctx context.Context, templateId string) ([]sdto.TemplateVariable, error) {

	variables := []sdto.TemplateVariable{}

	key, err := NotificationTemplate{Id: templateId}.GetKey()

	if err != nil {
		return variables, fmt.Errorf("failed to make notification template key - %w", err)
	}

	projExp := expression.
		ProjectionBuilder(expression.NamesList(expression.Name("variables")))

	expr, err := expression.
		NewBuilder().
		WithProjection(projExp).
		Build()

	if err != nil {
		return variables, fmt.Errorf("failed to build expression - %w", err)
	}

	resp, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:                aws.String(NotificationsTemplateTable),
		Key:                      key,
		ProjectionExpression:     expr.Projection(),
		ExpressionAttributeNames: expr.Names(),
	})

	if err != nil {
		return variables, fmt.Errorf("failed to retrieve the notification template variables - %w", err)
	}

	if len(resp.Item) == 0 {
		return variables, internal.EntityNotFound{
			Id:   templateId,
			Type: registry.NotificationTemplateType,
		}
	}

	tmp := struct {
		Variables []TemplateVariable `dynamodbav:"variables"`
	}{}

	err = attributevalue.UnmarshalMap(resp.Item, &tmp)

	if err != nil {
		return variables, fmt.Errorf("failed to unmarshal variables - %w", err)
	}

	variables = make([]sdto.TemplateVariable, 0, len(tmp.Variables))

	for _, v := range tmp.Variables {
		variables = append(variables, sdto.TemplateVariable{
			Name:       v.Name,
			Type:       v.Type,
			Required:   v.Required,
			Validation: v.Validation,
		})
	}

	return variables, nil
}
