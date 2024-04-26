package internal

import (
	"context"
	"math"
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
			return "", DistributionListNotFound{dlName}
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

	page := 1

	if filters.Page != nil {
		page = *filters.Page
	}

	pageSize := min(len(userNotifications), 15)

	if filters.PageSize != nil {
		pageSize = *filters.PageSize
		pageSize = min(pageSize, len(userNotifications)-(page-1)*pageSize)
	}

	totalRecords := len(userNotifications)
	userNotifications = userNotifications[(page-1)*pageSize:]
	userNotifications = userNotifications[:page]

	return makePage(page, pageSize, totalRecords, userNotifications), nil
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

	if dl := s.getDistributionList(dlName); dl != nil {
		return DistributionListAlreadyExists{dlName}
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

func makePage[T any](page, pageSize, totalRecords int, data []T) dto.Page[T] {

	totalPages := int(math.Ceil(float64(totalRecords) / float64(pageSize)))

	var nextPage *int = nil
	var prevPage *int = nil

	if page+1 <= totalPages {
		np := page + 1
		nextPage = &np
	}

	if page != 1 {
		pp := page - 1
		prevPage = &pp
	}

	return dto.Page[T]{
		CurrentPage:  page,
		NextPage:     nextPage,
		PrevPage:     prevPage,
		TotalPages:   totalPages,
		TotalRecords: totalRecords,
		Data:         data,
	}

}

func (s *InMemoryStorage) GetDistributionLists(ctx context.Context, filter dto.PageFilter) (dto.Page[dto.DistributionListSummary], error) {

	page, pageSize := 1, min(len(s.distributionLists), 500)

	if filter.Page != nil {
		page = *filter.Page
	}

	if filter.PageSize != nil {
		pageSize = *filter.PageSize
		pageSize = min(pageSize, len(s.distributionLists)-(page-1)*pageSize)
	}

	lists := s.distributionLists[(page-1)*pageSize:]
	summaries := make([]dto.DistributionListSummary, 0, pageSize)

	for _, dl := range lists[:pageSize] {

		summary := dto.DistributionListSummary{
			Name:               dl.Name,
			NumberOfRecipients: len(dl.Recipients),
		}

		summaries = append(summaries, summary)
	}

	return makePage(page, pageSize, len(s.distributionLists), summaries), nil
}

func (s *InMemoryStorage) GetRecipients(ctx context.Context, distlistName string, filter dto.PageFilter) (dto.Page[string], error) {

	dl := s.getDistributionList(distlistName)

	if dl == nil {
		return dto.Page[string]{}, DistributionListNotFound{distlistName}
	}

	recipients := make([]string, 0, len(dl.Recipients))

	for recipient := range dl.Recipients {
		recipients = append(recipients, recipient)
	}

	sort.Slice(recipients, func(i int, j int) bool {
		return recipients[i] < recipients[j]
	})

	page := 1

	if filter.Page != nil {
		page = *filter.Page
	}

	pageSize := min(len(recipients), 500)

	if filter.PageSize != nil {
		pageSize = *filter.PageSize
		pageSize = min(pageSize, len(recipients)-(page-1)*pageSize)
	}

	filteredRecipients := recipients[(page-1)*pageSize:]
	filteredRecipients = filteredRecipients[:pageSize]

	return makePage(page, pageSize, len(recipients), filteredRecipients), nil
}

func (s *InMemoryStorage) AddRecipients(ctx context.Context, distlistName string, recipients []string) (dto.DistributionListSummary, error) {

	dl := s.getDistributionList(distlistName)

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

	dl := s.getDistributionList(distlistName)

	if dl == nil {
		err := DistributionListNotFound{distlistName}
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
