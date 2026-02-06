package cli

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/devbydaniel/meetingcli/internal/output"
)

func NewStopCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop recording and process the meeting",
		Long:  "Stop the background recording, transcribe the audio, and generate a summary.",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := output.NewFormatter(os.Stdout)

			state, err := deps.App.StopRecording.Execute()
			if err != nil {
				return err
			}

			duration := time.Since(state.StartedAt)
			formatter.RecordingStopped(duration)

			return processRecording(deps, state.AudioPath, state.MeetingDir, formatter)
		},
	}

	return cmd
}
