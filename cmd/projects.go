package cmd

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/model"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage projects",
	Long:  "List, add, show, archive, and delete projects.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return projectsListRun(cmd, args)
	},
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE:  projectsListRun,
}

var projectsAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new project",
	Args:  cobra.MinimumNArgs(1),
	RunE:  projectsAddRun,
}

var projectsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show project details and its tasks",
	Args:  cobra.ExactArgs(1),
	RunE:  projectsShowRun,
}

var projectsArchiveCmd = &cobra.Command{
	Use:   "archive <id>",
	Short: "Archive a project (set status to completed)",
	Args:  cobra.ExactArgs(1),
	RunE:  projectsArchiveRun,
}

var projectsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a project",
	Args:  cobra.ExactArgs(1),
	RunE:  projectsDeleteRun,
}

func init() {
	rootCmd.AddCommand(projectsCmd)

	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsAddCmd)
	projectsCmd.AddCommand(projectsShowCmd)
	projectsCmd.AddCommand(projectsArchiveCmd)
	projectsCmd.AddCommand(projectsDeleteCmd)

	projectsAddCmd.Flags().String("area", "", "Area of responsibility")
	projectsAddCmd.Flags().String("due", "", "Due date (YYYY-MM-DD)")
	projectsAddCmd.Flags().String("description", "", "Project description")
}

func projectsListRun(cmd *cobra.Command, args []string) error {
	ps := store.NewProjectStore(db.Get())
	projects, err := ps.List("")
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		ui.PrintInfo("No projects found.")
		return nil
	}

	t := ui.NewTable("ID", "Name", "Status", "Area", "Tasks", "Due")
	for _, p := range projects {
		due := ""
		if p.DueDate.Valid {
			due = p.DueDate.String
		}
		t.AddRow(
			strconv.FormatInt(p.ID, 10),
			p.Name,
			p.Status,
			p.Area,
			strconv.Itoa(p.TaskCount),
			due,
		)
	}
	t.Render()
	return nil
}

func projectsAddRun(cmd *cobra.Command, args []string) error {
	name := args[0]
	area, _ := cmd.Flags().GetString("area")
	due, _ := cmd.Flags().GetString("due")
	description, _ := cmd.Flags().GetString("description")

	p := &model.Project{
		Name:        name,
		Description: description,
		Status:      "active",
		Area:        area,
	}
	if due != "" {
		p.DueDate = sql.NullString{String: due, Valid: true}
	}

	ps := store.NewProjectStore(db.Get())
	if err := ps.Create(p); err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Project created: %s (#%d)", p.Name, p.ID))
	return nil
}

func projectsShowRun(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid project ID: %s", args[0])
	}

	ps := store.NewProjectStore(db.Get())
	p, err := ps.GetByID(id)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	due := "-"
	if p.DueDate.Valid {
		due = p.DueDate.String
	}

	ui.PrintSection(p.Name)
	ui.PrintInfo(fmt.Sprintf("ID:          %d", p.ID))
	ui.PrintInfo(fmt.Sprintf("Status:      %s", p.Status))
	ui.PrintInfo(fmt.Sprintf("Area:        %s", p.Area))
	ui.PrintInfo(fmt.Sprintf("Due:         %s", due))
	ui.PrintInfo(fmt.Sprintf("Open Tasks:  %d", p.TaskCount))
	if p.Description != "" {
		ui.PrintInfo(fmt.Sprintf("Description: %s", p.Description))
	}

	// Show tasks belonging to this project
	ts := store.NewTaskStore(db.Get())
	tasks, err := ts.List(store.TaskFilter{ProjectID: id})
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	if len(tasks) > 0 {
		ui.PrintSection("Tasks")
		t := ui.NewTable("ID", "Title", "Category", "Status", "Priority", "Due")
		for _, task := range tasks {
			taskDue := ""
			if task.DueDate.Valid {
				taskDue = task.DueDate.String
			}
			t.AddRow(
				strconv.FormatInt(task.ID, 10),
				task.Title,
				task.Category,
				task.Status,
				task.PriorityLabel(),
				taskDue,
			)
		}
		t.Render()
	}

	return nil
}

func projectsArchiveRun(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid project ID: %s", args[0])
	}

	ps := store.NewProjectStore(db.Get())
	p, err := ps.GetByID(id)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	p.Status = "completed"
	if err := ps.Update(p); err != nil {
		return fmt.Errorf("failed to archive project: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Project archived: %s (#%d)", p.Name, p.ID))
	return nil
}

func projectsDeleteRun(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid project ID: %s", args[0])
	}

	ps := store.NewProjectStore(db.Get())
	p, err := ps.GetByID(id)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	confirmed, err := ui.Confirm(fmt.Sprintf("Delete project '%s' (#%d)?", p.Name, p.ID))
	if err != nil {
		return err
	}
	if !confirmed {
		ui.PrintWarn("Cancelled.")
		return nil
	}

	if err := ps.Delete(id); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Project deleted: %s (#%d)", p.Name, p.ID))
	return nil
}
