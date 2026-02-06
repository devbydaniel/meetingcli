package audio

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Recorder manages ffmpeg-based audio recording.
type Recorder struct{}

// NewRecorder creates a new Recorder.
func NewRecorder() *Recorder {
	return &Recorder{}
}

// CheckFFmpeg verifies that ffmpeg is installed.
func (r *Recorder) CheckFFmpeg() error {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found. Install with: brew install ffmpeg")
	}
	return nil
}

// StartBackground starts ffmpeg recording in the background and returns the process.
// Records from the given aggregate device (by name) to outputPath as 16kHz mono WAV.
func (r *Recorder) StartBackground(deviceName string, outputPath string) (*os.Process, error) {
	args := []string{
		"-f", "avfoundation",
		"-i", ":" + deviceName,
		"-ac", "1",
		"-ar", "16000",
		"-y", // overwrite
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = nil

	// Log stderr to file next to the recording for diagnostics
	logPath := outputPath + ".ffmpeg.log"
	logFile, err := os.Create(logPath)
	if err != nil {
		cmd.Stderr = nil
	} else {
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		if logFile != nil {
			logFile.Close()
		}
		return nil, fmt.Errorf("starting ffmpeg: %w", err)
	}

	// Close log file when process exits (in background)
	go func() {
		cmd.Wait()
		if logFile != nil {
			logFile.Close()
		}
	}()

	return cmd.Process, nil
}

// StartForeground starts ffmpeg recording in the foreground (blocking).
// Returns when the process exits (e.g., after receiving SIGINT).
func (r *Recorder) StartForeground(deviceName string, outputPath string) error {
	args := []string{
		"-f", "avfoundation",
		"-i", ":" + deviceName,
		"-ac", "1",
		"-ar", "16000",
		"-y",
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

// StopProcess sends SIGINT to a process and waits for it to exit.
func (r *Recorder) StopProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}

	// Send SIGINT for graceful ffmpeg shutdown (finalizes file headers)
	if err := proc.Signal(syscall.SIGINT); err != nil {
		return fmt.Errorf("signaling process %d: %w", pid, err)
	}

	// Wait for process to exit.
	// ffmpeg exits with non-zero on SIGINT — that's expected, so we ignore the error.
	_, _ = proc.Wait()
	return nil
}
