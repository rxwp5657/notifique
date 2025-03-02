package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/notifique/internal/server"
	"github.com/notifique/internal/server/controllers"
	"github.com/notifique/internal/server/dto"
	"github.com/notifique/internal/testutils"
	r "github.com/notifique/internal/testutils/registry"
)

type NotificationRegistryTester interface {
	controllers.NotificationRegistry
	r.ContainerTester
	GetNotification(ctx context.Context, notificationId string) (dto.NotificationReq, error)
	GetNotificationStatus(ctx context.Context, notificationId string) (*controllers.NotificationStatus, error)
}

func TestNotificationRegistryPostgres(t *testing.T) {

	ctx := context.Background()
	tester, close, err := r.NewPostgresIntegrationTester(ctx)

	if err != nil {
		t.Fatal("failed to init postgres tester - ", err)
	}

	defer close()

	testCreateNotification(ctx, t, tester)
	testUpdateNotificationStatus(ctx, t, tester)
	testDeleteNotification(ctx, t, tester)
}

func TestNotificationRegistryDynamo(t *testing.T) {
	ctx := context.Background()
	tester, close, err := r.NewDynamoRegistryTester(ctx)

	if err != nil {
		t.Fatal("failed to init dynamo tester - ", err)
	}

	defer close()

	testCreateNotification(ctx, t, tester)
	testUpdateNotificationStatus(ctx, t, tester)
}

func testCreateNotification(ctx context.Context, t *testing.T, nt NotificationRegistryTester) {

	userId := "1234"

	testNofiticationReq := testutils.MakeTestNotificationRequest()

	defer r.Clear(ctx, t, nt)

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

		assert.Equal(t, controllers.Created, *status)
	})
}

func testUpdateNotificationStatus(ctx context.Context, t *testing.T, nt NotificationRegistryTester) {

	user := "1234"
	testNofiticationReq := testutils.MakeTestNotificationRequest()
	notificationId, err := nt.SaveNotification(ctx, user, testNofiticationReq)

	if err != nil {
		t.Fatal(err)
	}

	defer r.Clear(ctx, t, nt)

	t.Run("Should be able to update the notification status", func(t *testing.T) {

		log := controllers.NotificationStatusLog{
			NotificationId: notificationId,
			Status:         controllers.Queued,
			ErrorMsg:       nil,
		}

		err := nt.UpdateNotificationStatus(ctx, log)

		assert.Nil(t, err)

		status, err := nt.GetNotificationStatus(ctx, notificationId)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, controllers.Queued, *status)
	})
}

func testDeleteNotification(ctx context.Context, t *testing.T, nt NotificationRegistryTester) {

	user := "1234"

	testNotifications := map[controllers.NotificationStatus]string{
		controllers.Created: "",
		controllers.Queued:  "",
		controllers.Sending: "",
		controllers.Sent:    "",
		controllers.Failed:  "",
	}

	for status := range testNotifications {
		testNofiticationReq := testutils.MakeTestNotificationRequest()
		notificationId, err := nt.SaveNotification(ctx, user, testNofiticationReq)

		if err != nil {
			t.Fatal(err)
		}

		err = nt.UpdateNotificationStatus(ctx, controllers.NotificationStatusLog{
			NotificationId: notificationId,
			Status:         status,
		})

		if err != nil {
			t.Fatal(err)
		}

		testNotifications[status] = notificationId
	}

	defer r.Clear(ctx, t, nt)

	tests := []struct {
		name           string
		notificationId string
		expectedError  error
	}{
		{
			name:           "Can delete a notification in CREATED state",
			notificationId: testNotifications[controllers.Created],
			expectedError:  nil,
		},
		{
			name:           "Can delete a notification in FAILED state",
			notificationId: testNotifications[controllers.Failed],
			expectedError:  nil,
		},
		{
			name:           "Can delete a notification in SENT state",
			notificationId: testNotifications[controllers.Sent],
			expectedError:  nil,
		},
		{
			name:           "Should fail deleting a notification on QUEUED state",
			notificationId: testNotifications[controllers.Queued],
			expectedError: server.InvalidNotificationStatus{
				Id:     testNotifications[controllers.Queued],
				Status: string(controllers.Queued),
			},
		},
		{
			name:           "Should fail deleting a notification on SENDING state",
			notificationId: testNotifications[controllers.Sending],
			expectedError: server.InvalidNotificationStatus{
				Id:     testNotifications[controllers.Sending],
				Status: string(controllers.Sending),
			},
		},
		{
			name:           "Should be able to delete a notification that doesn't exist",
			notificationId: uuid.NewString(),
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualErr := nt.DeleteNotification(ctx, tt.notificationId)
			assert.Equal(t, tt.expectedError, actualErr)

			if tt.expectedError == nil {
				status, err := nt.GetNotificationStatus(ctx, tt.notificationId)

				if err != nil {
					t.Fatal(err)
				}

				assert.Nil(t, status)
			}
		})
	}
}
