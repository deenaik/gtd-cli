package cmd

import (
	"fmt"
	"strconv"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var somedayCmd = &cobra.Command{
	Use:   "someday",
	Short: "List someday/maybe tasks",
	Long:  "Show tasks in the someday/maybe category.",
	RunE:  somedayRun,
}

func init() {
	rootCmd.AddCommand(somedayCmd)
}

func somedayRun(cmd *cobra.Command, args []string) error {
	ts := store.NewTaskStore(db.Get())
	tasks, err := ts.List(store.TaskFilter{
		Category: "someday_maybe",
	})
	if err != nil {
		return fmt.Errorf("failed to list someday/maybe tasks: %w", err)
	}

	if len(tasks) == 0 {
		ui.PrintInfo("No someday/maybe tasks found.")
		return nil
	}

	t := ui.NewTable("ID", "Title", "Project", "Created")
	for _, task := range tasks {
		t.AddRow(
			strconv.FormatInt(task.ID, 10),
			task.Title,
			task.ProjectName,
			task.CreatedAt.Format("2006-01-02"),
		)
	}
	t.Render()
	return nil
}
