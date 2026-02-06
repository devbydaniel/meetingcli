package audio

import (
	"fmt"
	"os"
	"os/exec"
)

// Recorder manages ffmpeg-based mic recording.
type Recorder struct{}

func NewRecorder() *Recorder {
	return &Recorder{}
}

func (r *Recorder) CheckFFmpeg() error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found. Install with: brew install ffmpeg")
	}
	return nil
}

// RecordMic records from the default input device. Blocks until the process exits (e.g. SIGINT).
func (r *Recorder) RecordMic(outputPath string) error {
	cmd := exec.Command("ffmpeg",
		"-f", "avfoundation",
		"-i", ":default",
		"-ac", "1",
		"-ar", "16000",
		"-y",
		outputPath,
	)

	// Log stderr for diagnostics
	logPath := outputPath + ".ffmpeg.log"
	if logFile, err := os.Create(logPath); err == nil {
		cmd.Stderr = logFile
		defer logFile.Close()
	}

	return cmd.Run()
}

// MergeAudio combines system audio and mic audio into a single mono WAV.
func (r *Recorder) MergeAudio(systemPath, micPath, outputPath string) error {
	cmd := exec.Command("ffmpeg",
		"-i", systemPath,
		"-i", micPath,
		"-filter_complex", "[0:a][1:a]amix=inputs=2:duration=longest:dropout_transition=0[a]",
		"-map", "[a]",
		"-ac", "1",
		"-ar", "16000",
		"-y",
		outputPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("merging audio: %w\n%s", err, string(out))
	}
	return nil
}
