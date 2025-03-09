package registry

import dto "github.com/notifique/internal/server/dto"

func IsDeletableStatus(status dto.NotificationStatus) bool {

	deletableStatuses := map[dto.NotificationStatus]struct{}{
		dto.Sent:    {},
		dto.Failed:  {},
		dto.Created: {},
	}

	_, ok := deletableStatuses[status]

	return ok
}
