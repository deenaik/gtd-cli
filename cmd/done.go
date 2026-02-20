package cmd

import (
	"fmt"
	"strconv"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done <id> [id...]",
	Short: "Mark tasks as done",
	Long:  "Mark one or more tasks as completed by their IDs.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskStore := store.NewTaskStore(db.Get())

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

			if err := taskStore.MarkDone(id); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to complete task #%d: %v", id, err))
				continue
			}

			ui.PrintSuccess(fmt.Sprintf("Completed: #%d %s", id, task.Title))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(doneCmd)
}
