package testutils

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/notifique/internal/server/dto"
)

func MakeRecipients(numRecipients int) []string {

	recipients := make([]string, 0, numRecipients)

	for i := range cap(recipients) {
		recipients = append(recipients, strconv.Itoa(i))
	}

	return recipients
}

func MakeDistributionLists(numLists int) []dto.DistributionList {

	lists := make([]dto.DistributionList, 0, numLists)

	for i := range numLists {
		list := dto.DistributionList{
			Name:       fmt.Sprintf("Test List %d", i),
			Recipients: MakeRecipients(rand.Intn(10)),
		}

		lists = append(lists, list)
	}

	return lists
}

func MakeSummaries(lists []dto.DistributionList) []dto.DistributionListSummary {
	summaries := make([]dto.DistributionListSummary, 0, len(lists))

	for _, l := range lists {
		summary := dto.DistributionListSummary{
			Name:               l.Name,
			NumberOfRecipients: len(l.Recipients),
		}

		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	return summaries
}

func MakeTestNotificationRequest() dto.NotificationReq {
	testNofiticationReq := dto.NotificationReq{
		Title:            "Notification 1",
		Contents:         "Notification Contents 1",
		Topic:            "Testing",
		Priority:         "MEDIUM",
		DistributionList: nil,
		Recipients:       []string{"1234"},
		Channels:         []string{"in-app", "e-mail"},
	}

	return testNofiticationReq
}

func MakeStrWithSize(size int) string {
	str, i := "", 0

	for i < size {
		str += "+"
		i += 1
	}

	return str
}

func MakeTestUserNotifications(numNotifications int, userId string) ([]dto.UserNotification, error) {
	testNotifications := make([]dto.UserNotification, 0, numNotifications)

	for i := range numNotifications {
		id, err := uuid.NewV7()

		if err != nil {
			return testNotifications, err
		}

		notification := dto.UserNotification{
			Id:        id.String(),
			Title:     fmt.Sprintf("Test title %d", i),
			Contents:  fmt.Sprintf("Test contents %d", i),
			Topic:     "Testing",
			CreatedAt: time.Now().Format(time.RFC3339Nano),
		}

		testNotifications = append(testNotifications, notification)
	}

	sort.Slice(testNotifications, func(i, j int) bool {
		return testNotifications[i].Id > testNotifications[j].Id
	})

	return testNotifications, nil
}

func MakeTestUserConfig(userId string) dto.UserConfig {
	cfg := dto.UserConfig{
		EmailConfig: dto.ChannelConfig{OptIn: true, SnoozeUntil: nil},
		SMSConfig:   dto.ChannelConfig{OptIn: true, SnoozeUntil: nil},
		InAppConfig: dto.ChannelConfig{OptIn: true, SnoozeUntil: nil},
	}

	return cfg
}
