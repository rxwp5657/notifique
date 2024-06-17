package storage

import "time"

type notification struct {
	Id               string
	Title            string
	Contents         string
	ImageUrl         *string
	Topic            string
	Priority         string
	DistributionList *string
	CreatedAt        time.Time
	status           string
}

type notificationRecipients struct {
	NotificationId string
	Recipients     string
}

type notificationChannels struct {
	NotificationId string
	Channel        string
}

type notificationStatusLog struct {
	NotificationId string    `db:"notification_id"`
	Status         string    `db:"status"`
	StatusDate     time.Time `db:"status_date"`
	Error          *string   `db:"error_message"`
}

const InsertNotification = `
INSERT INTO notifications (
	title,
	contents,
	image_url,
	topic,
	priority,
	distribution_list,
	created_at,
	status
) VALUES (
	@title,
	@contents,
	@imageUrl,
	@topic,
	@priority,
	@distributionList,
	@createdAt,
	@status
) RETURNING
	id;
`

const InsertNotificationRecipients = `
INSERT INTO notification_recipients (
	notification_id,
	recipient
) VALUES (
	@notificationId,
	@recipient
);
`

const InsertChannels = `
INSERT INTO notification_channels (
	notification_id,
	channel
) VALUES (
	@notificationId,
	@channel
);
`

const InsertNotificationStatusLog = `
INSERT INTO notification_status_log (
	notification_id,
    status_date,
    "status",
    error_message
) VALUES (
	@notificationId,
	@statusDate,
	@status,
	@errorMessage
);
`
const UpdateNotificationStatus = `
UPDATE
	notifications
SET
	status = @status
WHERE
	id = @id;
`
