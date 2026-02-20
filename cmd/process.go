package cmd

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/model"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var processAI bool

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Interactively process inbox items",
	Long:  "Walk through unprocessed inbox items one by one and decide what to do with each.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if processAI {
			ui.PrintWarn("AI mode not yet available")
		}

		d := db.Get()
		inbox := store.NewInboxStore(d)
		tasks := store.NewTaskStore(d)
		projects := store.NewProjectStore(d)
		contexts := store.NewContextStore(d)

		items, err := inbox.ListUnprocessed()
		if err != nil {
			return fmt.Errorf("failed to list inbox: %w", err)
		}

		if len(items) == 0 {
			ui.PrintInfo("Inbox is empty. Nothing to process!")
			return nil
		}

		ui.PrintSection(fmt.Sprintf("Processing %d inbox items", len(items)))

		for i, item := range items {
			fmt.Println()
			ui.PrintInfo(fmt.Sprintf("[%d/%d] %s", i+1, len(items), item.Title))
			if item.Body != "" {
				fmt.Printf("  %s\n", item.Body)
			}

			choice, err := ui.SelectOne("What is it?", []string{
				"Actionable - Next Action",
				"Actionable - Waiting For",
				"Someday/Maybe",
				"Reference (delete)",
				"Trash (delete)",
			})
			if err != nil {
				return fmt.Errorf("selection cancelled: %w", err)
			}

			switch choice {
			case "Reference (delete)", "Trash (delete)":
				if err := inbox.MarkProcessed(item.ID); err != nil {
					return fmt.Errorf("failed to mark processed: %w", err)
				}
				if err := inbox.Delete(item.ID); err != nil {
					return fmt.Errorf("failed to delete item: %w", err)
				}
				ui.PrintSuccess(fmt.Sprintf("Deleted: %s", item.Title))
				continue

			case "Actionable - Next Action", "Actionable - Waiting For", "Someday/Maybe":
				task, err := promptForTask(item, choice, projects, contexts)
				if err != nil {
					return fmt.Errorf("task creation cancelled: %w", err)
				}
				task.Source = item.Source
				task.SourceRef = item.SourceRef

				if err := tasks.Create(task); err != nil {
					return fmt.Errorf("failed to create task: %w", err)
				}
				if err := inbox.MarkProcessed(item.ID); err != nil {
					return fmt.Errorf("failed to mark processed: %w", err)
				}
				ui.PrintSuccess(fmt.Sprintf("Created task #%d: %s", task.ID, task.Title))
			}
		}

		fmt.Println()
		ui.PrintSuccess("Inbox processing complete!")
		return nil
	},
}

func promptForTask(item model.InboxItem, choice string, projects *store.ProjectStore, contexts *store.ContextStore) (*model.Task, error) {
	// Determine category
	var category string
	switch choice {
	case "Actionable - Next Action":
		category = "next_action"
	case "Actionable - Waiting For":
		category = "waiting_for"
	case "Someday/Maybe":
		category = "someday_maybe"
	}

	// Title
	title, err := ui.Input("Title", item.Title)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(title) == "" {
		title = item.Title
	}

	task := &model.Task{
		Title:    title,
		Category: category,
		Status:   "pending",
	}

	// Project (optional)
	projectList, _ := projects.List("active")
	if len(projectList) > 0 {
		projectOptions := []string{"(none)"}
		for _, p := range projectList {
			projectOptions = append(projectOptions, p.Name)
		}
		projChoice, err := ui.SelectOne("Project (optional)", projectOptions)
		if err != nil {
			return nil, err
		}
		if projChoice != "(none)" {
			proj, err := projects.FindByName(projChoice)
			if err == nil {
				task.ProjectID = sql.NullInt64{Int64: proj.ID, Valid: true}
			}
		}
	}

	// Context (optional)
	contextList, _ := contexts.List()
	if len(contextList) > 0 {
		contextOptions := []string{"(none)"}
		for _, c := range contextList {
			contextOptions = append(contextOptions, c.Name)
		}
		ctxChoice, err := ui.SelectOne("Context (optional)", contextOptions)
		if err != nil {
			return nil, err
		}
		if ctxChoice != "(none)" {
			ctx, err := contexts.FindByName(ctxChoice)
			if err == nil {
				task.ContextID = sql.NullInt64{Int64: ctx.ID, Valid: true}
			}
		}
	}

	// Priority
	priChoice, err := ui.SelectOne("Priority", []string{
		"0 - None",
		"1 - Low",
		"2 - Medium",
		"3 - High",
		"4 - Urgent",
	})
	if err != nil {
		return nil, err
	}
	if len(priChoice) > 0 {
		pri, _ := strconv.Atoi(string(priChoice[0]))
		task.Priority = pri
	}

	// Due date
	due, err := ui.Input("Due date (YYYY-MM-DD, or leave blank)", "")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(due) != "" {
		task.DueDate = sql.NullString{String: due, Valid: true}
	}

	return task, nil
}

func init() {
	processCmd.Flags().BoolVar(&processAI, "ai", false, "Use AI to suggest categorization (placeholder)")
	rootCmd.AddCommand(processCmd)
}
