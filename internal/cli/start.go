package cli

import (
	"github.com/spf13/cobra"
)

func NewStartCmd(deps *Dependencies) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Record a meeting",
		Long:  "Record mic + system audio. Press Ctrl+C to stop, then transcribe and summarize.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRecording(deps, name)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Meeting name (used in folder name)")
	return cmd
}
