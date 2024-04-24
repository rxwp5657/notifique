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
	ReadAt    *string `json:"read_at,omitempty"`
	Topic     string  `json:"topic"`
}
