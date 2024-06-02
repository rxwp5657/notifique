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

const INSERT_NOTIFICATION = `
INSERT INTO notifications (
	title,
	contents,
	image_url,
	topic,
	priority,
	distribution_list,
	created_at
) VALUES (
	@title,
	@contents,
	@imageUrl,
	@topic,
	@priority,
	@distributionList,
	@createdAt
) RETURNING
	id;
`

const INSERT_NOTIFICATION_RECIPIENTS = `
INSERT INTO notification_recipients (
	notification_id,
	recipient
) VALUES (
	@notificationId,
	@recipient
);
`

const INSERT_CHANNELS = `
INSERT INTO notification_channels (
	notification_id,
	channel
) VALUES (
	@notificationId,
	@channel
);
`
