package dto

type UserNotificationFilters struct {
	UserId   string
	Page     *int     `form:"page" binding:"omitempty,min=0"`
	PageSize *int     `form:"pageSize" binding:"omitempty,min=0"`
	Topics   []string `form:"topics" binding:"unique"`
}

type UserNotification struct {
	Id        string  `json:"id"`
	Title     string  `json:"title"`
	Contents  string  `json:"contents"`
	CreatedAt string  `json:"created_at"`
	Image     *string `json:"image"`
	ReadAt    *string `json:"read_at,omitempty"`
	Topic     string  `json:"topic"`
}

type UserNotificationUriParam struct {
	Id string `form:"id"`
}

type UserNotificationReq struct {
	Title    string  `json:"title"`
	Contents string  `json:"contents"`
	Topic    string  `json:"topic" binding:"required,min=1,max=120"`
	Image    *string `json:"image" binding:"omitempty,uri"`
}
