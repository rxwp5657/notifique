package storage

type notification struct {
	Id               string
	Title            string
	Contents         string
	ImageUrl         *string
	Topic            string
	Priority         string
	DistributionList *string
	CreatedAt        string
}

type notificationRecipients struct {
	NotificationId string
	Recipients     string
}

type notificationChannels struct {
	NotificationId string
	Channel        string
}
