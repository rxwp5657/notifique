package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/notifique/shared/cache"
)

type UserInfo struct {
	UserId string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
}

type cognitoUserInfo struct {
	client     *cognitoidentityprovider.Client
	userPoolID string
	cache      cache.Cache
}

func NewCognitoUserInfoProvider(client *cognitoidentityprovider.Client, userPoolID string) *cognitoUserInfo {
	return &cognitoUserInfo{
		client:     client,
		userPoolID: userPoolID,
	}
}

func (c *cognitoUserInfo) getInfoFromCache(ctx context.Context, userID string) (UserInfo, error, bool) {

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

func (c *cognitoUserInfo) getInfoFromCognitoPool(ctx context.Context, userID string) (UserInfo, error) {

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

func (c *cognitoUserInfo) GetUserInfo(ctx context.Context, userID string) (UserInfo, error) {

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
