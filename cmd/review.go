package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/deenaik/gtd-cli/internal/ai"
	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Weekly review workflow",
	RunE:  reviewStartRun,
}

var reviewStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a guided weekly review",
	RunE:  reviewStartRun,
}

var reviewStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show review status and history",
	RunE:  reviewStatusRun,
}

var reviewHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show review history",
	RunE:  reviewHistoryRun,
}

var reviewAI bool

func init() {
	rootCmd.AddCommand(reviewCmd)
	reviewCmd.AddCommand(reviewStartCmd)
	reviewCmd.AddCommand(reviewStatusCmd)
	reviewCmd.AddCommand(reviewHistoryCmd)
	reviewStartCmd.Flags().BoolVar(&reviewAI, "ai", false, "Generate AI insights")
	reviewCmd.Flags().BoolVar(&reviewAI, "ai", false, "Generate AI insights")
}

func reviewStartRun(cmd *cobra.Command, args []string) error {
	d := db.Get()
	taskStore := store.NewTaskStore(d)
	inboxStore := store.NewInboxStore(d)
	projectStore := store.NewProjectStore(d)
	reviewStore := store.NewReviewStore(d)

	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday())+1) // Monday
	if now.Weekday() == time.Sunday {
		weekStart = now.AddDate(0, 0, -6)
	}
	weekStartStr := weekStart.Format("2006-01-02")

	fmt.Println()
	fmt.Println(ui.BoldStyle.Render("=== Weekly Review ==="))
	fmt.Printf("Week of %s\n\n", weekStartStr)

	// Step 1: Inbox check
	ui.PrintSection("Step 1: Clear Your Inbox")
	inboxCount, _ := inboxStore.Count()
	if inboxCount > 0 {
		ui.PrintWarn(fmt.Sprintf("You have %d unprocessed inbox items. Process them first with 'gtd process'.", inboxCount))
	} else {
		ui.PrintSuccess("Inbox is empty!")
	}

	// Step 2: Review next actions
	ui.PrintSection("Step 2: Review Next Actions")
	nextActions, _ := taskStore.List(store.TaskFilter{Category: "next_action"})
	fmt.Printf("  %d active next actions\n", len(nextActions))
	for _, t := range nextActions {
		status := "  "
		if t.DueDate.Valid && t.DueDate.String < now.Format("2006-01-02") {
			status = ui.ErrorStyle.Render("OVERDUE")
		}
		fmt.Printf("  [%d] %s %s\n", t.ID, t.Title, status)
	}

	// Step 3: Review waiting-for
	ui.PrintSection("Step 3: Review Waiting-For Items")
	waiting, _ := taskStore.List(store.TaskFilter{Category: "waiting_for"})
	if len(waiting) == 0 {
		fmt.Println("  No waiting-for items.")
	}
	for _, t := range waiting {
		fmt.Printf("  [%d] %s (from: %s)\n", t.ID, t.Title, t.DelegatedTo)
	}

	// Step 4: Review projects
	ui.PrintSection("Step 4: Review Projects")
	projects, _ := projectStore.List("active")
	for _, p := range projects {
		fmt.Printf("  [%d] %s (%d open tasks)\n", p.ID, p.Name, p.TaskCount)
	}

	// Step 5: Review someday/maybe
	ui.PrintSection("Step 5: Review Someday/Maybe")
	someday, _ := taskStore.List(store.TaskFilter{Category: "someday_maybe"})
	fmt.Printf("  %d items in someday/maybe list\n", len(someday))

	// Step 6: Stats
	ui.PrintSection("Step 6: Weekly Stats")
	completedCount, _ := taskStore.CompletedSince(weekStartStr)
	createdCount, _ := taskStore.CreatedSince(weekStartStr)
	fmt.Printf("  Tasks completed this week: %d\n", completedCount)
	fmt.Printf("  Tasks created this week:   %d\n", createdCount)

	// Capture notes
	notes, err := ui.TextArea("Any notes or reflections for this week?", "Optional notes...")
	if err != nil {
		notes = ""
	}

	// AI insights
	var insights string
	if reviewAI {
		ui.PrintSection("Generating AI Insights...")
		stats := ai.WeeklyStats{
			CompletedCount:   completedCount,
			CreatedCount:     createdCount,
			PendingCount:     len(nextActions),
			WaitingCount:     len(waiting),
			SomedayCount:     len(someday),
			ProjectCount:     len(projects),
			InboxCount:       inboxCount,
		}
		insights, err = ai.WeeklyInsights(context.Background(), stats)
		if err != nil {
			ui.PrintWarn("AI insights unavailable: " + err.Error())
		} else {
			fmt.Println()
			fmt.Println(insights)
		}
	}

	// Save review
	review := &store.WeeklyReview{
		WeekStart:      weekStartStr,
		TasksCompleted: completedCount,
		TasksCreated:   createdCount,
		Notes:          notes,
		AIInsights:     insights,
		CompletedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	}
	if err := reviewStore.Create(review); err != nil {
		return fmt.Errorf("failed to save review: %w", err)
	}

	fmt.Println()
	ui.PrintSuccess("Weekly review completed and saved!")
	return nil
}

func reviewStatusRun(cmd *cobra.Command, args []string) error {
	d := db.Get()
	reviewStore := store.NewReviewStore(d)

	latest, err := reviewStore.GetLatest()
	if err != nil {
		fmt.Println("No reviews found. Start one with 'gtd review start'.")
		return nil
	}

	fmt.Println()
	fmt.Println(ui.BoldStyle.Render("Latest Review"))
	fmt.Printf("  Week:      %s\n", latest.WeekStart)
	fmt.Printf("  Completed: %d tasks\n", latest.TasksCompleted)
	fmt.Printf("  Created:   %d tasks\n", latest.TasksCreated)
	if latest.CompletedAt.Valid {
		fmt.Printf("  Done at:   %s\n", latest.CompletedAt.Time.Format("2006-01-02 15:04"))
	}
	if latest.Notes != "" {
		fmt.Printf("  Notes:     %s\n", latest.Notes)
	}
	return nil
}

func reviewHistoryRun(cmd *cobra.Command, args []string) error {
	d := db.Get()
	reviewStore := store.NewReviewStore(d)

	reviews, err := reviewStore.List(10)
	if err != nil || len(reviews) == 0 {
		fmt.Println("No review history found.")
		return nil
	}

	tbl := ui.NewTable("Week", "Completed", "Created", "Date")
	for _, r := range reviews {
		doneAt := "-"
		if r.CompletedAt.Valid {
			doneAt = r.CompletedAt.Time.Format("2006-01-02")
		}
		tbl.AddRow(
			r.WeekStart,
			fmt.Sprintf("%d", r.TasksCompleted),
			fmt.Sprintf("%d", r.TasksCreated),
			doneAt,
		)
	}
	tbl.Render()
	return nil
}
