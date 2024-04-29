package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	c "github.com/notifique/controllers"
	"github.com/notifique/dto"
	"github.com/notifique/routes"
	"github.com/stretchr/testify/assert"
)

const userId = "12345"

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

func TestCreateNotification(t *testing.T) {

	storage := getStorage()
	router := makeNotificationRouter(&storage)

	testNofitication := dto.NotificationReq{
		Title:            "Notification 1",
		Contents:         "Notification Contents 1",
		Topic:            "Testing",
		Priority:         "MEDIUM",
		DistributionList: nil,
		Recipients:       []string{userId},
		Channels:         []string{"in-app", "e-mail"},
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
