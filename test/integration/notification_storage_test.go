package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/notifique/controllers"
	"github.com/notifique/dto"
	"github.com/notifique/test"
	st "github.com/notifique/test/storage"
)

type NotificationStorageTester interface {
	controllers.NotificationStorage
	st.ContainerTester
	GetNotification(ctx context.Context, notificationId string) (dto.NotificationReq, error)
	GetNotificationStatus(ctx context.Context, notificationId string) (string, error)
}

func TestNotificationStoragePostgres(t *testing.T) {

	ctx := context.Background()
	tester, close, err := st.NewPostgresIntegrationTester(ctx)

	if err != nil {
		t.Fatal("failed to init postgres tester - ", err)
	}

	defer close()

	testCreateNotification(ctx, t, tester)
	testUpdateNotificationStatus(ctx, t, tester)
}

func TestNotificationStorageDynamo(t *testing.T) {
	ctx := context.Background()
	tester, close, err := st.NewDynamoStorageTester(ctx)

	if err != nil {
		t.Fatal("failed to init postgres tester - ", err)
	}

	defer close()

	testCreateNotification(ctx, t, tester)
	testUpdateNotificationStatus(ctx, t, tester)
}

func testCreateNotification(ctx context.Context, t *testing.T, nt NotificationStorageTester) {

	userId := "1234"

	testNofiticationReq := test.MakeTestNotificationRequest()

	defer st.Clear(ctx, t, nt)

	t.Run("Should be able to create a new notification", func(t *testing.T) {
		notificationId, err := nt.SaveNotification(ctx, userId, testNofiticationReq)
		assert.Nil(t, err)
		assert.Nil(t, uuid.Validate(notificationId))

		notification, err := nt.GetNotification(ctx, notificationId)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, testNofiticationReq.Title, notification.Title)
		assert.Equal(t, testNofiticationReq.Contents, notification.Contents)
		assert.Equal(t, testNofiticationReq.Topic, notification.Topic)
		assert.Equal(t, testNofiticationReq.Priority, notification.Priority)
		assert.Equal(t, testNofiticationReq.DistributionList, notification.DistributionList)
		assert.ElementsMatch(t, testNofiticationReq.Recipients, notification.Recipients)
		assert.ElementsMatch(t, testNofiticationReq.Channels, notification.Channels)

		status, err := nt.GetNotificationStatus(ctx, notificationId)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, string(controllers.Created), status)
	})
}

func testUpdateNotificationStatus(ctx context.Context, t *testing.T, nt NotificationStorageTester) {

	user := "1234"
	testNofiticationReq := test.MakeTestNotificationRequest()
	notificationId, err := nt.SaveNotification(ctx, user, testNofiticationReq)

	if err != nil {
		t.Fatal(err)
	}

	defer st.Clear(ctx, t, nt)

	t.Run("Should be able to update the notification status", func(t *testing.T) {

		log := controllers.NotificationStatusLog{
			NotificationId: notificationId,
			Status:         controllers.Published,
			ErrorMsg:       nil,
		}

		err := nt.UpdateNotificationStatus(ctx, log)

		assert.Nil(t, err)

		status, err := nt.GetNotificationStatus(ctx, notificationId)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, string(controllers.Published), status)
	})
}
