package storage

type userConfig struct {
	UserId           string
	EmailOptIn       bool
	EmailSnoozeUntil string
	SMSOptIn         bool
	SMSSnoozeUntil   string
	InAppOptIn       bool
	InAppSnoozeUntil string
	PushOptIn        bool
	PushSnoozeUntil  string
}
