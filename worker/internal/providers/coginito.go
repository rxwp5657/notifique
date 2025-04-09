package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/notifique/shared/cache"
)

type UserPoolID string

type UserInfo struct {
	UserId string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
}

type CognitoIdentityProviderConfig struct {
	BaseEndpoint *string
	Region       *string
}

type CognitoIdentityProviderConfigurator interface {
	GetCognitoIdentityProviderConfig() CognitoIdentityProviderConfig
}

type CognitoUserInfoConfigurator interface {
	GetUserPoolId() (UserPoolID, error)
}

type CognitoUserInfoCfg struct {
	UserPoolID UserPoolID
	Cache      cache.Cache
	Client     *cognitoidentityprovider.Client
}

type CognitoUserInfo struct {
	client     *cognitoidentityprovider.Client
	userPoolID string
	cache      cache.Cache
}

func (c *CognitoUserInfo) getInfoFromCache(ctx context.Context, userID string) (UserInfo, error, bool) {

	info := UserInfo{}
	userInfo, err, ok := c.cache.Get(ctx, cache.Key(userID))

	if err != nil {
		return info, fmt.Errorf("error getting user info from cache: %w", err), false
	}

	if !ok {
		return info, nil, false
	}

	err = json.Unmarshal([]byte(userInfo), &info)

	if err != nil {
		return info, fmt.Errorf("error unmarshalling user info: %w", err), false
	}

	return info, nil, true
}

func (c *CognitoUserInfo) getInfoFromCognitoPool(ctx context.Context, userID string) (UserInfo, error) {

	info := UserInfo{UserId: userID}

	input := &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: &c.userPoolID,
		Username:   &userID,
	}

	result, err := c.client.AdminGetUser(ctx, input)

	if err != nil {
		return info, err
	}

	for _, attr := range result.UserAttributes {
		switch *attr.Name {
		case "email":
			info.Email = *attr.Value
		case "phone_number":
			info.Phone = *attr.Value
		case "name":
			info.Name = *attr.Value
		}
	}

	return info, nil
}

func (c *CognitoUserInfo) GetUserInfo(ctx context.Context, userID string) (UserInfo, error) {

	userInfo, err, ok := c.getInfoFromCache(ctx, userID)

	if err != nil || ok {
		return userInfo, err
	}

	userInfo, err = c.getInfoFromCognitoPool(ctx, userID)

	if err != nil {
		return userInfo, fmt.Errorf("error getting user info from cognito: %w", err)
	}

	marshalled, err := json.Marshal(userInfo)

	if err != nil {
		return userInfo, fmt.Errorf("error marshalling user info: %w", err)
	}

	err = c.cache.Set(ctx, cache.Key(userID), string(marshalled), time.Hour*24)

	if err != nil {
		return userInfo, fmt.Errorf("error setting user info in cache: %w", err)
	}

	return userInfo, nil
}

func NewCognitoIdentityProvider(c CognitoIdentityProviderConfigurator) (*cognitoidentityprovider.Client, error) {
	clientCfg := c.GetCognitoIdentityProviderConfig()

	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		return nil, fmt.Errorf("failed to load default config: %w", err)
	}

	client := cognitoidentityprovider.NewFromConfig(cfg, func(o *cognitoidentityprovider.Options) {
		if clientCfg.BaseEndpoint != nil {
			o.BaseEndpoint = clientCfg.BaseEndpoint
		}

		if clientCfg.Region != nil {
			o.Region = *clientCfg.Region
		}
	})

	return client, nil
}

func NewCognitoUserInfoProvider(cfg CognitoUserInfoCfg) *CognitoUserInfo {
	return &CognitoUserInfo{
		client:     cfg.Client,
		userPoolID: string(cfg.UserPoolID),
		cache:      cfg.Cache,
	}
}
