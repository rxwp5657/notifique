package controllers

import "time"

const (
	UserIdHeaderKey       = "userId"
	userNotificationEvent = "userNotification"
	NotificationStatusTTL = 15 * time.Minute
	NotificationHashTTL   = 5 * time.Minute
)
