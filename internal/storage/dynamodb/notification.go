package storage

const (
	NOTIFICATIONS_TABLE              = "Notifications"
	NOTIFICATION_HASH_KEY            = "id"
	NOTIFICATION_STATUS_LOG_TABLE    = "NotificationStatusLogs"
	NOTIFICATION_STATUS_LOG_HASH_KEY = "id"
	NOTIFICATION_STATUS_LOG_SORT_KEY = "statusDate"
)

type notificationStatusLog struct {
	NotificationId string  `dynamodbav:"id"`
	Status         string  `dynamodbav:"status"`
	StatusDate     string  `dynamodbav:"statusDate"`
	Error          *string `dynamodbav:"errorMsg"`
}

type notification struct {
	Id               string   `dynamodbav:"id"`
	CreatedBy        string   `dynamodbav:"createdBy"`
	CreatedAt        string   `dynamodbav:"createdAt"`
	Title            string   `dynamodbav:"title"`
	Contents         string   `dynamodbav:"contents"`
	Image            *string  `dynamodbav:"image"`
	Topic            string   `dynamodbav:"topic"`
	Priority         string   `dynamodbav:"priority"`
	DistributionList *string  `dynamodbav:"distributionList"`
	Recipients       []string `dynamodbav:"recipients"`
	Channels         []string `dynamodbav:"channels"`
}
