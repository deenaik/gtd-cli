package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's dashboard",
	Long:  "Display overdue tasks, tasks due today, top next actions, and inbox count.",
	RunE:  todayRun,
}

func init() {
	rootCmd.AddCommand(todayCmd)
	todayCmd.Flags().Bool("tomorrow", false, "Show dashboard for tomorrow instead of today")
}

func todayRun(cmd *cobra.Command, args []string) error {
	targetDate := time.Now()

	// Check --tomorrow flag; must guard against nil Flags (when called from rootCmd default)
	if cmd.Flags() != nil {
		if tomorrow, _ := cmd.Flags().GetBool("tomorrow"); tomorrow {
			targetDate = targetDate.AddDate(0, 0, 1)
		}
	}

	dateStr := targetDate.Format("2006-01-02")
	displayDate := targetDate.Format("Monday, January 2, 2006")

	d := db.Get()
	ts := store.NewTaskStore(d)
	is := store.NewInboxStore(d)

	// Header
	ui.PrintSection(fmt.Sprintf("GTD Dashboard - %s", displayDate))

	// 1. Overdue section: tasks with due_date < today and status != done
	overdueTasks, err := ts.List(store.TaskFilter{
		DueBefore: dateStr,
	})
	if err != nil {
		return fmt.Errorf("failed to list overdue tasks: %w", err)
	}
	// Filter out tasks due exactly on the target date (DueBefore uses <=),
	// so we only keep tasks with due_date strictly before the target date.
	var overdue []struct {
		id       int64
		title    string
		due      string
		project  string
		priority string
	}
	for _, task := range overdueTasks {
		if task.DueDate.Valid && task.DueDate.String < dateStr {
			overdue = append(overdue, struct {
				id       int64
				title    string
				due      string
				project  string
				priority string
			}{task.ID, task.Title, task.DueDate.String, task.ProjectName, task.PriorityLabel()})
		}
	}

	ui.PrintSection("Overdue")
	if len(overdue) == 0 {
		ui.PrintInfo("No overdue tasks.")
	} else {
		t := ui.NewTable("ID", "Title", "Due", "Project", "Priority")
		for _, o := range overdue {
			t.AddRow(
				strconv.FormatInt(o.id, 10),
				o.title,
				o.due,
				o.project,
				o.priority,
			)
		}
		t.Render()
	}

	// 2. Due Today section: tasks due exactly on the target date
	dueTodayTasks, err := ts.List(store.TaskFilter{
		DueBefore: dateStr,
	})
	if err != nil {
		return fmt.Errorf("failed to list due-today tasks: %w", err)
	}
	var dueToday []struct {
		id       int64
		title    string
		project  string
		context  string
		priority string
	}
	for _, task := range dueTodayTasks {
		if task.DueDate.Valid && task.DueDate.String == dateStr {
			dueToday = append(dueToday, struct {
				id       int64
				title    string
				project  string
				context  string
				priority string
			}{task.ID, task.Title, task.ProjectName, task.ContextName, task.PriorityLabel()})
		}
	}

	ui.PrintSection("Due Today")
	if len(dueToday) == 0 {
		ui.PrintInfo("No tasks due today.")
	} else {
		t := ui.NewTable("ID", "Title", "Project", "Context", "Priority")
		for _, d := range dueToday {
			t.AddRow(
				strconv.FormatInt(d.id, 10),
				d.title,
				d.project,
				d.context,
				d.priority,
			)
		}
		t.Render()
	}

	// 3. Next Actions section: top 10 next actions sorted by priority
	nextActions, err := ts.List(store.TaskFilter{
		Category: "next_action",
	})
	if err != nil {
		return fmt.Errorf("failed to list next actions: %w", err)
	}

	ui.PrintSection("Next Actions")
	if len(nextActions) == 0 {
		ui.PrintInfo("No next actions.")
	} else {
		limit := 10
		if len(nextActions) < limit {
			limit = len(nextActions)
		}
		t := ui.NewTable("ID", "Title", "Priority", "Project", "Context", "Due")
		for _, task := range nextActions[:limit] {
			due := ""
			if task.DueDate.Valid {
				due = task.DueDate.String
			}
			t.AddRow(
				strconv.FormatInt(task.ID, 10),
				task.Title,
				task.PriorityLabel(),
				task.ProjectName,
				task.ContextName,
				due,
			)
		}
		t.Render()
	}

	// 4. Inbox section: count of unprocessed items
	inboxCount, err := is.Count()
	if err != nil {
		return fmt.Errorf("failed to count inbox: %w", err)
	}

	ui.PrintSection("Inbox")
	if inboxCount == 0 {
		ui.PrintInfo("Inbox is empty.")
	} else {
		ui.PrintWarn(fmt.Sprintf("%d unprocessed item(s) in inbox.", inboxCount))
	}

	fmt.Println()
	return nil
}
