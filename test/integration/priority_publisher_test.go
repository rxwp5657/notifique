package integration_test

import (
	"context"
	"testing"

	c "github.com/notifique/controllers"
	di "github.com/notifique/dependency_injection"
	dto "github.com/notifique/dto"
	"github.com/stretchr/testify/assert"
)

func TestRabbitMQPriorityPublisher(t *testing.T) {
	testApp, close, err := di.InjectPgRabbitMQPriorityIntegrationTest(context.TODO())

	if err != nil {
		t.Fatalf("failed to create container app - %v", err)
	}

	defer close()

	testPriorityPublisher(t, testApp.Storage, testApp.Publisher)
}

func TestSQSPriorityPublisher(t *testing.T) {
	testApp, close, err := di.InjectPgSQSPriorityIntegrationTest(context.TODO())

	if err != nil {
		t.Fatalf("failed to create container app - %v", err)
	}

	defer close()

	testPriorityPublisher(t, testApp.Storage, testApp.Publisher)
}

func testPriorityPublisher(t *testing.T, s c.NotificationStorage, p c.NotificationPublisher) {

	userId := "1234"

	testNotificationReq := dto.NotificationReq{
		Title:            "Notification 1",
		Contents:         "Notification Contents 1",
		Topic:            "Testing",
		Priority:         "MEDIUM",
		DistributionList: nil,
		Recipients:       []string{userId},
		Channels:         []string{"in-app", "e-mail"},
	}

	notificationId, err := s.SaveNotification(context.TODO(), userId, testNotificationReq)

	if err != nil {
		t.Fatalf("failed to insert test notification - %v", err)
	}

	testNotification := c.Notification{
		NotificationReq: testNotificationReq,
		Id:              notificationId,
	}

	t.Run("Can publish a new notification", func(t *testing.T) {
		err := p.Publish(context.TODO(), testNotification)
		assert.Nil(t, err)
	})
}
