package usecases

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"text/template"
	"time"

	"github.com/devbydaniel/meetingcli/internal/audio"
	"github.com/devbydaniel/meetingcli/internal/domain/meeting"
)

// Record handles recording a meeting. Foreground only — blocks until Ctrl+C.
type Record struct {
	Capturer       *audio.SystemAudioCapturer
	Recorder       *audio.Recorder
	MeetingsDir    string
	FolderTemplate string
}

type RecordOptions struct {
	Name string
}

type FolderTemplateData struct {
	Year, Month, Day, Hour, Minute, Second, Name string
}

// Execute runs a recording session. Blocks until interrupted (Ctrl+C).
// Returns the result with paths to the merged audio and meeting dir.
func (r *Record) Execute(opts *RecordOptions) (*meeting.RecordingResult, error) {
	if err := r.Recorder.CheckFFmpeg(); err != nil {
		return nil, err
	}

	// Create meeting directory
	now := time.Now()
	dirName, err := r.renderFolderName(now, opts.Name)
	if err != nil {
		return nil, fmt.Errorf("rendering folder name: %w", err)
	}
	meetingDir := filepath.Join(r.MeetingsDir, dirName)
	if err := os.MkdirAll(meetingDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating meeting directory: %w", err)
	}

	micPath := filepath.Join(meetingDir, "mic.wav")
	systemPath := filepath.Join(meetingDir, "system.wav")
	audioPath := filepath.Join(meetingDir, "recording.wav")

	// Start system audio capture (cgo, streams to disk)
	if err := r.Capturer.StartCapture(systemPath); err != nil {
		return nil, err
	}

	// Record mic in foreground — blocks until SIGINT
	// We trap SIGINT ourselves so we can clean up both streams.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	micDone := make(chan error, 1)
	go func() {
		micDone <- r.Recorder.RecordMic(micPath)
	}()

	// Wait for interrupt or mic to stop
	select {
	case <-sigCh:
		// User pressed Ctrl+C
	case <-micDone:
		// ffmpeg exited on its own (unlikely)
	}
	signal.Stop(sigCh)

	// Stop system audio capture and finalize WAV
	r.Capturer.StopCapture()

	// Wait for ffmpeg to finish (it also gets the SIGINT)
	<-micDone

	// Merge system + mic into recording.wav
	if err := r.Recorder.MergeAudio(systemPath, micPath, audioPath); err != nil {
		// Fall back to mic-only
		fmt.Fprintf(os.Stderr, "warning: could not merge audio: %v\n", err)
		if _, statErr := os.Stat(micPath); statErr == nil {
			_ = os.Rename(micPath, audioPath)
		}
	}

	return &meeting.RecordingResult{
		StartedAt:  now,
		AudioPath:  audioPath,
		MeetingDir: meetingDir,
	}, nil
}

func (r *Record) renderFolderName(t time.Time, name string) (string, error) {
	tmpl, err := template.New("folder").Parse(r.FolderTemplate)
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
