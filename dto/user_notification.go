package dto

type UserNotificationFilters struct {
	UserId string
	Take   *int     `form:"take" binding:"omitempty,min=0"`
	Skip   *int     `form:"skip" binding:"omitempty,min=0"`
	Topics []string `form:"topics" binding:"unique"`
}

type UserNotification struct {
	Id        string  `json:"id"`
	Title     string  `json:"title"`
	Contents  string  `json:"contents"`
	CreatedAt string  `json:"created_at"`
	ReadAt    *string `json:"read_at,omitempty"`
	Topic     string  `json:"topic"`
}
