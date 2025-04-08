package integration_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/notifique/shared/dto"
	"github.com/notifique/worker/internal/di"
	"github.com/notifique/worker/internal/worker"
	"github.com/stretchr/testify/assert"
)

type ConsumerIntegrationTest interface {
	worker.QueueConsumer
	Publish(ctx context.Context, n dto.NotificationMsgPayload) error
	Start(ctx context.Context)
}

type MessageIdFunc func(payloadIdx int, msg dto.NotificationMsgPayload) string

func TestRabbitMQConsumer(t *testing.T) {

	notificationChan := make(chan dto.NotificationMsg)

	consumer, close, err := di.InjectRabbitMQConsumerIntegrationTest(context.TODO(), notificationChan)

	if err != nil {
		t.Fatalf("failed to inject RabbitMQ consumer: %v", err)
	}

	defer close()

	getMessageId := func(payloadIdx int, _ dto.NotificationMsgPayload) string {
		return strconv.Itoa(payloadIdx + 1)
	}

	testConsumer(t, consumer, notificationChan, getMessageId)
}

func TestSQSConsumer(t *testing.T) {
	notificationChan := make(chan dto.NotificationMsg)

	consumer, close, err := di.InjectSQSConsumerIntegrationTest(context.TODO(), notificationChan)

	if err != nil {
		t.Fatalf("failed to inject RabbitMQ consumer: %v", err)
	}

	defer close()

	getMessageId := func(_ int, p dto.NotificationMsgPayload) string {
		return p.Hash
	}

	testConsumer(t, consumer, notificationChan, getMessageId)
}

func testConsumer(t *testing.T, consumer ConsumerIntegrationTest, notificationChan <-chan dto.NotificationMsg, mf MessageIdFunc) {

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute*1)
	defer cancel()

	payloads := []dto.NotificationMsgPayload{{
		NotificationReq: dto.NotificationReq{
			RawContents: &dto.RawContents{
				Title:    "First Test Notification",
				Contents: "This is the first test notification",
			},
			Topic:      "test-topic-1",
			Priority:   dto.High,
			Recipients: []string{"user1"},
			Channels:   []dto.NotificationChannel{dto.Email},
		},
		Id:   "test-id-1",
		Hash: "1",
	}, {
		NotificationReq: dto.NotificationReq{
			RawContents: &dto.RawContents{
				Title:    "Second Test Notification",
				Contents: "This is the second test notification",
			},
			Topic:      "test-topic-2",
			Priority:   dto.Low,
			Recipients: []string{"user2"},
			Channels:   []dto.NotificationChannel{dto.InApp},
		},
		Id:   "test-id-2",
		Hash: "2",
	}}

	for _, p := range payloads {
		err := consumer.Publish(ctx, p)
		if err != nil {
			t.Fatalf("failed to publish message: %v", err)
		}
	}

	t.Run("Can consume messages", func(t *testing.T) {
		go consumer.Start(ctx)

		receivedPayloads := make([]dto.NotificationMsgPayload, 0, len(payloads))

		for range len(payloads) {
			select {
			case msg := <-notificationChan:
				receivedPayloads = append(receivedPayloads, msg.Payload)
				assert.Nil(t, consumer.Ack(ctx, msg.DeleteTag))
			case <-ctx.Done():
				t.Fatal("context done before receiving message")
			}
		}

		assert.ElementsMatch(t, payloads, receivedPayloads)
	})
}
