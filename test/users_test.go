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
	"github.com/notifique/routes"
	"github.com/stretchr/testify/assert"
)

func makeUsersRoute(us c.UserStorage) *gin.Engine {
	r := gin.Default()
	routes.SetupUsersRoutes(r, us)
	return r
}

func TestGetUserNotifications(t *testing.T) {
	storage := getStorage()
	router := makeUsersRoute(&storage)

	testNofitication := dto.UserNotificationReq{
		Title:    "Notification 1",
		Contents: "Notification Contents 1",
		Topic:    "Testing",
	}

	ctx := context.Background()
	storage.CreateUserNotification(ctx, userId, testNofitication)

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/v0/users/me/notifications", nil)
	req.Header.Add("userId", userId)

	router.ServeHTTP(w, req)

	page := dto.Page[dto.UserNotification]{}

	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("Failed to unmarshal response")
	}

	if len(page.Data) == 0 {
		t.Fatalf("Num notifications is zero, expected at least one")
	}

	notification := page.Data[0]

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, notification.Title, testNofitication.Title)
	assert.Equal(t, notification.Contents, testNofitication.Contents)
	assert.Equal(t, notification.Topic, testNofitication.Topic)
	assert.Nil(t, notification.ReadAt)
	assert.NotEmpty(t, notification.CreatedAt)

	assert.Equal(t, 1, page.CurrentPage)
	assert.Nil(t, page.NextPage)
	assert.Nil(t, page.PrevPage)
	assert.Equal(t, 1, page.TotalPages)
	assert.Equal(t, 1, page.TotalRecords)
}

func TestGetUserConfiguration(t *testing.T) {
	storage := getStorage()
	router := makeUsersRoute(&storage)

	expectedConfig := []dto.ChannelConfig{
		{Channel: "e-mail", OptIn: true},
		{Channel: "sms", OptIn: true},
		{Channel: "in-app", OptIn: true},
	}

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/v0/users/me/notifications/config", nil)
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
	router := makeUsersRoute(&storage)

	testNofitication := dto.UserNotificationReq{
		Title:    "Notification 1",
		Contents: "Notification Contents 1",
		Topic:    "Testing",
	}

	ctx := context.Background()
	notificationId, _ := storage.CreateUserNotification(ctx, userId, testNofitication)

	w := httptest.NewRecorder()

	url := fmt.Sprintf("/v0/users/me/notifications/%s", notificationId)
	req, _ := http.NewRequest("PATCH", url, nil)
	req.Header.Add("userId", userId)

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestSetReadStatusOnMissingNotification(t *testing.T) {
	storage := getStorage()
	router := makeUsersRoute(&storage)

	notificationId := uuid.NewString()

	errorTemplate := "Notification %v not found"
	expectedError := fmt.Sprintf(errorTemplate, notificationId)

	w := httptest.NewRecorder()

	url := fmt.Sprintf("/v0/users/me/notifications/%s", notificationId)
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
	router := makeUsersRoute(&storage)

	callWithUserConfig := func(config []dto.ChannelConfig) (*http.Request, *httptest.ResponseRecorder) {

		w := httptest.NewRecorder()

		body := make(map[string][]dto.ChannelConfig)
		body["config"] = config

		marshalled, _ := json.Marshal(body)
		reader := bytes.NewReader(marshalled)

		req, _ := http.NewRequest("PATCH", "/v0/users/me/notifications/config", reader)
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
