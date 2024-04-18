package internal

import (
	"context"
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

type InMemoryStorage struct {
	notifications     map[string]notification
	userNotifications map[string][]dto.UserNotification
	usersConfig       map[string][]dto.ChannelConfig
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

func MakeInMemoryStorage() InMemoryStorage {
	storage := InMemoryStorage{}

	storage.notifications = make(map[string]notification)
	storage.userNotifications = make(map[string][]dto.UserNotification)
	storage.usersConfig = make(map[string][]dto.ChannelConfig)

	return storage
}
