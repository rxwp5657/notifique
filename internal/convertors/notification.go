package convertors

import (
	c "github.com/notifique/controllers"
)

func MakeStatusLogs(notification c.Notification, status c.NotificationStatus, errMsg *string) []c.NotificationStatusLog {

	logs := make([]c.NotificationStatusLog, 0, len(notification.Recipients))

	for _, recipient := range notification.Recipients {
		logs = append(logs, c.NotificationStatusLog{
			NotificationId: notification.Id,
			Recipient:      recipient,
			Status:         status,
			ErrorMsg:       errMsg,
		})
	}

	if notification.DistributionList != nil {
		logs = append(logs, c.NotificationStatusLog{
			NotificationId: notification.Id,
			Recipient:      *notification.DistributionList,
			Status:         status,
			ErrorMsg:       errMsg,
		})
	}

	return logs
}
