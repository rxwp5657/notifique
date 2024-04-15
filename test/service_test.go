package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/notifique/dto"
	"github.com/notifique/internal"
	"github.com/notifique/routes"
	"github.com/stretchr/testify/assert"
)

const userId = "12345"

func getStorage() internal.InMemoryStorage {
	return internal.MakeInMemoryStorage()
}

func TestGetUserNotifications(t *testing.T) {
	storage := getStorage()
	router := routes.SetupRoutes(&storage)

	testNofitication := dto.NotificationReq{
		Title:      "Notification 1",
		Contents:   "Notification Contents 1",
		Topic:      "Testing",
		Recipients: []string{userId},
		Channels:   []string{"in-app", "e-mail"},
	}

	ctx := context.Background()
	storage.SaveNotification(ctx, testNofitication)

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/v0/notifications", nil)
	req.Header.Add("userId", userId)

	router.ServeHTTP(w, req)

	notifications := make([]dto.UserNotification, 0)

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
	assert.NotEmpty(t, notification.CreatedAt)
}

func TestGetUserConfiguration(t *testing.T) {
	storage := getStorage()
	router := routes.SetupRoutes(&storage)

	expectedConfig := []dto.ChannelConfig{
		{Channel: "e-mail", OptIn: true},
		{Channel: "sms", OptIn: true},
		{Channel: "in-app", OptIn: true},
	}

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/v0/notifications/config", nil)
	req.Header.Add("userId", userId)

	router.ServeHTTP(w, req)

	userConfig := make([]dto.ChannelConfig, 0)

	if err := json.Unmarshal(w.Body.Bytes(), &userConfig); err != nil {
		t.FailNow()
	}

	assert.Equal(t, 200, w.Code)
	assert.ElementsMatch(t, expectedConfig, userConfig)
}

func TestSetReadStatus(t *testing.T) {
	storage := getStorage()
	router := routes.SetupRoutes(&storage)

	testNofitication := dto.NotificationReq{
		Title:      "Notification 1",
		Contents:   "Notification Contents 1",
		Topic:      "Testing",
		Recipients: []string{userId},
		Channels:   []string{"in-app", "e-mail"},
	}

	ctx := context.Background()
	notificationId, _ := storage.SaveNotification(ctx, testNofitication)

	w := httptest.NewRecorder()

	url := fmt.Sprintf("/v0/notifications/%s", notificationId)
	req, _ := http.NewRequest("PATCH", url, nil)
	req.Header.Add("userId", userId)

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestSetReadStatusOnMissingNotification(t *testing.T) {
	storage := getStorage()
	router := routes.SetupRoutes(&storage)

	notificationId := uuid.NewString()

	errorTemplate := "Notification %v not found"
	expectedError := fmt.Sprintf(errorTemplate, notificationId)

	w := httptest.NewRecorder()

	url := fmt.Sprintf("/v0/notifications/%s", notificationId)
	req, _ := http.NewRequest("PATCH", url, nil)
	req.Header.Add("userId", userId)

	router.ServeHTTP(w, req)

	resp := make(map[string]string)

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.FailNow()
	}

	assert.Equal(t, 404, w.Code)
	assert.Equal(t, expectedError, resp["error"])
}
