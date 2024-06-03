package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/notifique/controllers"
	"github.com/notifique/dto"
	"github.com/notifique/internal/publisher"
	storage "github.com/notifique/internal/storage/dynamodb"
	pstorage "github.com/notifique/internal/storage/postgres"
	"github.com/notifique/routes"
	"github.com/stretchr/testify/assert"
)

type publisherType string

const (
	SQS publisherType = "SQS"
)

type NotificationStorage interface {
	controllers.NotificationStorage
	controllers.DistributionListStorage
}

func makeNotificationStorage(t storageType, uri string) (NotificationStorage, error) {
	switch t {
	case DYNAMODB:
		client, err := storage.MakeDynamoDBClient(&uri)
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

func TestNotificationsController(t *testing.T) {

	dbContainer, dbCloser, err := setupContainer(POSTGRES)

	if err != nil {
		t.Fatalf("failed to create db container - %s", err)
	}

	sqsContainer, sqsCloser, err := setupLocalStack(context.TODO())

	if err != nil {
		t.Fatalf("failed to create sqs container - %s", err)
	}

	uri := dbContainer.GetURI()
	storage, err := makeNotificationStorage(POSTGRES, uri)

	if err != nil {
		t.Fatalf("failed to create storage - %s", err)
	}

	sqsClient, err := publisher.MakeSQSClient(&sqsContainer.URI)

	if err != nil {
		t.Fatalf("failed to create sqs client - %s", err)
	}

	sqsPublisher := publisher.MakeSQSPublisher(sqsClient, sqsContainer.SQSUrls)

	router := gin.Default()
	routes.SetupNotificationRoutes(router, storage, &sqsPublisher)

	// Needed so we can apply the distribution list name validation
	routes.SetupDistributionListRoutes(router, storage)

	userId := "1234"

	defer func() {
		if err := dbCloser(); err != nil {
			t.Fatalf("failed to terminate db container")
		}

		if err := sqsCloser(); err != nil {
			t.Fatalf("failed to terminate sqs container")
		}
	}()

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

		req, _ := http.NewRequest("POST", "/v0/notifications", reader)
		req.Header.Add("userId", userId)

		router.ServeHTTP(w, req)

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
