package cli

import (
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/devbydaniel/meetingcli/internal/output"
)

func NewListCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recorded meetings",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := output.NewFormatter(os.Stdout)

			entries, err := os.ReadDir(deps.Config.MeetingsDir)
			if err != nil {
				if os.IsNotExist(err) {
					formatter.Info("No meetings found")
					return nil
				}
				return err
			}

			// Filter to directories only
			var dirs []os.DirEntry
			for _, e := range entries {
				if e.IsDir() {
					dirs = append(dirs, e)
				}
			}

			if len(dirs) == 0 {
				formatter.Info("No meetings found")
				return nil
			}

			// Sort by name (which is date-based) descending
			sort.Slice(dirs, func(i, j int) bool {
				return dirs[i].Name() > dirs[j].Name()
			})

			formatter.MeetingListHeader()
			for _, d := range dirs {
				meetingPath := deps.Config.MeetingsDir + "/" + d.Name()
				_, transcriptErr := os.Stat(meetingPath + "/transcript.md")
				_, summaryErr := os.Stat(meetingPath + "/summary.md")
				formatter.MeetingListItem(d.Name(), transcriptErr == nil, summaryErr == nil)
			}

			return nil
		},
	}

	return cmd
}
