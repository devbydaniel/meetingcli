package cli

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/devbydaniel/meetingcli/internal/output"
)

func NewDoctorCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check prerequisites",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := output.NewFormatter(os.Stdout)
			ok := true

			if _, err := exec.LookPath("ffmpeg"); err != nil {
				f.SetupCheck("ffmpeg", false, "not found. Install with: brew install ffmpeg")
				ok = false
			} else {
				f.SetupCheck("ffmpeg", true, "installed")
			}

			f.SetupCheck("Screen recording", true, "permission will be requested on first recording")

			if deps.Config.MistralAPIKey != "" {
				f.SetupCheck("Mistral API key", true, "configured")
			} else {
				f.SetupCheck("Mistral API key", false, "not set. Set MEETINGCLI_MISTRAL_API_KEY or add to config")
				ok = false
			}

			if deps.Config.AnthropicKey != "" {
				f.SetupCheck("Anthropic API key", true, "configured")
			} else {
				f.SetupCheck("Anthropic API key", false, "not set. Set MEETINGCLI_ANTHROPIC_API_KEY or add to config")
				ok = false
			}

			f.SetupCheck("Meetings directory", true, deps.Config.MeetingsDir)

			if ok {
				f.Success("\nAll prerequisites met. Ready to record!")
			} else {
				f.Warning("\nSome prerequisites are missing.")
			}
			return nil
		},
	}
}
