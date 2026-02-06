package cli

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/devbydaniel/meetingcli/internal/domain/meeting/usecases"
	"github.com/devbydaniel/meetingcli/internal/output"
)

func NewStartCmd(deps *Dependencies) *cobra.Command {
	var name string
	var sync bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start recording a meeting",
		Long:  "Start recording audio from microphone and system audio.\nUse --sync to record in foreground (Ctrl+C to stop), or run in background and use 'meeting stop'.",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := output.NewFormatter(os.Stdout)

			opts := &usecases.StartOptions{
				Name: name,
				Sync: sync,
			}

			if sync {
				return runSyncRecording(deps, opts, formatter)
			}

			return runAsyncRecording(deps, opts, formatter)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Meeting name (used in folder name)")
	cmd.Flags().BoolVar(&sync, "sync", false, "Record in foreground (Ctrl+C to stop)")

	return cmd
}

func runAsyncRecording(deps *Dependencies, opts *usecases.StartOptions, formatter *output.Formatter) error {
	state, err := deps.App.StartRecording.Execute(opts)
	if err != nil {
		return err
	}

	formatter.RecordingStarted(state, false)
	return nil
}

func runSyncRecording(deps *Dependencies, opts *usecases.StartOptions, formatter *output.Formatter) error {
	state, err := deps.App.StartRecording.Execute(opts)
	if err != nil {
		return err
	}

	formatter.RecordingStarted(state, true)

	// In sync mode, StartRecording.Execute blocks until ffmpeg exits.
	// When it returns, the recording has been stopped and audio devices cleaned up.
	duration := time.Since(state.StartedAt)
	formatter.RecordingStopped(duration)

	// Now process the recording
	return processRecording(deps, state.AudioPath, state.MeetingDir, formatter)
}

func processRecording(deps *Dependencies, audioPath string, meetingDir string, formatter *output.Formatter) error {
	// Transcribe
	formatter.Transcribing()
	result, err := deps.App.Transcribe.Execute(audioPath, meetingDir)
	if err != nil {
		return err
	}
	formatter.TranscribeDone(filepath.Join(meetingDir, "transcript.md"))

	// Summarize
	formatter.Summarizing()
	_, err = deps.App.Summarize.Execute(result.Text, meetingDir)
	if err != nil {
		return err
	}
	formatter.SummarizeDone(filepath.Join(meetingDir, "summary.md"))

	formatter.MeetingComplete(meetingDir)
	return nil
}
