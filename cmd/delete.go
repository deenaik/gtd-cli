package cmd

import (
	"fmt"
	"strconv"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id> [id...]",
	Short: "Delete tasks by ID",
	Long:  "Delete one or more tasks by their IDs. Requires confirmation before deleting.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskStore := store.NewTaskStore(db.Get())

		// Parse and validate all IDs first
		type taskEntry struct {
			id    int64
			title string
		}
		var toDelete []taskEntry

		for _, arg := range args {
			id, err := strconv.ParseInt(arg, 10, 64)
			if err != nil {
				ui.PrintError(fmt.Sprintf("Invalid task ID: %s", arg))
				continue
			}

			task, err := taskStore.GetByID(id)
			if err != nil {
				ui.PrintError(fmt.Sprintf("Task #%d not found", id))
				continue
			}

			toDelete = append(toDelete, taskEntry{id: id, title: task.Title})
		}

		if len(toDelete) == 0 {
			return fmt.Errorf("no valid tasks to delete")
		}

		// Show what will be deleted
		for _, entry := range toDelete {
			ui.PrintWarn(fmt.Sprintf("Will delete: #%d %s", entry.id, entry.title))
		}

		confirmed, err := ui.Confirm(fmt.Sprintf("Delete %d task(s)?", len(toDelete)))
		if err != nil {
			return fmt.Errorf("confirmation cancelled: %w", err)
		}
		if !confirmed {
			ui.PrintInfo("Cancelled.")
			return nil
		}

		for _, entry := range toDelete {
			if err := taskStore.Delete(entry.id); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to delete task #%d: %v", entry.id, err))
				continue
			}
			ui.PrintSuccess(fmt.Sprintf("Deleted: #%d %s", entry.id, entry.title))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
