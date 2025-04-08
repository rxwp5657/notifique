package controllers

import "time"

const (
	userNotificationEvent = "userNotification"
	NotificationStatusTTL = 15 * time.Minute
	NotificationHashTTL   = 5 * time.Minute
)
