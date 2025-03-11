package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/notifique/internal/dto"
	"github.com/notifique/internal/testutils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	di "github.com/notifique/internal/di"
	mk "github.com/notifique/internal/testutils/mocks"
)

const userNotificationsUrl string = "/users/me/notifications"
const userConfigUrl string = "/users/me/notifications/config"

func TestUserController(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	testApp, err := di.InjectMockedBackend(context.TODO(), controller)

	if err != nil {
		t.Fatalf("failed to create mocked backend - %v", err)
	}

	testGetUserNotifications(t, testApp.Engine, *testApp.Registry.MockUserRegistry)
	testGetUserConfig(t, testApp.Engine, *testApp.Registry.MockUserRegistry)
	testUpdateUserConfig(t, testApp.Engine, *testApp.Registry.MockUserRegistry)
}

func testGetUserNotifications(t *testing.T, e *gin.Engine, mock mk.MockUserRegistry) {

	testNotifications, err := testutils.MakeTestUserNotifications(3, testUserId)

	if err != nil {
		t.Fatal(err)
	}

	getNotifications := func(filters dto.UserNotificationFilters) *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		req, _ := http.NewRequest(http.MethodGet, userNotificationsUrl, nil)
		req.Header.Add("userId", testUserId)
		testutils.AddUserFilters(req, &filters)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Should be able to retrieve notifications page", func(t *testing.T) {
		mock.
			EXPECT().
			GetUserNotifications(gomock.Any(), gomock.Any()).
			Return(dto.Page[dto.UserNotification]{
				NextToken:   nil,
				PrevToken:   nil,
				ResultCount: len(testNotifications),
				Data:        testNotifications,
			}, nil)

		filters := dto.UserNotificationFilters{}

		w := getNotifications(filters)

		resp := dto.Page[dto.UserNotification]{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, w.Code, http.StatusOK)
		assert.ElementsMatch(t, testNotifications, resp.Data)
	})

	t.Run("Should fail if there are duplicated topics on the filter", func(t *testing.T) {
		topic := "test"

		filters := dto.UserNotificationFilters{
			PageFilter: dto.PageFilter{},
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

func testGetUserConfig(t *testing.T, e *gin.Engine, mock mk.MockUserRegistry) {

	cfg := testutils.MakeTestUserConfig(testUserId)

	getUserConfig := func(user string) *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		req, _ := http.NewRequest(http.MethodGet, userConfigUrl, nil)
		req.Header.Add("userId", user)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Can get the user's configuration", func(t *testing.T) {
		mock.
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

func testUpdateUserConfig(t *testing.T, e *gin.Engine, mock mk.MockUserRegistry) {

	updateUserConfig := func(cfg dto.UserConfig) *httptest.ResponseRecorder {

		marshalled, _ := json.Marshal(cfg)
		reader := bytes.NewReader(marshalled)

		w := httptest.NewRecorder()

		req, _ := http.NewRequest(http.MethodPut, userConfigUrl, reader)
		req.Header.Add("userId", testUserId)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Should be able to update the user config", func(t *testing.T) {
		userConfig := testutils.MakeTestUserConfig(testUserId)
		userConfig.EmailConfig = dto.ChannelConfig{OptIn: false, SnoozeUntil: nil}
		userConfig.SMSConfig = dto.ChannelConfig{OptIn: true, SnoozeUntil: nil}

		mock.
			EXPECT().
			UpdateUserConfig(gomock.Any(), gomock.Any(), gomock.Any()).
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
