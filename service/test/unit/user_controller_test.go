package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/notifique/service/internal"
	"github.com/notifique/service/internal/dto"
	"github.com/notifique/service/internal/registry"
	"github.com/notifique/service/internal/testutils"
	"github.com/notifique/shared/auth"
	"github.com/notifique/shared/cache"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	di "github.com/notifique/service/internal/di"
	sdto "github.com/notifique/shared/dto"
)

const userNotificationsUrl string = "/users/me/notifications"
const userConfigUrl string = "/users/me/notifications/config"
const userConfigKey = "notifications:endpoint:a2ec7c69d00e4549c50802368fe1c047:/users/1234/notifications/config*"
const userNotificationsKey = "notifications:endpoint:db31c468fd68d7f5824526c3acb4087e:/users/1234/notifications*"

func TestUserController(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	testApp, err := di.InjectMockedBackend(context.TODO(), controller)

	if err != nil {
		t.Fatalf("failed to create mocked backend - %v", err)
	}

	testGetUserNotifications(t, testApp.Engine, testApp)
	testGetUserConfig(t, testApp.Engine, testApp)
	testUpdateUserConfig(t, testApp.Engine, testApp)
	testSetReadStatus(t, testApp.Engine, testApp)
	testCreateNotifications(t, testApp.Engine, testApp)
}

func testGetUserNotifications(t *testing.T, e *gin.Engine, mock *di.MockedBackend) {

	testNotifications, err := testutils.MakeTestUserNotifications(3, testUserId)

	if err != nil {
		t.Fatal(err)
	}

	getNotifications := func(filters dto.UserNotificationFilters) *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		req, _ := http.NewRequest(http.MethodGet, userNotificationsUrl, nil)
		req.Header.Add(string(auth.UserHeader), testUserId)
		testutils.AddUserFilters(req, &filters)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Should be able to retrieve notifications page", func(t *testing.T) {
		mock.Registry.MockUserRegistry.
			EXPECT().
			GetUserNotifications(gomock.Any(), gomock.Any()).
			Return(sdto.Page[dto.UserNotification]{
				NextToken:   nil,
				PrevToken:   nil,
				ResultCount: len(testNotifications),
				Data:        testNotifications,
			}, nil)

		filters := dto.UserNotificationFilters{}

		w := getNotifications(filters)

		resp := sdto.Page[dto.UserNotification]{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, w.Code, http.StatusOK)
		assert.ElementsMatch(t, testNotifications, resp.Data)
	})

	t.Run("Should fail if there are duplicated topics on the filter", func(t *testing.T) {
		topic := "test"

		filters := dto.UserNotificationFilters{
			PageFilter: sdto.PageFilter{},
			Topics:     []string{topic, topic},
		}

		w := getNotifications(filters)

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := "Key: 'UserNotificationFilters.Topics' Error:Field validation for 'Topics' failed on the 'unique' tag"

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})
}

func testGetUserConfig(t *testing.T, e *gin.Engine, mock *di.MockedBackend) {

	cfg := testutils.MakeTestUserConfig(testUserId)

	getUserConfig := func(user string) *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		req, _ := http.NewRequest(http.MethodGet, userConfigUrl, nil)
		req.Header.Add("userId", user)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Can get the user's configuration", func(t *testing.T) {
		mock.Registry.MockUserRegistry.
			EXPECT().
			GetUserConfig(gomock.Any(), gomock.Any()).
			Return(cfg, nil)

		w := getUserConfig(testUserId)

		resp := dto.UserConfig{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, w.Code, http.StatusOK)
		assert.Equal(t, cfg, resp)
	})
}

func testUpdateUserConfig(t *testing.T, e *gin.Engine, mock *di.MockedBackend) {

	updateUserConfig := func(cfg dto.UserConfig) *httptest.ResponseRecorder {

		marshalled, _ := json.Marshal(cfg)
		reader := bytes.NewReader(marshalled)

		w := httptest.NewRecorder()

		req, _ := http.NewRequest(http.MethodPut, userConfigUrl, reader)
		req.Header.Add(string(auth.UserHeader), testUserId)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Should be able to update the user config", func(t *testing.T) {
		userConfig := testutils.MakeTestUserConfig(testUserId)
		userConfig.EmailConfig = dto.ChannelConfig{OptIn: false, SnoozeUntil: nil}
		userConfig.SMSConfig = dto.ChannelConfig{OptIn: true, SnoozeUntil: nil}

		mock.Registry.MockUserRegistry.
			EXPECT().
			UpdateUserConfig(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)

		mock.Cache.
			EXPECT().
			DelWithPrefix(gomock.Any(), cache.Key(userConfigKey)).
			Return(nil)

		w := updateUserConfig(userConfig)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Should fail if the snooze is on the past", func(t *testing.T) {
		snoozeUntil := time.Now().AddDate(0, 0, -10).Format(time.RFC3339)

		userConfig := testutils.MakeTestUserConfig(testUserId)
		userConfig.EmailConfig = dto.ChannelConfig{
			OptIn:       false,
			SnoozeUntil: &snoozeUntil,
		}

		w := updateUserConfig(userConfig)

		resp := map[string]string{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := "UserConfig.EmailConfig.SnoozeUntil' Error:Field validation for 'SnoozeUntil' failed on the 'future' tag"

		assert.Equal(t, w.Code, http.StatusBadRequest)
		assert.Contains(t, resp["error"], expectedMsg)
	})
}

func testSetReadStatus(t *testing.T, e *gin.Engine, mock *di.MockedBackend) {
	notificationId := uuid.NewString()
	readStatusUrl := fmt.Sprintf("%s/%s", userNotificationsUrl, notificationId)

	setReadStatus := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, readStatusUrl, nil)
		req.Header.Add(string(auth.UserHeader), testUserId)
		e.ServeHTTP(w, req)
		return w
	}

	t.Run("Should be able to mark notification as read", func(t *testing.T) {
		mock.Registry.MockUserRegistry.
			EXPECT().
			SetReadStatus(gomock.Any(), testUserId, notificationId).
			Return(nil)

		mock.Cache.
			EXPECT().
			DelWithPrefix(gomock.Any(), cache.Key(userNotificationsKey)).
			Return(nil)

		w := setReadStatus()
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Should return 404 if notification not found", func(t *testing.T) {
		mock.Registry.MockUserRegistry.
			EXPECT().
			SetReadStatus(gomock.Any(), testUserId, notificationId).
			Return(internal.EntityNotFound{Id: notificationId, Type: registry.NotificationType})

		w := setReadStatus()

		resp := make(map[string]string)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, resp["error"], "Notification not found")
	})

	t.Run("Should return 500 on unexpected errors", func(t *testing.T) {
		mock.Registry.MockUserRegistry.
			EXPECT().
			SetReadStatus(gomock.Any(), testUserId, notificationId).
			Return(errors.New("unexpected error"))

		w := setReadStatus()
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func testCreateNotifications(t *testing.T, e *gin.Engine, mock *di.MockedBackend) {

	createNotifications := func(batch []sdto.UserNotificationReq) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		marshalled, _ := json.Marshal(batch)
		reader := bytes.NewReader(marshalled)
		req, _ := http.NewRequest(http.MethodPost, "/users/notifications", reader)
		req.Header.Add(string(auth.UserHeader), testUserId)
		e.ServeHTTP(w, req)
		return w
	}

	tests := []struct {
		name           string
		notifications  []sdto.UserNotificationReq
		expectedCode   int
		expectedErrMsg string
		mockSetup      func()
	}{
		{
			name: "Should create notifications successfully",
			notifications: []sdto.UserNotificationReq{{
				UserId:   testUserId,
				Title:    "Test notification",
				Contents: "Test contents",
				Topic:    "test-topic",
			}},
			expectedCode: http.StatusNoContent,
			mockSetup: func() {
				testNotification := dto.UserNotification{
					Id:        uuid.NewString(),
					Title:     "Test notification",
					Contents:  "Test contents",
					Topic:     "test-topic",
					CreatedAt: time.Now().Format(time.RFC3339),
					ReadAt:    nil,
				}

				mock.Registry.MockUserRegistry.
					EXPECT().
					CreateNotifications(gomock.Any(), gomock.Any()).
					Return([]dto.UserNotification{testNotification}, nil).
					Times(1)

				mock.Cache.
					EXPECT().
					DelWithPrefix(gomock.Any(), cache.Key(userNotificationsKey)).
					Return(nil)

				mock.Broker.
					EXPECT().
					Publish(gomock.Any(), testUserId, testNotification).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "Should fail if title is empty",
			notifications: []sdto.UserNotificationReq{{
				UserId:   testUserId,
				Title:    "",
				Contents: "Test contents",
				Topic:    "test-topic",
			}},
			expectedCode:   http.StatusBadRequest,
			expectedErrMsg: "[0]: Key: 'UserNotificationReq.Title' Error:Field validation for 'Title' failed on the 'required' tag",
		},
		{
			name: "Should fail if contents is empty",
			notifications: []sdto.UserNotificationReq{{
				UserId:   testUserId,
				Title:    "Test title",
				Contents: "",
				Topic:    "test-topic",
			}},
			expectedCode:   http.StatusBadRequest,
			expectedErrMsg: "[0]: Key: 'UserNotificationReq.Contents' Error:Field validation for 'Contents' failed on the 'required' tag",
		},
		{
			name: "Should fail if topic is empty",
			notifications: []sdto.UserNotificationReq{{
				UserId:   testUserId,
				Title:    "Test title",
				Contents: "Test contents",
				Topic:    "",
			}},
			expectedCode:   http.StatusBadRequest,
			expectedErrMsg: "[0]: Key: 'UserNotificationReq.Topic' Error:Field validation for 'Topic' failed on the 'required' tag",
		},
		{
			name: "Should fail if title exceeds max length",
			notifications: []sdto.UserNotificationReq{{
				UserId:   testUserId,
				Title:    testutils.MakeStrWithSize(121),
				Contents: "Test contents",
				Topic:    "test-topic",
			}},
			expectedCode:   http.StatusBadRequest,
			expectedErrMsg: "[0]: Key: 'UserNotificationReq.Title' Error:Field validation for 'Title' failed on the 'max' tag",
		},
		{
			name: "Should fail if contents exceeds max length",
			notifications: []sdto.UserNotificationReq{{
				UserId:   testUserId,
				Title:    "Test title",
				Contents: testutils.MakeStrWithSize(1025),
				Topic:    "test-topic",
			}},
			expectedCode:   http.StatusBadRequest,
			expectedErrMsg: "[0]: Key: 'UserNotificationReq.Contents' Error:Field validation for 'Contents' failed on the 'max' tag",
		},
		{
			name: "Should fail if topic exceeds max length",
			notifications: []sdto.UserNotificationReq{{
				UserId:   testUserId,
				Title:    "Test title",
				Contents: "Test contents",
				Topic:    testutils.MakeStrWithSize(121),
			}},
			expectedCode:   http.StatusBadRequest,
			expectedErrMsg: "[0]: Key: 'UserNotificationReq.Topic' Error:Field validation for 'Topic' failed on the 'max' tag",
		},
		{
			name: "Should fail if image URL is invalid",
			notifications: []sdto.UserNotificationReq{{
				UserId:   testUserId,
				Title:    "Test title",
				Contents: "Test contents",
				Topic:    "test-topic",
				Image:    testutils.StrPtr("invalid-url"),
			}},
			expectedCode:   http.StatusBadRequest,
			expectedErrMsg: "[0]: Key: 'UserNotificationReq.Image' Error:Field validation for 'Image' failed on the 'uri' tag",
		},
		{
			name: "Should fail if user ID is empty",
			notifications: []sdto.UserNotificationReq{{
				UserId:   "",
				Title:    "Test title",
				Contents: "Test contents",
				Topic:    "test-topic",
			}},
			expectedCode:   http.StatusBadRequest,
			expectedErrMsg: "[0]: Key: 'UserNotificationReq.UserId' Error:Field validation for 'UserId' failed on the 'required' tag",
		},
		{
			name: "Should fail with internal error",
			notifications: []sdto.UserNotificationReq{{
				UserId:   testUserId,
				Title:    "Test title",
				Contents: "Test contents",
				Topic:    "test-topic",
			}},
			expectedCode: http.StatusInternalServerError,
			mockSetup: func() {
				mock.Registry.MockUserRegistry.
					EXPECT().
					CreateNotifications(gomock.Any(), gomock.Any()).
					Return([]dto.UserNotification{}, errors.New("internal error")).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetup != nil {
				tt.mockSetup()
			}

			w := createNotifications(tt.notifications)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedErrMsg != "" {
				resp := make(map[string]string)
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Contains(t, resp["error"], tt.expectedErrMsg)
			}
		})
	}
}
