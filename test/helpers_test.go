package test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/notifique/dto"
)

type UserNotificationsTester interface {
	CreateUserNotification(ctx context.Context, userId string, un dto.UserNotification) error
	DeleteUserNotification(ctx context.Context, userId string, un dto.UserNotification) error
}

func reverse[T any](data []T) []T {
	dataLen := len(data)
	reversed := make([]T, 0, dataLen)

	for i := range len(data) {
		reversed = append(reversed, data[dataLen-i-1])
	}

	return reversed
}

func createTestUserNotifications(numNotifications int, userId string, tester UserNotificationsTester) ([]dto.UserNotification, error) {
	testNotifications := make([]dto.UserNotification, 0, numNotifications)

	for i := range numNotifications {
		notification := dto.UserNotification{
			Id:        uuid.NewString(),
			Title:     fmt.Sprintf("Test title %d", i),
			Contents:  fmt.Sprintf("Test contents %d", i),
			Topic:     "Testing",
			CreatedAt: time.Now().Format(time.RFC3339Nano),
		}

		err := tester.CreateUserNotification(context.TODO(), userId, notification)

		if err != nil {
			e := fmt.Errorf("failed to create user notification - %w", err)
			return []dto.UserNotification{}, e
		}

		testNotifications = append(testNotifications, notification)
	}

	return testNotifications, nil
}

func deleteTestUserNotifications(userId string, notifications []dto.UserNotification, tester UserNotificationsTester) error {

	for _, n := range notifications {
		err := tester.DeleteUserNotification(context.TODO(), userId, n)

		if err != nil {
			return fmt.Errorf("failed to delete the user notification - %w", err)
		}
	}

	return nil
}

func addPaginationFilters(req *http.Request, filters *dto.PageFilter) {

	if req == nil || filters == nil {
		return
	}

	q := req.URL.Query()

	if filters.NextToken != nil {
		q.Add("nextToken", fmt.Sprint(*filters.NextToken))
	}

	if filters.MaxResults != nil {
		q.Add("maxResults", fmt.Sprint(*filters.MaxResults))
	}

	req.URL.RawQuery = q.Encode()
}
