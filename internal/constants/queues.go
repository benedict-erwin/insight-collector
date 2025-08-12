package constants

// Queue priority constants for job processing
const (
	QueueCritical = "critical" // High priority jobs (user activities, security, payments)
	QueueDefault  = "default"  // Normal priority jobs (notifications, processing)
	QueueLow      = "low"      // Background jobs (cleanup, reports, analytics)
)

// GetAllQueues returns all valid queue names
func GetAllQueues() []string {
	return []string{
		QueueCritical,
		QueueDefault,
		QueueLow,
	}
}

// IsValidQueue checks if queue name is valid
func IsValidQueue(queue string) bool {
	for _, validQueue := range GetAllQueues() {
		if queue == validQueue {
			return true
		}
	}
	return false
}

// GetQueuePriority returns numeric priority for queue (higher = more important)
func GetQueuePriority(queue string) int {
	switch queue {
	case QueueCritical:
		return 3
	case QueueDefault:
		return 2
	case QueueLow:
		return 1
	default:
		return 0
	}
}