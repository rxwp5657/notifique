package registry

import c "github.com/notifique/internal/server/controllers"

func IsDeletableStatus(status c.NotificationStatus) bool {

	deletableStatuses := map[c.NotificationStatus]struct{}{
		c.Sent:    {},
		c.Failed:  {},
		c.Created: {},
	}

	_, ok := deletableStatuses[status]

	return ok
}
