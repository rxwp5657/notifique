package dto

type UserNotificationResp struct {
	Id        string   `json:"id"`
	Title     string   `json:"title"`
	Contents  string   `json:"contents"`
	CreatedAt string   `json:"created_at"`
	ReadAt    *string  `json:"read_at,omitempty"`
	Topic     string   `json:"topic"`
	Channels  []string `json:"channels"`
}
