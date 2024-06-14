package containers

import "github.com/notifique/internal/publisher"

const (
	LowPriorityQueue    = "notifique-low"
	MediumPriorityQueue = "notifique-medium"
	HighPriorityQueue   = "notifique-high"
)

func MakePriorityQueueConfig() (queues publisher.PriorityQueues) {

	low := LowPriorityQueue
	medium := MediumPriorityQueue
	high := HighPriorityQueue

	queues.Low = &low
	queues.Medium = &medium
	queues.High = &high

	return
}
