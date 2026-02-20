package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
)

// dailySummaryStartHour and dailySummaryEndHour define the window (17:00 -
// 18:00) in which the daily summary notification is sent.
const (
	dailySummaryStartHour = 17
	dailySummaryEndHour   = 18
)

// RunProgressMonitor snapshots key productivity metrics and stores them in the
// progress_metrics table. Once per day, during the 17:00-18:00 window, it
// generates a daily summary notification.
func RunProgressMonitor(ctx context.Context) error {
	d := db.Get()
	tasks := store.NewTaskStore(d)
	inbox := store.NewInboxStore(d)

	// Calculate today's start for queries.
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayStr := todayStart.Format("2006-01-02 15:04:05")

	// Count tasks completed today.
	completedToday, err := tasks.CompletedSince(todayStr)
	if err != nil {
		slog.Warn("progress monitor: count completed today", "error", err)
		completedToday = 0
	}

	// Count pending tasks.
	pendingCount, err := queryScalarInt(d,
		`SELECT COUNT(*) FROM tasks WHERE status IN ('pending', 'active')`)
	if err != nil {
		slog.Warn("progress monitor: count pending", "error", err)
		pendingCount = 0
	}

	// Count overdue tasks.
	overdueCount, err := queryScalarInt(d,
		`SELECT COUNT(*) FROM tasks WHERE status IN ('pending', 'active') AND due_date IS NOT NULL AND due_date < date('now')`)
	if err != nil {
		slog.Warn("progress monitor: count overdue", "error", err)
		overdueCount = 0
	}

	// Count unprocessed inbox items.
	inboxCount, err := inbox.Count()
	if err != nil {
		slog.Warn("progress monitor: count inbox", "error", err)
		inboxCount = 0
	}

	// Store the snapshot.
	_, err = d.Exec(`
		INSERT INTO progress_metrics (completed_today, pending_count, overdue_count, inbox_count, avg_completion_rate)
		VALUES (?, ?, ?, ?, ?)`,
		completedToday, pendingCount, overdueCount, inboxCount, calculateCompletionRate(),
	)
	if err != nil {
		slog.Error("progress monitor: insert metrics", "error", err)
		// Continue to try the daily summary even if metrics storage fails.
	}

	slog.Info("progress monitor: snapshot recorded",
		"completed_today", completedToday,
		"pending", pendingCount,
		"overdue", overdueCount,
		"inbox", inboxCount,
	)

	// Check if we should send the daily summary (17:00-18:00 window).
	if now.Hour() >= dailySummaryStartHour && now.Hour() < dailySummaryEndHour {
		sendDailySummary(ctx, completedToday, pendingCount, overdueCount, inboxCount)
	}

	return nil
}

// sendDailySummary generates and sends the end-of-day summary notification.
// It deduplicates using the notifications_sent table so the summary is only
// sent once per day.
func sendDailySummary(ctx context.Context, completed, pending, overdue, inbox int) {
	today := time.Now().Format("2006-01-02")
	summaryKey := fmt.Sprintf("daily-summary-%s", today)

	if hasNotificationBeenSent(summaryKey, "daily_summary", 0) {
		return
	}

	title := "GTD Daily Summary"
	message := fmt.Sprintf(
		"Completed today: %d\nPending tasks: %d\nOverdue: %d\nInbox items: %d",
		completed, pending, overdue, inbox,
	)

	if overdue > 0 {
		message += fmt.Sprintf("\n\nYou have %d overdue task(s) that need attention.", overdue)
	}

	if inbox > 0 {
		message += fmt.Sprintf("\n%d unprocessed inbox item(s) to review.", inbox)
	}

	if completed == 0 && pending > 0 {
		message += "\n\nNo tasks completed today. Consider tackling a quick win!"
	} else if completed >= 5 {
		message += "\n\nGreat productivity today!"
	}

	NotifyAll(title, message)
	recordNotificationSent(summaryKey, "daily_summary")

	slog.Info("progress monitor: daily summary sent",
		"completed", completed, "pending", pending, "overdue", overdue, "inbox", inbox)
}

// calculateCompletionRate computes the average daily completion rate over the
// last 7 days.
func calculateCompletionRate() float64 {
	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02 15:04:05")

	d := db.Get()
	var total int
	err := d.QueryRow(`
		SELECT COUNT(*) FROM tasks WHERE status = 'done' AND completed_at >= ?`, weekAgo).Scan(&total)
	if err != nil || total == 0 {
		return 0.0
	}

	return float64(total) / 7.0
}
