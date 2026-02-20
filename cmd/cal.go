package cmd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/deenaik/gtd-cli/internal/ai"
	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/google"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var calCmd = &cobra.Command{
	Use:   "cal",
	Short: "Google Calendar integration",
	Long:  "View calendar events, check availability, create events, and AI-schedule tasks.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default: show today's events
		return calTodayRun(cmd, args)
	},
}

// --- cal today ---

var calTodayCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's events",
	RunE:  calTodayRun,
}

func calTodayRun(cmd *cobra.Command, args []string) error {
	today := time.Now().Format("2006-01-02")
	events, err := google.ListEvents(today, today)
	if err != nil {
		return fmt.Errorf("failed to list events: %w", err)
	}

	ui.PrintSection(fmt.Sprintf("Calendar - %s", time.Now().Format("Monday, January 2, 2006")))

	if len(events) == 0 {
		ui.PrintInfo("No events today.")
		return nil
	}

	table := ui.NewTable("Time", "Title", "Location", "ID")
	for _, e := range events {
		id := e.ID
		if len(id) > 8 {
			id = id[:8]
		}
		table.AddRow(formatEventTime(e.Start.String(), e.End.String()), e.Title, e.Location, id)
	}
	table.Render()
	return nil
}

// --- cal week ---

var calWeekCmd = &cobra.Command{
	Use:   "week",
	Short: "Show this week's events (Mon-Sun)",
	RunE: func(cmd *cobra.Command, args []string) error {
		now := time.Now()
		// Find Monday of this week
		weekday := now.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		monday := now.AddDate(0, 0, -int(weekday)+int(time.Monday))
		sunday := monday.AddDate(0, 0, 6)

		from := monday.Format("2006-01-02")
		to := sunday.Format("2006-01-02")

		events, err := google.ListEvents(from, to)
		if err != nil {
			return fmt.Errorf("failed to list events: %w", err)
		}

		ui.PrintSection(fmt.Sprintf("Calendar - Week of %s to %s", from, to))

		if len(events) == 0 {
			ui.PrintInfo("No events this week.")
			return nil
		}

		table := ui.NewTable("Date", "Time", "Title", "Location")
		for _, e := range events {
			date := ""
			startStr := e.Start.String()
			if len(startStr) >= 10 {
				date = startStr[:10]
			}
			table.AddRow(date, formatEventTime(startStr, e.End.String()), e.Title, e.Location)
		}
		table.Render()
		return nil
	},
}

// --- cal create ---

var calCreateTitle string
var calCreateFrom string
var calCreateTo string

var calCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a calendar event",
	RunE: func(cmd *cobra.Command, args []string) error {
		if calCreateTitle == "" || calCreateFrom == "" || calCreateTo == "" {
			return fmt.Errorf("--title, --from, and --to are required")
		}

		event, err := google.CreateEvent(calCreateTitle, calCreateFrom, calCreateTo)
		if err != nil {
			return fmt.Errorf("failed to create event: %w", err)
		}

		ui.PrintSuccess(fmt.Sprintf("Created event: %s (%s - %s)", event.Title, event.Start.String(), event.End.String()))
		return nil
	},
}

// --- cal freebusy ---

var calFreeBusyFrom string
var calFreeBusyTo string

var calFreeBusyCmd = &cobra.Command{
	Use:   "freebusy",
	Short: "Show free/busy time slots",
	RunE: func(cmd *cobra.Command, args []string) error {
		from := calFreeBusyFrom
		to := calFreeBusyTo

		// Default: today 9am-5pm
		if from == "" {
			from = time.Now().Format("2006-01-02") + "T09:00:00"
		}
		if to == "" {
			to = time.Now().Format("2006-01-02") + "T17:00:00"
		}

		slots, err := google.FreeBusy(from, to)
		if err != nil {
			return fmt.Errorf("failed to query free/busy: %w", err)
		}

		ui.PrintSection(fmt.Sprintf("Free/Busy - %s to %s", from, to))

		if len(slots) == 0 {
			ui.PrintInfo("No busy slots found. You're free!")
			return nil
		}

		table := ui.NewTable("Start", "End")
		for _, s := range slots {
			table.AddRow(s.Start, s.End)
		}
		table.Render()
		return nil
	},
}

// --- cal schedule ---

var calScheduleCmd = &cobra.Command{
	Use:   "schedule <task-id>",
	Short: "AI suggest best time to schedule a task",
	Args:  cobra.ExactArgs(1),
	RunE:  calScheduleRun,
}

func calScheduleRun(cmd *cobra.Command, args []string) error {
	taskID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid task ID: %w", err)
	}

	ts := store.NewTaskStore(db.Get())
	task, err := ts.GetByID(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Get free/busy for next 3 days
	now := time.Now()
	from := now.Format("2006-01-02") + "T09:00:00"
	to := now.AddDate(0, 0, 3).Format("2006-01-02") + "T17:00:00"

	busySlots, err := google.FreeBusy(from, to)
	if err != nil {
		return fmt.Errorf("failed to query free/busy: %w", err)
	}

	// Convert google.FreeBusySlot to ai.BusySlot
	var aiBusy []ai.BusySlot
	for _, s := range busySlots {
		aiBusy = append(aiBusy, ai.BusySlot{Start: s.Start, End: s.End})
	}

	dueDate := ""
	if task.DueDate.Valid {
		dueDate = task.DueDate.String
	}

	aiTask := ai.TaskForSchedule{
		Title:        task.Title,
		Description:  task.Description,
		Priority:     task.Priority,
		Energy:       task.Energy,
		TimeEstimate: task.TimeEstimate,
		DueDate:      dueDate,
		Context:      task.ContextName,
	}

	ui.PrintInfo(fmt.Sprintf("Finding best time for: %s", task.Title))

	suggestion, err := ai.SuggestSchedule(context.Background(), aiTask, aiBusy)
	if err != nil {
		return fmt.Errorf("AI scheduling failed: %w", err)
	}

	ui.PrintSection("Schedule Suggestion")

	if suggestion.Start != "" && suggestion.End != "" {
		fmt.Printf("Suggested time: %s - %s\n", suggestion.Start, suggestion.End)
	}
	if suggestion.Reasoning != "" {
		fmt.Printf("Reasoning: %s\n", suggestion.Reasoning)
	}
	fmt.Println()

	// Offer to create the event
	if suggestion.Start != "" && suggestion.End != "" {
		confirmed, err := ui.Confirm(fmt.Sprintf("Create event for %s - %s?", suggestion.Start, suggestion.End))
		if err != nil {
			return nil
		}
		if confirmed {
			event, err := google.CreateEvent(task.Title, suggestion.Start, suggestion.End)
			if err != nil {
				return fmt.Errorf("failed to create event: %w", err)
			}
			// Update task with calendar event ID
			task.CalendarEventID = event.ID
			if updateErr := ts.Update(task); updateErr != nil {
				ui.PrintWarn(fmt.Sprintf("Event created but failed to update task: %v", updateErr))
			}
			ui.PrintSuccess(fmt.Sprintf("Created event: %s (%s - %s)", event.Title, event.Start.String(), event.End.String()))
		}
	}

	return nil
}

// formatEventTime formats start/end times for display.
func formatEventTime(start, end string) string {
	startTime := extractTime(start)
	endTime := extractTime(end)
	if startTime != "" && endTime != "" {
		return fmt.Sprintf("%s - %s", startTime, endTime)
	}
	if startTime != "" {
		return startTime
	}
	return start
}

func extractTime(datetime string) string {
	// Handle RFC3339 first
	t, err := time.Parse(time.RFC3339, datetime)
	if err == nil {
		return t.Format("15:04")
	}
	// Try datetime without timezone
	t, err = time.Parse("2006-01-02T15:04:05", datetime)
	if err == nil {
		return t.Format("15:04")
	}
	// If it's just a date (all-day event), return "all day"
	if len(datetime) == 10 {
		return "all day"
	}
	return ""
}

func init() {
	// cal create flags
	calCreateCmd.Flags().StringVar(&calCreateTitle, "title", "", "Event title (required)")
	calCreateCmd.Flags().StringVar(&calCreateFrom, "from", "", "Start datetime, e.g. 2024-01-15T09:00:00 (required)")
	calCreateCmd.Flags().StringVar(&calCreateTo, "to", "", "End datetime, e.g. 2024-01-15T10:00:00 (required)")
	_ = calCreateCmd.MarkFlagRequired("title")
	_ = calCreateCmd.MarkFlagRequired("from")
	_ = calCreateCmd.MarkFlagRequired("to")

	// cal freebusy flags
	calFreeBusyCmd.Flags().StringVar(&calFreeBusyFrom, "from", "", "Start datetime (default: today 9am)")
	calFreeBusyCmd.Flags().StringVar(&calFreeBusyTo, "to", "", "End datetime (default: today 5pm)")

	// Register subcommands
	calCmd.AddCommand(calTodayCmd)
	calCmd.AddCommand(calWeekCmd)
	calCmd.AddCommand(calCreateCmd)
	calCmd.AddCommand(calFreeBusyCmd)
	calCmd.AddCommand(calScheduleCmd)

	rootCmd.AddCommand(calCmd)
}
