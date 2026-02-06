package cli

import (
	"github.com/spf13/cobra"

	"github.com/devbydaniel/meetingcli/config"
	"github.com/devbydaniel/meetingcli/internal/app"
	"github.com/devbydaniel/meetingcli/internal/version"
)

type Dependencies struct {
	App    *app.App
	Config *config.Config
}

func NewRootCmd(deps *Dependencies) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "meeting",
		Short: "Record meetings, transcribe, and summarize",
		Long:  "A CLI tool that records meetings, generates transcripts using Mistral Voxtral, and creates AI summaries using Claude Haiku.",
	}

	rootCmd.Version = version.Version
	rootCmd.SetVersionTemplate(version.Full() + "\n")

	rootCmd.AddCommand(NewStartCmd(deps))
	rootCmd.AddCommand(NewListCmd(deps))
	rootCmd.AddCommand(NewDoctorCmd(deps))

	return rootCmd
}
