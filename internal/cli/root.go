package cli

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/devbydaniel/meetingcli/config"
	"github.com/devbydaniel/meetingcli/internal/app"
	"github.com/devbydaniel/meetingcli/internal/domain/meeting/usecases"
	"github.com/devbydaniel/meetingcli/internal/output"
	"github.com/devbydaniel/meetingcli/internal/version"
)

type Dependencies struct {
	App    *app.App
	Config *config.Config
}

func NewRootCmd(deps *Dependencies) *cobra.Command {
	var name string

	rootCmd := &cobra.Command{
		Use:   "meeting",
		Short: "Record meetings, transcribe, and summarize",
		Long:  "A CLI tool that records meetings, generates transcripts using Mistral Voxtral, and creates AI summaries using Claude Haiku.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRecording(deps, name)
		},
	}

	rootCmd.Version = version.Version
	rootCmd.SetVersionTemplate(version.Full() + "\n")

	rootCmd.Flags().StringVarP(&name, "name", "n", "", "Meeting name (used in folder name)")

	rootCmd.AddCommand(NewStartCmd(deps))
	rootCmd.AddCommand(NewListCmd(deps))
	rootCmd.AddCommand(NewDoctorCmd(deps))

	return rootCmd
}

// runRecording contains the shared recording logic used by both the root command and start subcommand.
func runRecording(deps *Dependencies, name string) error {
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
}
