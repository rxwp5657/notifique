package dto

type ChannelConfig struct {
	OptIn       bool    `json:"optIn"`
	SnoozeUntil *string `json:"snoozeUntil" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00,future"`
}

type UserConfig struct {
	EmailConfig ChannelConfig `json:"emailConfig"`
	SMSConfig   ChannelConfig `json:"smsConfig"`
	InAppConfig ChannelConfig `json:"inappConfig"`
}
