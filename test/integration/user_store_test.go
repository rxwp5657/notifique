package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/notifique/controllers"
	"github.com/notifique/dto"
	"github.com/notifique/internal"
	"github.com/notifique/test"
	st "github.com/notifique/test/integration/store"
	"github.com/stretchr/testify/assert"
)

type UserStorageTester interface {
	controllers.UserStorage
	st.ContainerTester
	InsertUserNotifications(ctx context.Context, userId string, un []dto.UserNotification) error
	DeleteUserNotifications(ctx context.Context, userId string, ids []string) error
}

func TestUsersStoragePostgres(t *testing.T) {

	ctx := context.Background()
	tester, close, err := st.NewPostgresIntegrationTester(ctx)

	if err != nil {
		t.Fatal("failed to init postgres tester - ", err)
	}

	defer close()

	testRetrieveUserNotifications(ctx, t, tester)
	testSetReadStatus(ctx, t, tester)
	testUserConfig(ctx, t, tester)
}

func assertEqualUserNotifications(t *testing.T, a, b []dto.UserNotification) {

	assert.Equal(t, len(a), len(b))

	for i, n := range a {
		assert.Equal(t, n.Id, b[i].Id)
		assert.Equal(t, n.Title, b[i].Title)
		assert.Equal(t, n.Topic, b[i].Topic)
		assert.Equal(t, n.Contents, b[i].Contents)
		assert.Equal(t, n.Image, b[i].Image)
	}
}

func testRetrieveUserNotifications(ctx context.Context, t *testing.T, ust UserStorageTester) {
	userId := "1234"
	alternateTopic := "Alternate"

	testNotifications, err := test.CreateTestUserNotifications(5, userId)

	if err != nil {
		t.Fatal(err)
	}

	alternateNotification := &testNotifications[0]
	alternateNotification.Topic = alternateTopic

	err = ust.InsertUserNotifications(ctx, userId, testNotifications)

	if err != nil {
		t.Fatal(err)
	}

	defer st.Clear(ctx, t, ust)

	t.Run("Can retrieve a page of user notifications", func(t *testing.T) {
		filters := dto.UserNotificationFilters{
			UserId: userId,
		}

		page, err := ust.GetUserNotifications(ctx, filters)

		assert.Nil(t, err)
		assert.Nil(t, page.NextToken)
		assert.Nil(t, page.PrevToken)
		assert.Equal(t, page.ResultCount, len(testNotifications))
		assertEqualUserNotifications(t, testNotifications, page.Data)
	})

	t.Run("Can paginate user notifications", func(t *testing.T) {
		maxResults := 1

		userFilters := dto.UserNotificationFilters{
			UserId:     userId,
			PageFilter: dto.PageFilter{MaxResults: &maxResults},
		}

		notifications := make([]dto.UserNotification, 0, len(testNotifications))

		for {
			page, err := ust.GetUserNotifications(ctx, userFilters)

			if err != nil {
				t.Fatal(fmt.Errorf("failed to retrieve user notifications page - %w", err))
			}

			if page.ResultCount == 0 {
				break
			}

			notifications = append(notifications, page.Data...)
			userFilters.NextToken = page.NextToken
		}

		assertEqualUserNotifications(t, testNotifications, notifications)
	})

	t.Run("Can retrieve notifications from a certain topic", func(t *testing.T) {
		filters := dto.UserNotificationFilters{
			UserId: userId,
			Topics: []string{alternateTopic},
		}

		page, err := ust.GetUserNotifications(ctx, filters)

		assert.Nil(t, err)
		assert.Nil(t, page.NextToken)
		assert.Nil(t, page.PrevToken)
		assert.Equal(t, page.ResultCount, 1)
		assert.Equal(t, alternateNotification.Id, page.Data[0].Id)
		assert.Equal(t, alternateNotification.Topic, page.Data[0].Topic)
	})
}

func testSetReadStatus(ctx context.Context, t *testing.T, ust UserStorageTester) {
	userId := "1234"

	testNotifications, err := test.CreateTestUserNotifications(1, userId)
	testNotification := testNotifications[0]

	if err != nil {
		t.Fatal(err)
	}

	err = ust.InsertUserNotifications(ctx, userId, testNotifications)

	if err != nil {
		t.Fatal(err)
	}

	defer st.Clear(ctx, t, ust)

	t.Run("Can set the read at status of a notification", func(t *testing.T) {
		err := ust.SetReadStatus(ctx, userId, testNotification.Id)
		assert.Nil(t, err)

		filters := dto.UserNotificationFilters{
			UserId: userId,
		}

		page, err := ust.GetUserNotifications(ctx, filters)

		if err != nil {
			t.Fatal(err)
		}

		assert.NotNil(t, page.Data[0].ReadAt)
	})

	t.Run("Should return an error if the notification doesn't exists", func(t *testing.T) {
		testId, err := uuid.NewV7()

		if err != nil {
			t.Fatal(err)
		}

		testIdStr := testId.String()
		err = ust.SetReadStatus(ctx, userId, testIdStr)

		assert.ErrorAs(t, err, &internal.NotificationNotFound{
			NotificationId: testIdStr,
		})
	})
}

func testUserConfig(ctx context.Context, t *testing.T, ust UserStorageTester) {

	userId := "1234"

	t.Run("Can retrieve the user config", func(t *testing.T) {

		expectedConfig := dto.UserConfig{
			EmailConfig: dto.ChannelConfig{OptIn: true, SnoozeUntil: nil},
			SMSConfig:   dto.ChannelConfig{OptIn: true, SnoozeUntil: nil},
			InAppConfig: dto.ChannelConfig{OptIn: true, SnoozeUntil: nil},
		}

		cfg, err := ust.GetUserConfig(ctx, userId)

		assert.Nil(t, err)
		assert.Equal(t, expectedConfig, cfg)
	})

	st.Clear(ctx, t, ust)

	t.Run("Can Update the user config", func(t *testing.T) {
		snoozeUntil := time.Now().AddDate(0, 0, 10).Format(time.RFC3339)

		userConfig := dto.UserConfig{
			EmailConfig: dto.ChannelConfig{OptIn: false, SnoozeUntil: nil},
			SMSConfig:   dto.ChannelConfig{OptIn: true, SnoozeUntil: &snoozeUntil},
			InAppConfig: dto.ChannelConfig{OptIn: true, SnoozeUntil: nil},
		}

		err := ust.UpdateUserConfig(ctx, userId, userConfig)

		assert.Nil(t, err)

		cfg, err := ust.GetUserConfig(ctx, userId)

		assert.Nil(t, err)
		assert.Equal(t, userConfig, cfg)
	})

	st.Clear(ctx, t, ust)
}
