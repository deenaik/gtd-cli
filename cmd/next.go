package cmd

import (
	"fmt"
	"strconv"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	nextContext string
	nextProject string
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "List next actions",
	Long:  "Show all tasks categorized as next actions. Filter by context or project.",
	RunE: func(cmd *cobra.Command, args []string) error {
		d := db.Get()
		taskStore := store.NewTaskStore(d)

		filter := store.TaskFilter{
			Category: "next_action",
		}

		if nextContext != "" {
			ctxStore := store.NewContextStore(d)
			ctx, err := ctxStore.FindByName(nextContext)
			if err != nil {
				return fmt.Errorf("context %q not found: %w", nextContext, err)
			}
			filter.ContextID = ctx.ID
		}

		if nextProject != "" {
			projStore := store.NewProjectStore(d)
			proj, err := projStore.FindByName(nextProject)
			if err != nil {
				return fmt.Errorf("project %q not found: %w", nextProject, err)
			}
			filter.ProjectID = proj.ID
		}

		tasks, err := taskStore.List(filter)
		if err != nil {
			return fmt.Errorf("failed to list tasks: %w", err)
		}

		if len(tasks) == 0 {
			ui.PrintInfo("No next actions found.")
			return nil
		}

		table := ui.NewTable("ID", "Title", "Priority", "Context", "Project", "Due")
		for _, t := range tasks {
			due := ""
			if t.DueDate.Valid {
				due = t.DueDate.String
			}
			table.AddRow(
				strconv.FormatInt(t.ID, 10),
				t.Title,
				t.PriorityLabel(),
				t.ContextName,
				t.ProjectName,
				due,
			)
		}
		table.Render()

		return nil
	},
}

func init() {
	nextCmd.Flags().StringVar(&nextContext, "context", "", "Filter by context name")
	nextCmd.Flags().StringVar(&nextProject, "project", "", "Filter by project name")
	rootCmd.AddCommand(nextCmd)
}
