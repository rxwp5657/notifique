package registry

import (
	"github.com/notifique/shared/dto"
)

func IsDeletableStatus(status dto.NotificationStatus) bool {

	deletableStatuses := map[dto.NotificationStatus]struct{}{
		dto.Sent:    {},
		dto.Failed:  {},
		dto.Created: {},
	}

	_, ok := deletableStatuses[status]

	return ok
}
