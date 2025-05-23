package integration_test

import (
	"context"
	"encoding/json"
	"testing"

	c "github.com/notifique/service/internal/controllers"
	di "github.com/notifique/service/internal/di"
	"github.com/notifique/shared/dto"
	"github.com/notifique/shared/hash"
	"github.com/stretchr/testify/assert"
)

func TestRabbitMQPriorityPublisher(t *testing.T) {
	testApp, close, err := di.InjectPgRabbitMQPriorityIntegrationTest(context.TODO())

	if err != nil {
		t.Fatalf("failed to create container app - %v", err)
	}

	defer close()

	testPriorityPublisher(t, testApp.Registry, testApp.Publisher)
}

func TestSQSPriorityPublisher(t *testing.T) {
	testApp, close, err := di.InjectPgSQSPriorityIntegrationTest(context.TODO())

	if err != nil {
		t.Fatalf("failed to create container app - %v", err)
	}

	defer close()

	testPriorityPublisher(t, testApp.Registry, testApp.Publisher)
}

func testPriorityPublisher(t *testing.T, s c.NotificationRegistry, p c.NotificationPublisher) {

	userId := "1234"

	rawContents := &dto.RawContents{
		Title:    "Notification 1",
		Contents: "Notification Contents 1",
	}

	testNotificationReq := dto.NotificationReq{
		RawContents:      rawContents,
		Topic:            "Testing",
		Priority:         "MEDIUM",
		DistributionList: nil,
		Recipients:       []string{userId},
		Channels:         []dto.NotificationChannel{"in-app", "e-mail"},
	}

	notificationId, err := s.SaveNotification(context.TODO(), userId, testNotificationReq)

	if err != nil {
		t.Fatalf("failed to insert test notification - %v", err)
	}

	marsahlled, err := json.Marshal(testNotificationReq)

	if err != nil {
		t.Fatalf("failed to marshal test notification - %v", err)
	}

	hash := hash.GetMd5Hash(string(marsahlled))

	testNotification := dto.NotificationMsgPayload{
		NotificationReq: testNotificationReq,
		Id:              notificationId,
		Hash:            hash,
	}

	t.Run("Can publish a new notification", func(t *testing.T) {
		err := p.Publish(context.TODO(), testNotification)
		assert.Nil(t, err)
	})
}
