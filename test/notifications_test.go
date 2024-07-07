package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	di "github.com/notifique/dependency_injection"
	"github.com/notifique/dto"
	"github.com/stretchr/testify/assert"
)

func TestNotificationsController(t *testing.T) {

	notificationsUrl := "/notifications"

	controller := gomock.NewController(t)
	defer controller.Finish()

	testApp, close, err := di.InjectPgMockedPubIntegrationTest(context.TODO(), controller)

	if err != nil {
		t.Fatalf("failed to create container app - %v", err)
	}

	defer close()

	// Setup Mock
	testApp.Publisher.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)

	userId := "1234"

	testNofitication := dto.NotificationReq{
		Title:            "Notification 1",
		Contents:         "Notification Contents 1",
		Topic:            "Testing",
		Priority:         "MEDIUM",
		DistributionList: nil,
		Recipients:       []string{userId},
		Channels:         []string{"in-app", "e-mail"},
	}

	createNotification := func(notification dto.NotificationReq) *httptest.ResponseRecorder {
		body, _ := json.Marshal(notification)
		reader := bytes.NewReader(body)

		w := httptest.NewRecorder()

		req, _ := http.NewRequest("POST", notificationsUrl, reader)
		req.Header.Add("userId", userId)

		testApp.Engine.ServeHTTP(w, req)

		return w
	}

	t.Run("Can create new notifications", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		w := createNotification(notification)

		assert.Equal(t, 204, w.Code)
	})

	t.Run("Should fail on bad channel", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		notification.Channels = append(notification.Channels, "Bad Channel")

		w := createNotification(notification)

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.FailNow()
		}

		errMsg := "Error:Field validation for 'Channels[2]'"
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], errMsg)
	})

	t.Run("Should fail on long title", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		notification.Title = makeStrWithSize(200)

		w := createNotification(notification)

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.FailNow()
		}

		errMsg := "Error:Field validation for 'Title'"
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], errMsg)
	})

	t.Run("Should fail on long contents", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		notification.Contents = makeStrWithSize(1025)

		w := createNotification(notification)

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.FailNow()
		}

		errMsg := "Error:Field validation for 'Contents'"
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], errMsg)
	})

	t.Run("Should fail on long topic", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		notification.Topic = makeStrWithSize(200)

		w := createNotification(notification)

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.FailNow()
		}

		errMsg := "Error:Field validation for 'Topic'"
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], errMsg)
	})

	t.Run("Should fail on duplicated recipients", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		notification.Recipients = append(notification.Recipients, userId)

		w := createNotification(notification)

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.FailNow()
		}

		errMsg := "Error:Field validation for 'Recipients'"
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], errMsg)
	})

	t.Run("Should fail on invalid priority", func(t *testing.T) {
		notification := copyNotification(testNofitication)
		notification.Priority = "Bad Priority"

		w := createNotification(notification)

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.FailNow()
		}

		errMsg := "Error:Field validation for 'Priority'"
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], errMsg)
	})
}
