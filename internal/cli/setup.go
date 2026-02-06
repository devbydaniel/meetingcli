package cli

import (
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/devbydaniel/meetingcli/internal/output"
)

func isBlackholeInstalledButNotLoaded() bool {
	out, err := exec.Command("brew", "list", "--cask").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == "blackhole-2ch" {
			return true
		}
	}
	return false
}

func NewSetupCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Check prerequisites for recording",
		Long:  "Verify that ffmpeg and BlackHole 2ch are installed and working.",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := output.NewFormatter(os.Stdout)
			allOk := true

			// Check ffmpeg
			if _, err := exec.LookPath("ffmpeg"); err != nil {
				formatter.SetupCheck("ffmpeg", false, "not found. Install with: brew install ffmpeg")
				allOk = false
			} else {
				formatter.SetupCheck("ffmpeg", true, "installed")
			}

			// Check BlackHole
			bh, err := deps.App.StartRecording.DeviceManager.FindBlackhole()
			if err != nil {
				// Check if installed but not loaded (needs reboot)
				msg := "not found. Install with: brew install blackhole-2ch"
				if isBlackholeInstalledButNotLoaded() {
					msg = "installed but not loaded. Reboot your Mac for it to take effect"
				}
				formatter.SetupCheck("BlackHole 2ch", false, msg)
				allOk = false
			} else {
				formatter.SetupCheck("BlackHole 2ch", true, "found ("+bh.Name+")")
			}

			// Check API keys
			if deps.Config.MistralAPIKey != "" {
				formatter.SetupCheck("Mistral API key", true, "configured")
			} else {
				formatter.SetupCheck("Mistral API key", false, "not set. Set MEETINGCLI_MISTRAL_API_KEY or add to config")
				allOk = false
			}

			if deps.Config.AnthropicKey != "" {
				formatter.SetupCheck("Anthropic API key", true, "configured")
			} else {
				formatter.SetupCheck("Anthropic API key", false, "not set. Set MEETINGCLI_ANTHROPIC_API_KEY or add to config")
				allOk = false
			}

			// Check meetings dir
			formatter.SetupCheck("Meetings directory", true, deps.Config.MeetingsDir)

			if allOk {
				formatter.Success("\nAll prerequisites met. Ready to record!")
			} else {
				formatter.Warning("\nSome prerequisites are missing. Fix them before recording.")
			}

			return nil
		},
	}

	return cmd
}
