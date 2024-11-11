package storage

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
	id = @notificationId;
`
