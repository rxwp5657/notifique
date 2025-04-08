package dto

type UserNotificationReq struct {
	UserId   string  `json:"userId" binding:"required"`
	Title    string  `json:"title" binding:"required,max=120"`
	Contents string  `json:"contents" binding:"required,max=1024"`
	Topic    string  `json:"topic" binding:"required,min=1,max=120"`
	Image    *string `json:"image" binding:"omitempty,uri"`
}

type UserEmailNotificationReq struct {
	UserNotificationReq
	Email  string `json:"email" binding:"required,email"`
	IsHtml bool   `json:"isHtml" binding:"required"`
}
