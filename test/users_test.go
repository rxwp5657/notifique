package test

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

	"github.com/notifique/controllers"
	"github.com/notifique/dto"
	storage "github.com/notifique/internal/storage/dynamodb"
	pstorage "github.com/notifique/internal/storage/postgres"
	"github.com/notifique/routes"
	"github.com/stretchr/testify/assert"
)

type UserNotificationsTester interface {
	controllers.UserStorage
	CreateUserNotification(ctx context.Context, userId string, un dto.UserNotification) error
	DeleteUserNotification(ctx context.Context, userId string, un dto.UserNotification) error
}

func makeUserStorage(t storageType, uri string) (UserNotificationsTester, error) {
	switch t {
	case DYNAMODB:
		client, err := storage.MakeClient(&uri)
		if err != nil {
			return nil, err
		}
		storage := storage.MakeDynamoDBStorage(client)
		return &storage, nil
	case POSTGRES:
		container, err := pstorage.MakePostgresStorage(uri)
		if err != nil {
			return nil, err
		}
		return container, err
	default:
		return nil, fmt.Errorf("invalid option - %s", t)
	}
}

func TestUserController(t *testing.T) {

	var container Container

	container, err := setupContainer(POSTGRES)

	if err != nil {
		t.Fatalf("failed to create container - %s", err)
	}

	uri := container.GetURI()

	storage, err := makeUserStorage(POSTGRES, uri)

	if err != nil {
		t.Fatalf("failed to create storage - %s", err)
	}

	router := gin.Default()
	routes.SetupUsersRoutes(router, storage)

	userId := "1234"

	t.Run("TestGetUserNotifications", func(t *testing.T) {

		numNotifications := 3
		testNotifications, err := createTestUserNotifications(numNotifications, userId, storage)

		if err != nil {
			t.Fatalf("failed to create user notifications - %s", err)
		}

		getUserNotifications := func(filters *dto.PageFilter) *httptest.ResponseRecorder {

			w := httptest.NewRecorder()

			req, _ := http.NewRequest("GET", "/v0/users/me/notifications", nil)
			req.Header.Add("userId", userId)

			addPaginationFilters(req, filters)

			router.ServeHTTP(w, req)

			return w
		}

		t.Run("Can get default page of user notifications", func(t *testing.T) {

			w := getUserNotifications(nil)

			page := dto.Page[dto.UserNotification]{}

			if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
				t.Fatal("Failed to unmarshal response")
			}

			if len(page.Data) == 0 {
				t.Fatal("Num notifications is zero, expected at least one")
			}

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, reverse(testNotifications), page.Data)

			assert.Nil(t, page.PrevToken)
			assert.Nil(t, page.NextToken)
			assert.Equal(t, numNotifications, page.ResultCount)
		})

		t.Run("Can paginate user notifications", func(t *testing.T) {

			maxResults := 1
			filters := dto.PageFilter{
				NextToken:  nil,
				MaxResults: &maxResults,
			}

			w := getUserNotifications(&filters)

			page := dto.Page[dto.UserNotification]{}

			if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
				t.Fatal("Failed to unmarshal response")
			}

			if len(page.Data) == 0 {
				t.Fatal("Num notifications is zero, expected at least one")
			}

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, reverse(testNotifications)[:1], page.Data)

			assert.Nil(t, page.PrevToken)
			assert.NotNil(t, page.NextToken)
			assert.Equal(t, maxResults, page.ResultCount)
		})

		deleteTestUserNotifications(userId, testNotifications, storage)
	})

	t.Run("TestSetReadStatus", func(t *testing.T) {

		testNotifications, err := createTestUserNotifications(1, userId, storage)

		if err != nil {
			t.Fatalf("failed to create user notifications - %s", err)
		}

		testNotification := testNotifications[0]

		setReadStatus := func(userId, notificationId string) *httptest.ResponseRecorder {

			w := httptest.NewRecorder()

			url := fmt.Sprintf("/v0/users/me/notifications/%s", notificationId)
			req, _ := http.NewRequest("PATCH", url, nil)
			req.Header.Add("userId", userId)

			router.ServeHTTP(w, req)

			return w
		}

		t.Run("Should be able to set the notification read status", func(t *testing.T) {
			w := setReadStatus(userId, testNotification.Id)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("Should fail to set the read status on a non-existing notification", func(t *testing.T) {
			notificationId := uuid.NewString()
			w := setReadStatus(userId, notificationId)

			resp := make(map[string]string, 0)

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			errTemplate := "Notification %v not found"
			errMsg := fmt.Sprintf(errTemplate, notificationId)

			assert.Equal(t, http.StatusNotFound, w.Code)
			assert.Equal(t, resp["error"], errMsg)
		})
	})

	t.Run("TestUserConfig", func(t *testing.T) {

		getUserConfig := func(userId string) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()

			req, _ := http.NewRequest("GET", "/v0/users/me/notifications/config", nil)
			req.Header.Add("userId", userId)

			router.ServeHTTP(w, req)

			return w
		}

		updateUserConfig := func(userId string, config dto.UserConfig) *httptest.ResponseRecorder {

			w := httptest.NewRecorder()

			marshalled, _ := json.Marshal(config)
			reader := bytes.NewReader(marshalled)

			req, _ := http.NewRequest("PUT", "/v0/users/me/notifications/config", reader)
			req.Header.Add("userId", userId)

			router.ServeHTTP(w, req)

			return w
		}

		t.Run("CanGetUserConfig", func(t *testing.T) {

			expectedConfig := dto.UserConfig{
				EmailConfig: dto.ChannelConfig{OptIn: true, SnoozeUntil: nil},
				SMSConfig:   dto.ChannelConfig{OptIn: true, SnoozeUntil: nil},
				InAppConfig: dto.ChannelConfig{OptIn: true, SnoozeUntil: nil},
			}

			w := getUserConfig(userId)

			var userConfig dto.UserConfig

			if err := json.Unmarshal(w.Body.Bytes(), &userConfig); err != nil {
				t.FailNow()
			}

			assert.Equal(t, 200, w.Code)
			assert.Equal(t, expectedConfig, userConfig)
		})

		t.Run("CanUpdateUserConfig", func(t *testing.T) {

			snoozeUntil := time.Now().AddDate(0, 0, 10).Format(time.RFC3339)

			userConfig := dto.UserConfig{
				EmailConfig: dto.ChannelConfig{OptIn: false, SnoozeUntil: nil},
				SMSConfig:   dto.ChannelConfig{OptIn: true, SnoozeUntil: &snoozeUntil},
			}

			w := updateUserConfig(userId, userConfig)

			assert.Equal(t, 200, w.Code)
		})

		t.Run("ShouldFailOnSnoozeTimeInThePast", func(t *testing.T) {
			snoozeUntil := time.Now().AddDate(0, 0, -10).Format(time.RFC3339)

			userConfig := dto.UserConfig{
				SMSConfig: dto.ChannelConfig{OptIn: true, SnoozeUntil: &snoozeUntil},
			}

			w := updateUserConfig(userId, userConfig)

			resp := make(map[string]string, 0)

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			errMsg := "Error:Field validation for 'SnoozeUntil' failed"

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, resp["error"], errMsg)
		})
	})
}
