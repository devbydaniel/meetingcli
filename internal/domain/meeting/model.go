package meeting

import "time"

// Meeting represents a completed meeting with its artifacts.
type Meeting struct {
	Name           string
	Dir            string
	StartedAt      time.Time
	EndedAt        time.Time
	AudioPath      string
	TranscriptPath string
	SummaryPath    string
}

// RecordingState tracks an active recording session.
// Persisted as JSON in the state directory.
type RecordingState struct {
	PID               int       `json:"pid"`
	StartedAt         time.Time `json:"started_at"`
	AudioPath         string    `json:"audio_path"`
	MeetingDir        string    `json:"meeting_dir"`
	OriginalOutputUID string    `json:"original_output_uid"`
	MultiOutputID     uint32    `json:"multi_output_id"`
	AggregateID       uint32    `json:"aggregate_id"`
}
