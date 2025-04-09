package dto

import (
	sdto "github.com/notifique/shared/dto"
)

type NotificationContentsType string

const (
	Template NotificationContentsType = "TEMPLATE"
	Raw      NotificationContentsType = "RAW"
)

type NotificationUriParams struct {
	NotificationId string `uri:"id" binding:"required,uuid"`
}

type NotificationSummary struct {
	Id           string                    `json:"id"`
	Topic        string                    `json:"topic"`
	ContentsType NotificationContentsType  `json:"contentsType"`
	Priority     sdto.NotificationPriority `json:"priority"`
	Status       sdto.NotificationStatus   `json:"status"`
	CreatedAt    string                    `json:"createdAt"`
	CreatedBy    string                    `json:"createdBy"`
}

type NotificationResp struct {
	sdto.NotificationReq
	Id        string                  `json:"id"`
	Status    sdto.NotificationStatus `json:"status"`
	CreatedAt string                  `json:"createdAt"`
	CreatedBy string                  `json:"createdBy"`
}
