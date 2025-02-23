package dynamoregistry

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"

	"github.com/notifique/internal/server/dto"
)

const (
	NotificationsTemplateTable  = "NotificationsTemplate"
	NotificationTemplateHashKey = "id"
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
	TitleTemplate    string             `dynamodbav:"titleTemplate"`
	ContentsTemplate string             `dynamodbav:"contentsTemplate"`
	Description      string             `dynamodbav:"description"`
	CreatedBy        string             `dynamodbav:"createdBy"`
	CreatedAt        string             `dynamodbav:"createdAt"`
	Variables        []TemplateVariable `dynamodbav:"variables"`
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

func (r *Registry) SaveTemplate(ctx context.Context, createdBy string, ntr dto.NotificationTemplateReq) (dto.NotificationTemplateCreatedResp, error) {

	resp := dto.NotificationTemplateCreatedResp{}

	nt := NotificationTemplate{
		Id:               uuid.NewString(),
		Name:             ntr.Name,
		TitleTemplate:    ntr.TitleTemplate,
		ContentsTemplate: ntr.ContentsTemplate,
		Description:      ntr.Description,
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
