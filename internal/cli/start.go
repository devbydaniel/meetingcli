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

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Record a meeting",
		Long:  "Record mic + system audio. Press Ctrl+C to stop, then transcribe and summarize.",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := output.NewFormatter(os.Stdout)

			formatter.Info("Recording started. Press Ctrl+C to stop.\n")

			result, err := deps.App.Record.Execute(&usecases.RecordOptions{Name: name})
			if err != nil {
				return err
			}

			duration := time.Since(result.StartedAt)
			formatter.RecordingStopped(duration)

			// Transcribe
			formatter.Transcribing()
			transcript, err := deps.App.Transcribe.Execute(result.AudioPath, result.MeetingDir)
			if err != nil {
				return err
			}
			formatter.TranscribeDone(filepath.Join(result.MeetingDir, "transcript.md"))

			// Summarize
			formatter.Summarizing()
			if _, err := deps.App.Summarize.Execute(transcript.Text, result.MeetingDir); err != nil {
				return err
			}
			formatter.SummarizeDone(filepath.Join(result.MeetingDir, "summary.md"))

			formatter.MeetingComplete(result.MeetingDir)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Meeting name (used in folder name)")
	return cmd
}
