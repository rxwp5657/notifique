package dto

type ChannelConfig struct {
	Channel     string  `json:"channel" binding:"required,oneof=e-mail sms in-app"`
	OptIn       bool    `json:"optIn"`
	SnoozeUntil *string `json:"snoozeUntil" binding:"datetime=2006-01-02T15:04:05Z07:00,gt"`
}

type UserConfig struct {
	Config []ChannelConfig `binding:"unique=Channel"`
}
