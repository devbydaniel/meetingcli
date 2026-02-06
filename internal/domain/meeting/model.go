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

// RecordingResult holds paths after a recording session completes.
type RecordingResult struct {
	StartedAt  time.Time
	AudioPath  string
	MeetingDir string
}
