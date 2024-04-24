package internal

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/notifique/dto"
)

type notification struct {
	dto.NotificationReq
	Id        string
	CreatedAt time.Time
	CreatedBy string
}

type distributionList struct {
	Name       string
	Recipients map[string]struct{}
}

type InMemoryStorage struct {
	notifications     map[string]notification
	userNotifications map[string][]dto.UserNotification
	usersConfig       map[string][]dto.ChannelConfig
	distributionLists []distributionList
}

func (s *InMemoryStorage) SaveNotification(ctx context.Context, createdBy string, notificationReq dto.NotificationReq) (string, error) {

	id := uuid.NewString()
	createdAt := time.Now()

	n := notification{notificationReq, id, createdAt, createdBy}

	s.notifications[id] = n

	hasInAppChannel := false

	for _, channel := range notificationReq.Channels {
		if channel == "in-app" {
			hasInAppChannel = true
			break
		}
	}

	if !hasInAppChannel {
		return id, nil
	}

	userNotification := dto.UserNotification{
		Id:        id,
		Title:     notificationReq.Title,
		Contents:  notificationReq.Contents,
		CreatedAt: createdAt.String(),
		ReadAt:    nil,
		Topic:     notificationReq.Topic,
	}

	for _, recipient := range notificationReq.Recipients {
		s.userNotifications[recipient] = append(s.userNotifications[recipient], userNotification)
	}

	return id, nil
}

func (s *InMemoryStorage) GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) ([]dto.UserNotification, error) {
	userNotifications := make([]dto.UserNotification, 0)

	topicsSet := make(map[string]struct{})

	for _, topic := range filters.Topics {
		topicsSet[topic] = struct{}{}
	}

	for _, n := range s.userNotifications[filters.UserId] {
		if _, ok := topicsSet[n.Topic]; ok || len(topicsSet) == 0 {
			userNotifications = append(userNotifications, n)
		}
	}

	if filters.Skip != nil {
		userNotifications = userNotifications[*filters.Skip:]
	}

	if filters.Take != nil {
		take := min(*filters.Take, len(userNotifications))
		userNotifications = userNotifications[:take]
	}

	return userNotifications, nil
}

func makeNewUserConfig() []dto.ChannelConfig {

	channels := []string{"e-mail", "sms", "in-app"}

	newUserConfig := make([]dto.ChannelConfig, 0, len(channels))

	for _, channel := range channels {
		config := dto.ChannelConfig{Channel: channel, OptIn: true}
		newUserConfig = append(newUserConfig, config)
	}

	return newUserConfig
}

func (s *InMemoryStorage) GetUserConfig(ctx context.Context, userId string) ([]dto.ChannelConfig, error) {
	userConfig, ok := s.usersConfig[userId]

	if !ok {
		newUserConfig := makeNewUserConfig()
		s.usersConfig[userId] = newUserConfig
		userConfig = newUserConfig
	}

	return userConfig, nil
}

func (s *InMemoryStorage) UpdateUserConfig(ctx context.Context, userId string, config dto.UserConfig) error {

	if _, ok := s.usersConfig[userId]; !ok {
		newUserConfig := makeNewUserConfig()
		s.usersConfig[userId] = newUserConfig
	}

	s.usersConfig[userId] = config.Config

	return nil
}

func (s *InMemoryStorage) SetReadStatus(ctx context.Context, userId, notificationId string) error {

	var userNotification *dto.UserNotification

	for _, n := range s.userNotifications[userId] {
		if n.Id == notificationId {
			userNotification = &n
			break
		}
	}

	if userNotification == nil {
		return NotificationNotFound{notificationId}
	} else {
		readAt := time.Now().String()
		userNotification.ReadAt = &readAt
	}

	return nil
}

func (s *InMemoryStorage) CreateDistributionList(ctx context.Context, newDL dto.DistributionList) error {

	dlName := newDL.Name

	for _, dl := range s.distributionLists {
		if dl.Name == dlName {
			return DistributionListAlreadyExists{dlName}
		}
	}

	dl := distributionList{}
	dl.Name = newDL.Name
	dl.Recipients = make(map[string]struct{}, 0)

	s.distributionLists = append(s.distributionLists, dl)

	for _, recipient := range newDL.Recipients {
		dl.Recipients[recipient] = struct{}{}
	}

	sort.Slice(s.distributionLists, func(i, j int) bool {
		return s.distributionLists[i].Name < s.distributionLists[j].Name
	})

	return nil
}

func (s *InMemoryStorage) GetDistributionLists(ctx context.Context, filter dto.PageFilter) ([]dto.DistributionListSummary, error) {

	skip := 0

	if filter.Skip != nil {
		skip = *filter.Skip
	}

	lists := s.distributionLists[skip:]
	take := min(len(lists), 50)

	if filter.Take != nil {
		take = min(len(lists), *filter.Take)
	}

	summaries := make([]dto.DistributionListSummary, 0, take)

	for _, dl := range lists[:take] {

		summary := dto.DistributionListSummary{
			Name:               dl.Name,
			NumberOfRecipients: len(dl.Recipients),
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func (s *InMemoryStorage) GetRecipients(ctx context.Context, distlistName string, filter dto.PageFilter) ([]string, error) {

	var dl *distributionList = nil

	for _, list := range s.distributionLists {
		if list.Name == distlistName {
			dl = &list
			break
		}
	}

	if dl == nil {
		return make([]string, 0), DistributionListNotFound{distlistName}
	}

	recipients := make([]string, 0, len(dl.Recipients))

	for recipient := range dl.Recipients {
		recipients = append(recipients, recipient)
	}

	sort.Slice(recipients, func(i int, j int) bool {
		return recipients[i] < recipients[j]
	})

	skip := 0

	if filter.Skip != nil {
		skip = min(len(recipients), *filter.Skip)
	}

	recipients = recipients[skip:]

	take := min(len(recipients), 50)

	if filter.Take != nil {
		take = min(len(recipients), *filter.Take)
	}

	recipients = recipients[:take]

	return recipients, nil
}

func (s *InMemoryStorage) AddRecipients(ctx context.Context, distlistName string, recipients []string) (dto.DistributionListSummary, error) {
	var dl *distributionList

	for _, dlist := range s.distributionLists {
		if dlist.Name == distlistName {
			dl = &dlist
			break
		}
	}

	if dl == nil {
		err := DistributionListNotFound{distlistName}
		return dto.DistributionListSummary{}, err
	}

	for _, recipient := range recipients {
		dl.Recipients[recipient] = struct{}{}
	}

	summary := dto.DistributionListSummary{
		Name:               dl.Name,
		NumberOfRecipients: len(dl.Recipients),
	}

	return summary, nil
}

func (s *InMemoryStorage) DeleteRecipients(ctx context.Context, distlistName string, recipients []string) (dto.DistributionListSummary, error) {
	var dl *distributionList

	for _, dlist := range s.distributionLists {
		if dlist.Name == distlistName {
			dl = &dlist
			break
		}
	}

	if dl == nil {
		err := DistributionListNotFound{distlistName}
		return dto.DistributionListSummary{}, err
	}

	for _, recipient := range recipients {
		delete(dl.Recipients, recipient)
	}

	fmt.Println(dl)

	summary := dto.DistributionListSummary{
		Name:               dl.Name,
		NumberOfRecipients: len(dl.Recipients),
	}

	return summary, nil
}

func MakeInMemoryStorage() InMemoryStorage {
	storage := InMemoryStorage{}

	storage.notifications = make(map[string]notification)
	storage.userNotifications = make(map[string][]dto.UserNotification)
	storage.usersConfig = make(map[string][]dto.ChannelConfig)
	storage.distributionLists = make([]distributionList, 0)

	return storage
}
