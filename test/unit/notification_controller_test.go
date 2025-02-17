package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	di "github.com/notifique/internal/di"
	"github.com/notifique/internal/server/dto"
	"github.com/notifique/internal/testutils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const notificationsUrl = "/notifications"

func TestNotificationController(t *testing.T) {

	controller := gomock.NewController(t)
	defer controller.Finish()

	testApp, err := di.InjectMockedBackend(context.TODO(), controller)

	if err != nil {
		t.Fatalf("failed to create mocked backend - %v", err)
	}

	testCreateNotification(t, testApp.Engine, *testApp)
}

func testCreateNotification(t *testing.T, e *gin.Engine, mock di.MockedBackend) {

	mock.
		Publisher.
		EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Return(nil)

	registryMock := mock.Registry.MockNotificationRegistry

	userId := "1234"

	createNotification := func(notificationReq dto.NotificationReq) *httptest.ResponseRecorder {
		body, _ := json.Marshal(notificationReq)
		reader := bytes.NewReader(body)

		w := httptest.NewRecorder()

		req, _ := http.NewRequest(http.MethodPost, notificationsUrl, reader)
		req.Header.Add("userId", userId)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Can create new notifications", func(t *testing.T) {
		registryMock.
			EXPECT().
			SaveNotification(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(uuid.NewString(), nil)

		req := testutils.MakeTestNotificationRequest()

		w := createNotification(req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Should fail if the channel is not supported", func(t *testing.T) {
		req := testutils.MakeTestNotificationRequest()
		req.Channels = append(req.Channels, "INVALID")

		w := createNotification(req)

		expectedMsg := `NotificationReq.Channels[2]' Error:Field validation for 'Channels[2]' failed on the 'oneof' tag`

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if the title exceeds the limits", func(t *testing.T) {
		req := testutils.MakeTestNotificationRequest()
		req.Title = testutils.MakeStrWithSize(121)

		w := createNotification(req)

		expectedMsg := `Key: 'NotificationReq.Title' Error:Field validation for 'Title' failed on the 'max' tag`

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if the title is empty", func(t *testing.T) {
		req := testutils.MakeTestNotificationRequest()
		req.Title = ""

		w := createNotification(req)

		expectedMsg := `Key: 'NotificationReq.Title' Error:Field validation for 'Title' failed on the 'required' tag`

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if the contents are empty", func(t *testing.T) {
		req := testutils.MakeTestNotificationRequest()
		req.Contents = ""

		w := createNotification(req)

		expectedMsg := `Key: 'NotificationReq.Contents' Error:Field validation for 'Contents' failed on the 'required' tag`

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if it the contents exceeds the limits", func(t *testing.T) {
		req := testutils.MakeTestNotificationRequest()
		req.Contents = testutils.MakeStrWithSize(1025)

		w := createNotification(req)

		expectedMsg := `Key: 'NotificationReq.Contents' Error:Field validation for 'Contents' failed on the 'max' tag`

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if the topic exceeds the limits", func(t *testing.T) {
		req := testutils.MakeTestNotificationRequest()
		req.Topic = testutils.MakeStrWithSize(121)

		w := createNotification(req)

		expectedMsg := `Key: 'NotificationReq.Topic' Error:Field validation for 'Topic' failed on the 'max' tag`

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if the topic is empty", func(t *testing.T) {
		req := testutils.MakeTestNotificationRequest()
		req.Topic = ""

		w := createNotification(req)

		expectedMsg := `Key: 'NotificationReq.Topic' Error:Field validation for 'Topic' failed on the 'required' tag`

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if there are duplicated recipients", func(t *testing.T) {
		req := testutils.MakeTestNotificationRequest()
		req.Recipients = append(req.Recipients, req.Recipients...)

		w := createNotification(req)

		expectedMsg := `Key: 'NotificationReq.Recipients' Error:Field validation for 'Recipients' failed on the 'unique' tag`

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if the priority is not supported", func(t *testing.T) {
		req := testutils.MakeTestNotificationRequest()
		req.Priority = "Test"

		w := createNotification(req)

		expectedMsg := `Key: 'NotificationReq.Priority' Error:Field validation for 'Priority' failed on the 'oneof' tag`

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if the image url is not an url", func(t *testing.T) {
		url := "not an url"
		req := testutils.MakeTestNotificationRequest()
		req.Image = &url

		w := createNotification(req)

		expectedMsg := `Key: 'NotificationReq.Image' Error:Field validation for 'Image' failed on the 'uri' tag`

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})
}
