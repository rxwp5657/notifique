package dto

type NotificationReq struct {
	Title      string   `json:"title" binding:"required,max=120"`
	Contents   string   `json:"contents" binding:"required,max=1024"`
	Image      *string  `json:"image" binding:"omitempty,uri"`
	Topic      string   `json:"topic" binding:"required,min=1,max=120"`
	Priority   string   `json:"priority" binding:"oneof=HIGH MEDIUM LOW"`
	Recipients []string `json:"recipients" binding:"unique,max=256,dive,min=1"`
	Channels   []string `json:"channels" binding:"unique,dive,oneof=e-mail sms in-app"`
}

type NotificationUriParams struct {
	NotificationId string `uri:"id" binding:"required,uuid"`
}
