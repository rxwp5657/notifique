package storage

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/notifique/dto"
	e "github.com/notifique/internal"
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

type getId[T any] func(d T) string

func (s *InMemoryStorage) getDistributionList(name string) *distributionList {

	var dl *distributionList = nil

	for _, list := range s.distributionLists {
		if list.Name == name {
			dl = &list
			break
		}
	}

	return dl
}

func (s *InMemoryStorage) SaveNotification(ctx context.Context, createdBy string, notificationReq dto.NotificationReq) (string, error) {

	if notificationReq.DistributionList != nil {
		dlName := *notificationReq.DistributionList
		dl := s.getDistributionList(dlName)
		if dl == nil {
			return "", e.DistributionListNotFound{Name: dlName}
		}
	}

	id := uuid.NewString()
	createdAt := time.Now()

	n := notification{notificationReq, id, createdAt, createdBy}

	s.notifications[id] = n

	return id, nil
}

func (s *InMemoryStorage) CreateUserNotification(ctx context.Context, userId string, notification dto.UserNotificationReq) (string, error) {

	id := uuid.NewString()

	un := dto.UserNotification{
		Id:        id,
		Title:     notification.Title,
		Contents:  notification.Contents,
		CreatedAt: time.Now().String(),
		Image:     nil,
		ReadAt:    nil,
		Topic:     notification.Topic,
	}

	s.userNotifications[userId] = append(s.userNotifications[userId], un)

	return id, nil
}

func makePage[T any](filters dto.PageFilter, getIdFn getId[T], data []T) (dto.Page[T], error) {

	nextTokenIdx := 0

	if filters.NextToken != nil {
		for idx, n := range data {
			if getIdFn(n) == *filters.NextToken {
				nextTokenIdx = idx + 1
			}
		}
	}

	if nextTokenIdx == len(data) {
		page := dto.Page[T]{
			NextToken:   nil,
			PrevToken:   filters.NextToken,
			ResultCount: 0,
		}

		return page, nil
	}

	data = data[nextTokenIdx:]

	pageSize := 50

	if filters.MaxResults != nil {
		pageSize = *filters.MaxResults
	}

	pageSize = min(pageSize, len(data))
	data = data[:pageSize]

	nextToken := getIdFn(data[len(data)-1])

	page := dto.Page[T]{
		NextToken:   &nextToken,
		PrevToken:   filters.NextToken,
		ResultCount: len(data),
		Data:        data,
	}

	return page, nil
}

func (s *InMemoryStorage) GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) (dto.Page[dto.UserNotification], error) {

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

	sort.Slice(userNotifications, func(i, j int) bool {
		return userNotifications[i].CreatedAt > userNotifications[j].CreatedAt
	})

	getNotificationId := func(n dto.UserNotification) string {
		return n.Id
	}

	return makePage(filters.PageFilter, getNotificationId, userNotifications)
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
		return e.NotificationNotFound{NotificationId: notificationId}
	} else {
		readAt := time.Now().String()
		userNotification.ReadAt = &readAt
	}

	return nil
}

func (s *InMemoryStorage) CreateDistributionList(ctx context.Context, newDL dto.DistributionList) error {

	dlName := newDL.Name

	if dl := s.getDistributionList(dlName); dl != nil {
		return e.DistributionListAlreadyExists{Name: dlName}
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

func (s *InMemoryStorage) GetDistributionLists(ctx context.Context, filters dto.PageFilter) (dto.Page[dto.DistributionListSummary], error) {

	summaries := make([]dto.DistributionListSummary, 0, len(s.distributionLists))

	for _, dl := range s.distributionLists {
		summary := dto.DistributionListSummary{
			Name:               dl.Name,
			NumberOfRecipients: len(dl.Recipients),
		}

		summaries = append(summaries, summary)
	}

	getDistributionListId := func(summary dto.DistributionListSummary) string {
		return summary.Name
	}

	return makePage(filters, getDistributionListId, summaries)
}

func (s *InMemoryStorage) GetRecipients(ctx context.Context, distlistName string, filter dto.PageFilter) (dto.Page[string], error) {

	dl := s.getDistributionList(distlistName)

	if dl == nil {
		err := e.DistributionListNotFound{Name: distlistName}
		return dto.Page[string]{}, err
	}

	recipients := make([]string, 0, len(dl.Recipients))

	for recipient := range dl.Recipients {
		recipients = append(recipients, recipient)
	}

	sort.Slice(recipients, func(i, j int) bool {
		return recipients[i] < recipients[j]
	})

	getUserRecipientId := func(recipient string) string {
		return recipient
	}

	return makePage(filter, getUserRecipientId, recipients)
}

func (s *InMemoryStorage) AddRecipients(ctx context.Context, distlistName string, recipients []string) (dto.DistributionListSummary, error) {

	dl := s.getDistributionList(distlistName)

	if dl == nil {
		err := e.DistributionListNotFound{Name: distlistName}
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

	dl := s.getDistributionList(distlistName)

	if dl == nil {
		err := e.DistributionListNotFound{Name: distlistName}
		return dto.DistributionListSummary{}, err
	}

	for _, recipient := range recipients {
		delete(dl.Recipients, recipient)
	}

	summary := dto.DistributionListSummary{
		Name:               dl.Name,
		NumberOfRecipients: len(dl.Recipients),
	}

	return summary, nil
}

func (s *InMemoryStorage) DeleteDistributionList(ctx context.Context, distlistName string) error {
	lists := make([]distributionList, 0)

	for _, dl := range s.distributionLists {
		if dl.Name != distlistName {
			lists = append(lists, dl)
		}
	}

	s.distributionLists = lists

	return nil
}

func MakeInMemoryStorage() InMemoryStorage {
	storage := InMemoryStorage{}

	storage.notifications = make(map[string]notification)
	storage.userNotifications = make(map[string][]dto.UserNotification)
	storage.usersConfig = make(map[string][]dto.ChannelConfig)
	storage.distributionLists = make([]distributionList, 0)

	return storage
}
