package dynamoregistry

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/notifique/internal/server/dto"
)

const (
	UserConfigTable       = "UserConfig"
	UserConfigHashKey     = "userId"
	UserConfigEmailKey    = "emailConfig"
	UserConfigSmsKey      = "smsConfig"
	UserConfigInAppKey    = "inAppConfig"
	UserConfigSnoozeUntil = "snoozeUntil"
	UserConfigOptIn       = "optIn"
)

type ChannelConfig struct {
	OptIn       bool    `dynamodbav:"optIn"`
	SnoozeUntil *string `dynamodbav:"snoozeUntil"`
}

type UserConfig struct {
	UserId      string        `dynamodbav:"userId"`
	EmailConfig ChannelConfig `dynamodbav:"emailConfig"`
	SMSConfig   ChannelConfig `dynamodbav:"smsConfig"`
	InAppConfig ChannelConfig `dynamodbav:"inAppConfig"`
}

func (cfg *UserConfig) GetKey() (DynamoKey, error) {
	key := make(map[string]types.AttributeValue)

	userId, err := attributevalue.Marshal(cfg.UserId)

	if err != nil {
		return key, fmt.Errorf("failed to make user config key - %w", err)
	}

	key["userId"] = userId

	return key, nil
}

func (s *Registry) getUserConfig(ctx context.Context, userId string) (*UserConfig, error) {

	tmpConfig := UserConfig{UserId: userId}
	key, err := tmpConfig.GetKey()

	if err != nil {
		return nil, err
	}

	resp, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(UserConfigTable),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get the user config - %w", err)
	}

	if len(resp.Item) == 0 {
		return nil, nil
	}

	config := UserConfig{}

	err = attributevalue.UnmarshalMap(resp.Item, &config)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall the user config - %w", err)
	}

	return &config, nil
}

func (s *Registry) createUserConfig(ctx context.Context, userId string) (*UserConfig, error) {
	config := UserConfig{
		UserId:      userId,
		EmailConfig: ChannelConfig{OptIn: true, SnoozeUntil: nil},
		SMSConfig:   ChannelConfig{OptIn: true, SnoozeUntil: nil},
		InAppConfig: ChannelConfig{OptIn: true, SnoozeUntil: nil},
	}

	item, err := attributevalue.MarshalMap(config)

	if err != nil {
		return nil, fmt.Errorf("failed to marshall the user config - %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(UserConfigTable),
		Item:      item,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to store notification - %w", err)
	}

	return &config, nil
}

func (s *Registry) GetUserConfig(ctx context.Context, userId string) (dto.UserConfig, error) {

	config, err := s.getUserConfig(ctx, userId)

	if err != nil {
		return dto.UserConfig{}, err
	}

	if config == nil {
		config, err = s.createUserConfig(ctx, userId)

		if err != nil {
			return dto.UserConfig{}, err
		}
	}

	cfg := dto.UserConfig{
		EmailConfig: dto.ChannelConfig{
			OptIn:       config.EmailConfig.OptIn,
			SnoozeUntil: config.EmailConfig.SnoozeUntil,
		},
		SMSConfig: dto.ChannelConfig{
			OptIn:       config.SMSConfig.OptIn,
			SnoozeUntil: config.SMSConfig.SnoozeUntil,
		},
		InAppConfig: dto.ChannelConfig{
			OptIn:       config.InAppConfig.OptIn,
			SnoozeUntil: config.InAppConfig.SnoozeUntil,
		},
	}

	return cfg, nil
}

func (s *Registry) UpdateUserConfig(ctx context.Context, userId string, config dto.UserConfig) error {

	usrCfg, err := s.getUserConfig(ctx, userId)

	if err != nil {
		return err
	}

	if usrCfg == nil {
		_, err := s.createUserConfig(ctx, userId)

		if err != nil {
			return err
		}
	}

	makeKeyFormatter := func(key string) func(string) expression.NameBuilder {
		return func(subKey string) expression.NameBuilder {
			return expression.Name(fmt.Sprintf("%v.%v", key, subKey))
		}
	}

	emailFmt := makeKeyFormatter(UserConfigEmailKey)
	smsFmt := makeKeyFormatter(UserConfigSmsKey)
	inAppFmt := makeKeyFormatter(UserConfigInAppKey)

	update := expression.Set(emailFmt(UserConfigOptIn), expression.Value(config.EmailConfig.OptIn))
	update.Set(emailFmt(UserConfigSnoozeUntil), expression.Value(config.EmailConfig.SnoozeUntil))
	update.Set(smsFmt(UserConfigOptIn), expression.Value(config.SMSConfig.OptIn))
	update.Set(smsFmt(UserConfigSnoozeUntil), expression.Value(config.SMSConfig.SnoozeUntil))
	update.Set(inAppFmt(UserConfigOptIn), expression.Value(config.InAppConfig.OptIn))
	update.Set(inAppFmt(UserConfigSnoozeUntil), expression.Value(config.InAppConfig.SnoozeUntil))

	expr, err := expression.NewBuilder().WithUpdate(update).Build()

	if err != nil {
		return fmt.Errorf("failed to make update query - %w", err)
	}

	tmpConfig := UserConfig{UserId: userId}
	key, err := tmpConfig.GetKey()

	if err != nil {
		return err
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(UserConfigTable),
		Key:                       key,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
		ConditionExpression:       expr.Condition(),
		ReturnValues:              types.ReturnValueUpdatedNew,
	})

	if err != nil {
		return fmt.Errorf("failed to set the read status - %w", err)
	}

	return nil
}
