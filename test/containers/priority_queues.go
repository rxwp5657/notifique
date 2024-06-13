package containers

import "github.com/notifique/internal/publisher"

const (
	PRIORITY_QUEUE_LOW_NAME    = "notifique-low"
	PRIORITY_QUEUE_MEDIUM_NAME = "notifique-medium"
	PRIORITY_QUEUE_HIGH_NAME   = "notifique-high"
)

func MakePriorityQueueConfig() (queues publisher.PriorityQueues) {
	low := PRIORITY_QUEUE_LOW_NAME
	medium := PRIORITY_QUEUE_MEDIUM_NAME
	high := PRIORITY_QUEUE_HIGH_NAME

	queues.Low = &low
	queues.Medium = &medium
	queues.High = &high

	return
}
