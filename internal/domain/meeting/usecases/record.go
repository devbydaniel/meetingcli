package usecases

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/devbydaniel/meetingcli/internal/audio"
	"github.com/devbydaniel/meetingcli/internal/domain/meeting"
)

// StartRecording manages starting a meeting recording.
type StartRecording struct {
	DeviceManager  *audio.DeviceManager
	Recorder       *audio.Recorder
	MeetingsDir    string
	StateDir       string
	FolderTemplate string
}

// FolderTemplateData holds the template variables available for folder naming.
type FolderTemplateData struct {
	Year   string
	Month  string
	Day    string
	Hour   string
	Minute string
	Second string
	Name   string
}

// StartOptions holds options for starting a recording.
type StartOptions struct {
	Name string // optional meeting name suffix
	Sync bool   // if true, record in foreground
}

func (s *StartRecording) stateFilePath() string {
	return filepath.Join(s.StateDir, "current.json")
}

// IsRecording checks if a recording is currently active.
func (s *StartRecording) IsRecording() bool {
	_, err := os.Stat(s.stateFilePath())
	return err == nil
}

// Execute starts a new recording session.
// In async mode, returns immediately after starting the background process.
// In sync mode, blocks until the recording is stopped (via SIGINT).
func (s *StartRecording) Execute(opts *StartOptions) (*meeting.RecordingState, error) {
	if s.IsRecording() {
		return nil, fmt.Errorf("a recording is already in progress. Run 'meeting stop' first")
	}

	// Check prerequisites
	if err := s.Recorder.CheckFFmpeg(); err != nil {
		return nil, err
	}

	// Find BlackHole
	bh, err := s.DeviceManager.FindBlackhole()
	if err != nil {
		return nil, fmt.Errorf("BlackHole 2ch not found: %w\nInstall with: brew install blackhole-2ch", err)
	}

	// Create audio devices
	devices, err := s.DeviceManager.CreateDevices(bh.UID)
	if err != nil {
		return nil, fmt.Errorf("creating audio devices: %w", err)
	}

	// Create meeting directory
	now := time.Now()
	dirName, err := s.renderFolderName(now, opts.Name)
	if err != nil {
		s.cleanup(devices)
		return nil, fmt.Errorf("rendering folder name: %w", err)
	}
	meetingDir := filepath.Join(s.MeetingsDir, dirName)
	if err := os.MkdirAll(meetingDir, 0o755); err != nil {
		s.cleanup(devices)
		return nil, fmt.Errorf("creating meeting directory: %w", err)
	}

	audioPath := filepath.Join(meetingDir, "recording.wav")

	state := &meeting.RecordingState{
		StartedAt:         now,
		AudioPath:         audioPath,
		MeetingDir:        meetingDir,
		OriginalOutputUID: devices.OriginalOutputUID,
		MultiOutputID:     devices.MultiOutputID,
		AggregateID:       devices.AggregateID,
	}

	if opts.Sync {
		// Write state so cleanup can happen even on crash
		if err := s.writeState(state); err != nil {
			s.cleanup(devices)
			return nil, err
		}

		// Record in foreground (blocks until SIGINT)
		_ = s.Recorder.StartForeground(devices.AggregateName, audioPath)

		// Clean up after recording
		s.restoreAndCleanup(devices)
		s.removeState()

		return state, nil
	}

	// Async: start background process
	proc, err := s.Recorder.StartBackground(devices.AggregateName, audioPath)
	if err != nil {
		s.cleanup(devices)
		return nil, fmt.Errorf("starting recording: %w", err)
	}

	state.PID = proc.Pid

	// Save state
	if err := s.writeState(state); err != nil {
		_ = proc.Kill()
		s.cleanup(devices)
		return nil, err
	}

	// Release the process so it continues in background
	_ = proc.Release()

	return state, nil
}

func (s *StartRecording) renderFolderName(t time.Time, name string) (string, error) {
	tmpl, err := template.New("folder").Parse(s.FolderTemplate)
	if err != nil {
		return "", fmt.Errorf("invalid folder template: %w", err)
	}

	data := FolderTemplateData{
		Year:   t.Format("2006"),
		Month:  t.Format("01"),
		Day:    t.Format("02"),
		Hour:   t.Format("15"),
		Minute: t.Format("04"),
		Second: t.Format("05"),
		Name:   name,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing folder template: %w", err)
	}
	return buf.String(), nil
}

func (s *StartRecording) writeState(state *meeting.RecordingState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	return os.WriteFile(s.stateFilePath(), data, 0o644)
}

func (s *StartRecording) removeState() {
	os.Remove(s.stateFilePath())
}

func (s *StartRecording) cleanup(devices *audio.CreatedDevices) {
	_ = s.DeviceManager.DestroyDevices(devices.MultiOutputID, devices.AggregateID)
}

func (s *StartRecording) restoreAndCleanup(devices *audio.CreatedDevices) {
	_ = s.DeviceManager.SwitchOutput(devices.OriginalOutputUID)
	_ = s.DeviceManager.DestroyDevices(devices.MultiOutputID, devices.AggregateID)
}

// StopRecording manages stopping a meeting recording.
type StopRecording struct {
	DeviceManager *audio.DeviceManager
	Recorder      *audio.Recorder
	StateDir      string
}

func (s *StopRecording) stateFilePath() string {
	return filepath.Join(s.StateDir, "current.json")
}

// Execute stops the current recording and returns the state for processing.
func (s *StopRecording) Execute() (*meeting.RecordingState, error) {
	data, err := os.ReadFile(s.stateFilePath())
	if err != nil {
		return nil, fmt.Errorf("no active recording found")
	}

	var state meeting.RecordingState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("reading recording state: %w", err)
	}

	// Stop ffmpeg
	if state.PID > 0 {
		if err := s.Recorder.StopProcess(state.PID); err != nil {
			// Process may have died — continue with cleanup
			fmt.Fprintf(os.Stderr, "warning: could not stop recording process: %v\n", err)
		}
	}

	// Restore audio output
	if state.OriginalOutputUID != "" {
		_ = s.DeviceManager.SwitchOutput(state.OriginalOutputUID)
	}

	// Destroy audio devices
	if state.MultiOutputID > 0 || state.AggregateID > 0 {
		_ = s.DeviceManager.DestroyDevices(state.MultiOutputID, state.AggregateID)
	}

	// Remove state file
	os.Remove(s.stateFilePath())

	return &state, nil
}
