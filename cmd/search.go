package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search tasks",
	Long:  "Full-text search across all tasks.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  searchRun,
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

func searchRun(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	ts := store.NewTaskStore(db.Get())
	tasks, err := ts.Search(query)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(tasks) == 0 {
		ui.PrintInfo(fmt.Sprintf("No results for '%s'.", query))
		return nil
	}

	ui.PrintInfo(fmt.Sprintf("Found %d result(s) for '%s':", len(tasks), query))
	fmt.Println()

	t := ui.NewTable("ID", "Title", "Category", "Status", "Project", "Context", "Due")
	for _, task := range tasks {
		due := ""
		if task.DueDate.Valid {
			due = task.DueDate.String
		}
		t.AddRow(
			strconv.FormatInt(task.ID, 10),
			task.Title,
			task.Category,
			task.Status,
			task.ProjectName,
			task.ContextName,
			due,
		)
	}
	t.Render()
	return nil
}
