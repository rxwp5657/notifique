package unit_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/notifique/shared/dto"
	"github.com/notifique/worker/internal/clients"
	"github.com/notifique/worker/internal/sender"
	"github.com/stretchr/testify/assert"
)

func TestNotificationServiceSender(t *testing.T) {

	setupTestServer := func(handler func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *sender.NotificationServiceSender) {
		server := httptest.NewServer(http.HandlerFunc(handler))

		senderClient := sender.NewNotificationServiceSender(clients.NotificationServiceClient{
			AuthProvider:           clients.NoAuth,
			NotificationServiceUrl: clients.NotificationServiceUrl(server.URL),
			NumRetries:             1,
			BaseDelay:              clients.BaseDelay(1 * time.Second),
			MaxDelay:               clients.MaxDelay(5 * time.Second),
		})

		return server, senderClient
	}

	t.Run("Can send notifications successfully", func(t *testing.T) {
		server, notificationSender := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/users/notifications", r.URL.Path)
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			batch := []dto.UserNotificationReq{}
			err := json.NewDecoder(r.Body).Decode(&batch)
			assert.NoError(t, err)

			w.WriteHeader(http.StatusOK)
		})

		defer server.Close()

		batch := []dto.UserNotificationReq{{
			UserId:   "user1",
			Title:    "Test Notification",
			Contents: "This is a test notification",
			Topic:    "test-topic",
			Image:    nil,
		}, {
			UserId:   "user2",
			Title:    "Test Notification",
			Contents: "This is a test notification",
			Topic:    "test-topic",
			Image:    nil,
		}}

		err := notificationSender.SendNotifications(context.Background(), batch)
		assert.NoError(t, err)
	})

	t.Run("Returns error when server returns non-200", func(t *testing.T) {
		server, notificationSender := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		defer server.Close()

		batch := []dto.UserNotificationReq{{
			UserId:   "user1",
			Title:    "Test Notification",
			Contents: "This is a test notification",
			Topic:    "test-topic",
			Image:    nil,
		}, {
			UserId:   "user2",
			Title:    "Test Notification",
			Contents: "This is a test notification",
			Topic:    "test-topic",
			Image:    nil,
		}}

		err := notificationSender.SendNotifications(context.Background(), batch)
		assert.Error(t, err)
	})

	t.Run("Handles rate limiting", func(t *testing.T) {
		attempts := 0
		server, notificationSender := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
			if attempts == 0 {
				w.Header().Set("X-RateLimit-Reset", "100")
				w.WriteHeader(http.StatusTooManyRequests)
				attempts++
			} else {
				w.WriteHeader(http.StatusOK)
			}
		})

		defer server.Close()

		batch := []dto.UserNotificationReq{{
			UserId:   "user1",
			Title:    "Test Notification",
			Contents: "This is a test notification",
			Topic:    "test-topic",
			Image:    nil,
		}, {
			UserId:   "user2",
			Title:    "Test Notification",
			Contents: "This is a test notification",
			Topic:    "test-topic",
			Image:    nil,
		}}

		err := notificationSender.SendNotifications(context.Background(), batch)
		assert.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})
}
