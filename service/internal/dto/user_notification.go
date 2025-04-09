package dto

import (
	sdto "github.com/notifique/shared/dto"
)

type UserNotificationFilters struct {
	sdto.PageFilter
	UserId string
	Topics []string `form:"topics" binding:"unique"`
}

type UserNotification struct {
	Id        string  `json:"id"`
	Title     string  `json:"title"`
	Contents  string  `json:"contents"`
	CreatedAt string  `json:"createdAt"`
	Image     *string `json:"image"`
	ReadAt    *string `json:"readAt,omitempty"`
	Topic     string  `json:"topic"`
}

type UserNotificationUriParam struct {
	Id string `uri:"id"`
}
