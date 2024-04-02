package internal

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/notifique/dto"
)

type notification struct {
	Id         string
	Title      string
	Contents   string
	Image      *string
	Topic      string
	CreatedAt  time.Time
	Recipients map[string]*time.Time
	Channels   []string
}

type userConfig struct {
	Config map[string]bool
}

type InMemoryStorage struct {
	notificationStore map[string]notification
	userConfigStore   map[string]userConfig
}

func getAvailableChannels() []string {
	return []string{"e-mail", "in-app", "sms"}
}

func (s *InMemoryStorage) SaveNotification(ctx context.Context, notificationReq dto.NotificationReq) (string, error) {

	recipients := make(map[string]*time.Time)

	for _, r := range notificationReq.Recipients {
		recipients[r] = nil
	}

	id := uuid.NewString()

	n := notification{
		Id:         id,
		Title:      notificationReq.Title,
		Contents:   notificationReq.Contents,
		Image:      notificationReq.Image,
		Topic:      notificationReq.Topic,
		Recipients: recipients,
		Channels:   notificationReq.Channels,
		CreatedAt:  time.Now(),
	}

	s.notificationStore[id] = n

	return id, nil
}

func (s *InMemoryStorage) GetUserNotifications(ctx context.Context, filters dto.NotificationFilters) ([]dto.UserNotificationResp, error) {

	expectedTopics := make(map[string]struct{})
	expectedChannels := make(map[string]struct{})

	for _, topic := range filters.Topics {
		expectedTopics[topic] = struct{}{}
	}

	for _, channel := range filters.Channels {
		expectedChannels[channel] = struct{}{}
	}

	userNotifications := make([]notification, 0)

	shouldIncludeNotification := func(noti notification) bool {
		if _, ok := noti.Recipients[filters.UserId]; !ok {
			return false
		}

		_, hasTopic := expectedTopics[noti.Topic]

		if len(filters.Topics) != 0 && !hasTopic {
			return false
		}

		if len(filters.Channels) == 0 {
			return true
		}

		for _, channel := range noti.Channels {
			_, hasChannel := expectedChannels[channel]
			if len(filters.Channels) != 0 && hasChannel {
				return true
			}
		}

		return false
	}

	for _, noti := range s.notificationStore {
		if shouldIncludeNotification(noti) {
			userNotifications = append(userNotifications, noti)
		}
	}

	if filters.Skip != nil && *filters.Skip >= len(userNotifications) {
		return make([]dto.UserNotificationResp, 0), nil
	}

	sort.Slice(userNotifications, func(i, j int) bool {
		return userNotifications[j].CreatedAt.Before(userNotifications[i].CreatedAt)
	})

	if filters.Skip != nil {
		userNotifications = userNotifications[*filters.Skip:]
	}

	if filters.Take != nil {
		take := min(*filters.Take, len(userNotifications))
		userNotifications = userNotifications[:take]
	}

	notificationResp := make([]dto.UserNotificationResp, 0, len(userNotifications))

	for _, n := range userNotifications {

		var readAt *string = nil

		if r := n.Recipients[filters.UserId]; r != nil {
			readAtStr := r.String()
			readAt = &readAtStr
		}

		nresp := dto.UserNotificationResp{
			Id:        n.Id,
			Title:     n.Title,
			Contents:  n.Contents,
			CreatedAt: n.CreatedAt.String(),
			ReadAt:    readAt,
			Topic:     n.Topic,
			Channels:  n.Channels,
		}

		notificationResp = append(notificationResp, nresp)
	}

	return notificationResp, nil
}

func (s *InMemoryStorage) SetReadStatus(ctx context.Context, userId, notificationId string) error {

	if _, ok := s.notificationStore[notificationId]; !ok {
		return NotificationNotFound{notificationId}
	}

	n := s.notificationStore[notificationId]

	if _, ok := n.Recipients[userId]; !ok {
		return RecipientNotFound{notificationId, userId}
	}

	now := time.Now()
	n.Recipients[userId] = &now

	return nil
}

func getInitialUserConfig() map[string]bool {

	config := make(map[string]bool)
	availableChannels := getAvailableChannels()

	for _, ac := range availableChannels {
		config[ac] = true
	}

	return config
}

func getUpdatedOptConfig(cfg userConfig, channels []string, val bool) userConfig {

	if cfg.Config == nil {
		cfg.Config = getInitialUserConfig()
	}

	for _, channel := range channels {
		cfg.Config[channel] = val
	}

	return cfg
}

func makeUserConf(cfg *userConfig) []dto.UserConfigResp {

	config := make([]dto.UserConfigResp, 0)

	for channel := range cfg.Config {
		configResp := dto.UserConfigResp{
			Channel: channel,
			OptedIn: cfg.Config[channel],
		}
		config = append(config, configResp)
	}

	return config
}

func (s *InMemoryStorage) updateOptConfg(userId string, channels []string, optIn bool) {

	cfg := getUpdatedOptConfig(s.userConfigStore[userId], channels, optIn)
	s.userConfigStore[userId] = cfg
}

func (s *InMemoryStorage) OptOut(ctx context.Context, userId string, channels []string) ([]dto.UserConfigResp, error) {

	s.updateOptConfg(userId, channels, false)
	cfg := s.userConfigStore[userId]

	return makeUserConf(&cfg), nil
}

func (s *InMemoryStorage) OptIn(ctx context.Context, userId string, channels []string) ([]dto.UserConfigResp, error) {

	s.updateOptConfg(userId, channels, true)
	cfg := s.userConfigStore[userId]

	return makeUserConf(&cfg), nil
}

func (s *InMemoryStorage) GetUserConfig(ctx context.Context, userId string) ([]dto.UserConfigResp, error) {

	if _, ok := s.userConfigStore[userId]; !ok {
		s.userConfigStore[userId] = userConfig{getInitialUserConfig()}
	}

	cfg := s.userConfigStore[userId]

	return makeUserConf(&cfg), nil
}

func MakeInMemoryStorage() InMemoryStorage {
	storage := InMemoryStorage{}
	storage.notificationStore = make(map[string]notification)
	storage.userConfigStore = make(map[string]userConfig)
	return storage
}
