package dto

type ChannelConfig struct {
	Channel     string  `json:"channel" binding:"required,oneof=e-mail sms in-app"`
	OptIn       bool    `json:"optIn"`
	SnoozeUntil *string `json:"snoozeUntil" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00,future"`
}

type UserConfig struct {
	Config []ChannelConfig `json:"config" binding:"unique=Channel,dive"`
}
