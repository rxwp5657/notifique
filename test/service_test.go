package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/notifique/dto"
	"github.com/notifique/internal"
	"github.com/notifique/routes"
	"github.com/stretchr/testify/assert"
)

func getStorage() internal.InMemoryStorage {
	return internal.MakeInMemoryStorage()
}

func TestGetUserNotifications(t *testing.T) {
	storage := getStorage()
	router := routes.SetupRoutes(&storage)

	userId := "12345"

	testNofitication := dto.NotificationReq{
		Title:      "Notification 1",
		Contents:   "Notification Contents 1",
		Topic:      "Testing",
		Recipients: []string{userId},
		Channels:   []string{"in-app e-mail"},
	}

	ctx := context.Background()
	storage.SaveNotification(ctx, testNofitication)

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/v0/users/notifications", nil)
	req.Header.Add("userId", userId)

	router.ServeHTTP(w, req)

	notifications := make([]dto.UserNotificationResp, 0)

	if err := json.Unmarshal(w.Body.Bytes(), &notifications); err != nil {
		t.Fatalf("Failed to unmarshal response")
	}

	if len(notifications) == 0 {
		t.Fatalf("Num notifications is zero, expected at least one")
	}

	notification := notifications[0]

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, notification.Title, testNofitication.Title)
	assert.Equal(t, notification.Contents, testNofitication.Contents)
	assert.Equal(t, notification.Topic, testNofitication.Topic)
	assert.Nil(t, notification.ReadAt)
	assert.ElementsMatch(t, notification.Channels, testNofitication.Channels)
	assert.NotEmpty(t, notification.CreatedAt)
}

func TestGetUserConfiguration(t *testing.T) {
	storage := getStorage()
	router := routes.SetupRoutes(&storage)

	userId := "12345"

	expectedConfig := []dto.UserConfigResp{
		{Channel: "e-mail", OptedIn: true},
		{Channel: "sms", OptedIn: true},
		{Channel: "in-app", OptedIn: true},
	}

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/v0/users/notifications/config", nil)
	req.Header.Add("userId", userId)

	router.ServeHTTP(w, req)

	userConfig := make([]dto.UserConfigResp, 0)

	if err := json.Unmarshal(w.Body.Bytes(), &userConfig); err != nil {
		t.FailNow()
	}

	assert.Equal(t, 200, w.Code)
	assert.ElementsMatch(t, expectedConfig, userConfig)
}
