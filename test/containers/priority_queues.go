package containers

import "github.com/notifique/internal/publish"

const (
	LowPriorityQueue    = "notifique-low"
	MediumPriorityQueue = "notifique-medium"
	HighPriorityQueue   = "notifique-high"
)

func NewPriorityQueueConfig() (queues publish.PriorityQueues) {

	low := LowPriorityQueue
	medium := MediumPriorityQueue
	high := HighPriorityQueue

	queues.Low = &low
	queues.Medium = &medium
	queues.High = &high

	return
}
