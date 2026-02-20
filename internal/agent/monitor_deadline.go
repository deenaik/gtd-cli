package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
)

// Deadline thresholds.
const (
	deadlineWarningWindow = 24 * time.Hour
	overdueRenotifyWindow = 4 * time.Hour
)

// RunDeadlineMonitor scans for tasks with approaching or overdue deadlines
// and sends tiered notifications:
//   - 24-hour warning for tasks due within the next day
//   - Overdue notification for tasks past their due date (re-notifies every 4h)
func RunDeadlineMonitor(ctx context.Context) error {
	d := db.Get()
	tasks := store.NewTaskStore(d)

	// Find tasks with a due_date that are not yet completed.
	dueTasks, err := tasks.List(store.TaskFilter{
		Status: "pending",
	})
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}

	now := time.Now()
	todayStr := now.Format("2006-01-02")
	tomorrowStr := now.Add(deadlineWarningWindow).Format("2006-01-02")

	var warningCount, overdueCount int

	for _, t := range dueTasks {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Skip tasks without a due date.
		if !t.DueDate.Valid || t.DueDate.String == "" {
			continue
		}

		dueDate := t.DueDate.String
		taskKey := fmt.Sprintf("task-%d", t.ID)

		// Parse the due date for comparison.
		dueTime, err := time.ParseInLocation("2006-01-02", dueDate, time.Local)
		if err != nil {
			slog.Warn("deadline monitor: parse due date", "task_id", t.ID, "due_date", dueDate, "error", err)
			continue
		}

		// Check if overdue (due date is before today).
		if dueDate < todayStr {
			overdueCount++

			notifyType := "deadline_overdue"
			// For overdue tasks, re-notify every 4 hours.
			if hasNotificationBeenSent(taskKey, notifyType, overdueRenotifyWindow) {
				continue
			}

			daysPast := int(now.Sub(dueTime).Hours() / 24)
			daysLabel := "1 day"
			if daysPast > 1 {
				daysLabel = fmt.Sprintf("%d days", daysPast)
			}

			title := fmt.Sprintf("OVERDUE: %s", t.Title)
			message := fmt.Sprintf("Task was due %s (%s ago)", dueDate, daysLabel)
			if t.ProjectName != "" {
				message += fmt.Sprintf("\nProject: %s", t.ProjectName)
			}

			NotifyAll(title, message)
			recordNotificationSent(taskKey, notifyType)

			slog.Info("deadline monitor: overdue notification",
				"task_id", t.ID, "title", t.Title, "due_date", dueDate)
			continue
		}

		// Check if due within the next 24 hours (due today or tomorrow).
		if dueDate >= todayStr && dueDate <= tomorrowStr {
			warningCount++

			notifyType := "deadline_warning"
			if hasNotificationBeenSent(taskKey, notifyType, 0) {
				continue
			}

			hoursLeft := time.Until(dueTime.Add(24 * time.Hour)).Hours()
			var timeLabel string
			if dueDate == todayStr {
				timeLabel = "today"
			} else {
				timeLabel = fmt.Sprintf("in %.0f hours", hoursLeft)
			}

			title := fmt.Sprintf("Deadline: %s", t.Title)
			message := fmt.Sprintf("Due %s (%s)", timeLabel, dueDate)
			if t.ProjectName != "" {
				message += fmt.Sprintf("\nProject: %s", t.ProjectName)
			}
			if t.ContextName != "" {
				message += fmt.Sprintf("\nContext: @%s", t.ContextName)
			}

			NotifyAll(title, message)
			recordNotificationSent(taskKey, notifyType)

			slog.Info("deadline monitor: warning notification",
				"task_id", t.ID, "title", t.Title, "due_date", dueDate, "time_label", timeLabel)
		}
	}

	slog.Info("deadline monitor: scan complete",
		"warnings", warningCount, "overdue", overdueCount)

	return nil
}
