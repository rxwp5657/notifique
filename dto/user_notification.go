package dto

type UserNotificationFilters struct {
	PageFilter
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

type UserNotificationReq struct {
	Title    string  `json:"title"`
	Contents string  `json:"contents"`
	Topic    string  `json:"topic" binding:"required,min=1,max=120"`
	Image    *string `json:"image" binding:"omitempty,uri"`
}
