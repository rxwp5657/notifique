package test

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/notifique/controllers"
	"github.com/notifique/dto"
)

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
			Id:       uuid.NewString(),
			Title:    fmt.Sprintf("Test title %d", i),
			Contents: fmt.Sprintf("Test contents %d", i),
			Topic:    "Testing",
			// Reduce milisec resolution to match postgre's resolution.
			CreatedAt: time.Now().Format("2006-01-02T15:04:05.999Z07:00"),
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

func copyNotification(notification dto.NotificationReq) dto.NotificationReq {
	cp := notification
	cp.Recipients = make([]string, len(notification.Recipients))
	cp.Channels = make([]string, len(notification.Channels))

	copy(cp.Recipients, notification.Recipients)
	copy(cp.Channels, notification.Channels)

	return cp
}

func makeStrWithSize(size int) string {
	field, i := "", 0

	for i < size {
		field += "+"
		i += 1
	}

	return field
}

func makeRecipientsList(numRecipients int) []string {
	recipients := make([]string, 0, numRecipients)

	for i := range numRecipients {
		recipients = append(recipients, fmt.Sprint(i))
	}

	return recipients
}

func crateTestDistributionLists(numLists int, storage controllers.DistributionListStorage) ([]dto.DistributionList, error) {
	lists := make([]dto.DistributionList, 0, numLists)

	for i := range numLists {
		list := dto.DistributionList{
			Name:       fmt.Sprintf("Test List %d", i),
			Recipients: makeRecipientsList(rand.Intn(10)),
		}

		err := storage.CreateDistributionList(context.TODO(), list)

		if err != nil {
			return []dto.DistributionList{}, err
		}

		lists = append(lists, list)
	}

	return lists, nil
}

func makeSummaries(lists []dto.DistributionList) []dto.DistributionListSummary {
	summaries := make([]dto.DistributionListSummary, 0, len(lists))

	for _, l := range lists {
		summary := dto.DistributionListSummary{
			Name:               l.Name,
			NumberOfRecipients: len(l.Recipients),
		}

		summaries = append(summaries, summary)
	}

	return summaries
}

func deleteDistributionLists(lists []dto.DistributionList, storage controllers.DistributionListStorage) error {

	for _, l := range lists {
		err := storage.DeleteDistributionList(context.TODO(), l.Name)

		if err != nil {
			return err
		}
	}

	return nil
}
