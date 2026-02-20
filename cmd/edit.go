package cmd

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a task",
	Long:  "Edit an existing task's fields. Only flags that are explicitly set will be updated.",
	Args:  cobra.ExactArgs(1),
	RunE:  editRun,
}

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().String("title", "", "New title")
	editCmd.Flags().String("description", "", "New description")
	editCmd.Flags().String("status", "", "New status (pending/active/done/delegated/deferred)")
	editCmd.Flags().String("category", "", "New category (next_action/waiting_for/someday_maybe)")
	editCmd.Flags().String("context", "", "Context name (looked up by name)")
	editCmd.Flags().String("project", "", "Project name (looked up by name)")
	editCmd.Flags().String("due", "", "Due date (YYYY-MM-DD, or empty to clear)")
	editCmd.Flags().Int("priority", -1, "Priority (0=none, 1=low, 2=medium, 3=high, 4=urgent)")
	editCmd.Flags().String("energy", "", "Energy level (low/medium/high)")
	editCmd.Flags().String("delegate", "", "Delegated-to person")
	editCmd.Flags().String("defer", "", "Defer until date (YYYY-MM-DD, or empty to clear)")
}

func editRun(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid task ID: %s", args[0])
	}

	d := db.Get()
	ts := store.NewTaskStore(d)
	task, err := ts.GetByID(id)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	changed := false

	if cmd.Flags().Changed("title") {
		task.Title, _ = cmd.Flags().GetString("title")
		changed = true
	}
	if cmd.Flags().Changed("description") {
		task.Description, _ = cmd.Flags().GetString("description")
		changed = true
	}
	if cmd.Flags().Changed("status") {
		task.Status, _ = cmd.Flags().GetString("status")
		changed = true
	}
	if cmd.Flags().Changed("category") {
		task.Category, _ = cmd.Flags().GetString("category")
		changed = true
	}
	if cmd.Flags().Changed("context") {
		ctxName, _ := cmd.Flags().GetString("context")
		if ctxName == "" {
			task.ContextID = sql.NullInt64{}
		} else {
			cs := store.NewContextStore(d)
			ctx, err := cs.FindByName(ctxName)
			if err != nil {
				return fmt.Errorf("context '%s' not found: %w", ctxName, err)
			}
			task.ContextID = sql.NullInt64{Int64: ctx.ID, Valid: true}
		}
		changed = true
	}
	if cmd.Flags().Changed("project") {
		projName, _ := cmd.Flags().GetString("project")
		if projName == "" {
			task.ProjectID = sql.NullInt64{}
		} else {
			ps := store.NewProjectStore(d)
			proj, err := ps.FindByName(projName)
			if err != nil {
				return fmt.Errorf("project '%s' not found: %w", projName, err)
			}
			task.ProjectID = sql.NullInt64{Int64: proj.ID, Valid: true}
		}
		changed = true
	}
	if cmd.Flags().Changed("due") {
		due, _ := cmd.Flags().GetString("due")
		if due == "" {
			task.DueDate = sql.NullString{}
		} else {
			task.DueDate = sql.NullString{String: due, Valid: true}
		}
		changed = true
	}
	if cmd.Flags().Changed("priority") {
		task.Priority, _ = cmd.Flags().GetInt("priority")
		changed = true
	}
	if cmd.Flags().Changed("energy") {
		task.Energy, _ = cmd.Flags().GetString("energy")
		changed = true
	}
	if cmd.Flags().Changed("delegate") {
		task.DelegatedTo, _ = cmd.Flags().GetString("delegate")
		changed = true
	}
	if cmd.Flags().Changed("defer") {
		deferDate, _ := cmd.Flags().GetString("defer")
		if deferDate == "" {
			task.DeferUntil = sql.NullString{}
		} else {
			task.DeferUntil = sql.NullString{String: deferDate, Valid: true}
		}
		changed = true
	}

	if !changed {
		ui.PrintWarn("No changes specified. Use flags to update fields (e.g. --title, --status).")
		return nil
	}

	if err := ts.Update(task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Task updated: %s (#%d)", task.Title, task.ID))
	return nil
}
