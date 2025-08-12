package asynq

// Payload
type Payload struct {
	TaskId   string      // Asynq TaskID metadata
	TaskType string      // Asynq TaskType metadata
	Data     interface{} // The Task Payload (JSON)
}
