package cmd

import (
	"fmt"
	"strconv"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var waitingCmd = &cobra.Command{
	Use:   "waiting",
	Short: "List waiting-for tasks",
	Long:  "Show tasks that are waiting for someone else (category=waiting_for).",
	RunE:  waitingRun,
}

func init() {
	rootCmd.AddCommand(waitingCmd)
	waitingCmd.Flags().String("who", "", "Filter by delegated-to person")
}

func waitingRun(cmd *cobra.Command, args []string) error {
	who, _ := cmd.Flags().GetString("who")

	ts := store.NewTaskStore(db.Get())
	tasks, err := ts.List(store.TaskFilter{
		Category: "waiting_for",
		Who:      who,
	})
	if err != nil {
		return fmt.Errorf("failed to list waiting-for tasks: %w", err)
	}

	if len(tasks) == 0 {
		ui.PrintInfo("No waiting-for tasks found.")
		return nil
	}

	t := ui.NewTable("ID", "Title", "Delegated To", "Project", "Due", "Created")
	for _, task := range tasks {
		due := ""
		if task.DueDate.Valid {
			due = task.DueDate.String
		}
		t.AddRow(
			strconv.FormatInt(task.ID, 10),
			task.Title,
			task.DelegatedTo,
			task.ProjectName,
			due,
			task.CreatedAt.Format("2006-01-02"),
		)
	}
	t.Render()
	return nil
}
