package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	c "github.com/notifique/controllers"
	"github.com/notifique/dto"
	"github.com/notifique/internal"
	"github.com/notifique/routes"
	"github.com/stretchr/testify/assert"
)

const userId = "12345"

func getStorage() internal.InMemoryStorage {
	return internal.MakeInMemoryStorage()
}

func makeNotificationRouter(ns c.NotificationStorage) *gin.Engine {
	r := gin.Default()
	routes.SetupNotificationRoutes(r, ns)
	return r
}

func makeStrWithSize(size int) string {
	field, i := "", 0

	for i < size {
		field += "+"
		i += 1
	}

	return field
}

func copyNotification(notification dto.NotificationReq) dto.NotificationReq {
	cp := notification
	cp.Recipients = make([]string, len(notification.Recipients))
	cp.Channels = make([]string, len(notification.Channels))

	copy(cp.Recipients, notification.Recipients)
	copy(cp.Channels, notification.Channels)

	return cp
}

func TestGetUserNotifications(t *testing.T) {
	storage := getStorage()
	router := makeNotificationRouter(&storage)

	testNofitication := dto.NotificationReq{
		Title:      "Notification 1",
		Contents:   "Notification Contents 1",
		Topic:      "Testing",
		Priority:   "HIGH",
		Recipients: []string{userId},
		Channels:   []string{"in-app", "e-mail"},
	}

	ctx := context.Background()
	storage.SaveNotification(ctx, userId, testNofitication)

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
	router := makeNotificationRouter(&storage)

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
	router := makeNotificationRouter(&storage)

	testNofitication := dto.NotificationReq{
		Title:      "Notification 1",
		Contents:   "Notification Contents 1",
		Topic:      "Testing",
		Priority:   "LOW",
		Recipients: []string{userId},
		Channels:   []string{"in-app", "e-mail"},
	}

	ctx := context.Background()
	notificationId, _ := storage.SaveNotification(ctx, userId, testNofitication)

	w := httptest.NewRecorder()

	url := fmt.Sprintf("/v0/notifications/%s", notificationId)
	req, _ := http.NewRequest("PATCH", url, nil)
	req.Header.Add("userId", userId)

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestSetReadStatusOnMissingNotification(t *testing.T) {
	storage := getStorage()
	router := makeNotificationRouter(&storage)

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

func TestUpdateUserConfig(t *testing.T) {
	storage := getStorage()
	router := makeNotificationRouter(&storage)

	callWithUserConfig := func(config []dto.ChannelConfig) (*http.Request, *httptest.ResponseRecorder) {

		w := httptest.NewRecorder()

		body := make(map[string][]dto.ChannelConfig)
		body["config"] = config

		marshalled, _ := json.Marshal(body)
		reader := bytes.NewReader(marshalled)

		req, _ := http.NewRequest("PATCH", "/v0/notifications/config", reader)
		req.Header.Add("userId", userId)

		router.ServeHTTP(w, req)

		return req, w
	}

	t.Run("Can update the user config", func(t *testing.T) {

		snoozeUntil := time.Now().AddDate(0, 0, 10).Format(time.RFC3339)

		userConfig := []dto.ChannelConfig{
			{Channel: "e-mail", OptIn: false, SnoozeUntil: nil},
			{Channel: "sms", OptIn: true, SnoozeUntil: &snoozeUntil},
		}

		_, w := callWithUserConfig(userConfig)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("Should fail on bad channel", func(t *testing.T) {

		userConfig := []dto.ChannelConfig{
			{Channel: "BadChannel", OptIn: true, SnoozeUntil: nil},
		}

		_, w := callWithUserConfig(userConfig)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("Should fail when passing a snooze date in the past", func(t *testing.T) {
		snoozeUntil := time.Now().AddDate(0, 0, -10).Format(time.RFC3339)

		userConfig := []dto.ChannelConfig{
			{Channel: "sms", OptIn: true, SnoozeUntil: &snoozeUntil},
		}

		_, w := callWithUserConfig(userConfig)

		assert.Equal(t, 400, w.Code)
	})
}

func TestCreateNotification(t *testing.T) {

	storage := getStorage()
	router := makeNotificationRouter(&storage)

	testNofitication := dto.NotificationReq{
		Title:      "Notification 1",
		Contents:   "Notification Contents 1",
		Topic:      "Testing",
		Priority:   "MEDIUM",
		Recipients: []string{userId},
		Channels:   []string{"in-app", "e-mail"},
	}

	callWithNotification := func(notification dto.NotificationReq) (*http.Request, *httptest.ResponseRecorder) {
		body, _ := json.Marshal(notification)
		reader := bytes.NewReader(body)

		w := httptest.NewRecorder()

		req, _ := http.NewRequest("POST", "/v0/notifications", reader)
		req.Header.Add("userId", userId)

		router.ServeHTTP(w, req)

		return req, w
	}

	t.Run("Can create new notifications", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		_, w := callWithNotification(notification)

		assert.Equal(t, 204, w.Code)
	})

	t.Run("Should fail on bad channel", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		notification.Channels = append(notification.Channels, "Bad Channel")

		_, w := callWithNotification(notification)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("Should fail on long title", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		notification.Title = makeStrWithSize(200)

		_, w := callWithNotification(notification)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("Should fail on long contents", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		notification.Contents = makeStrWithSize(1025)

		_, w := callWithNotification(notification)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("Should fail on long topic", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		notification.Topic = makeStrWithSize(200)

		_, w := callWithNotification(notification)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("Should fail on duplicated recipients", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		notification.Recipients = append(notification.Recipients, userId)

		_, w := callWithNotification(notification)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("Should fail on invalid priority", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		notification.Priority = "Bad Priority"

		_, w := callWithNotification(notification)

		assert.Equal(t, 400, w.Code)
	})
}
