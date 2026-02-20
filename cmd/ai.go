package cmd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/deenaik/gtd-cli/internal/ai"
	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-powered productivity features",
	Long:  "Use AI to get task suggestions, categorize inbox items, generate insights, and schedule tasks.",
}

// --- ai suggest ---

var aiSuggestCmd = &cobra.Command{
	Use:   "suggest <task-id>",
	Short: "AI suggestions for a task",
	Long:  "Get AI-powered suggestions for breaking down or improving a task.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}

		ts := store.NewTaskStore(db.Get())
		task, err := ts.GetByID(taskID)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}

		ui.PrintInfo(fmt.Sprintf("Analyzing task #%d: %s", task.ID, task.Title))

		ctx := context.Background()
		text := task.Title
		if task.Description != "" {
			text += "\n" + task.Description
		}

		captured, err := ai.SmartCapture(ctx, text)
		if err != nil {
			return fmt.Errorf("AI analysis failed: %w", err)
		}

		ui.PrintSection("AI Suggestions")
		fmt.Printf("Title:        %s\n", captured.Title)
		if captured.Description != "" {
			fmt.Printf("Description:  %s\n", captured.Description)
		}
		if captured.Project != "" {
			fmt.Printf("Project:      %s\n", captured.Project)
		}
		if captured.Context != "" {
			fmt.Printf("Context:      %s\n", captured.Context)
		}
		if captured.Priority > 0 {
			fmt.Printf("Priority:     %d\n", captured.Priority)
		}
		if captured.DueDate != "" {
			fmt.Printf("Due Date:     %s\n", captured.DueDate)
		}
		if captured.Energy != "" {
			fmt.Printf("Energy:       %s\n", captured.Energy)
		}
		if captured.TimeEstimate > 0 {
			fmt.Printf("Time Est:     %d min\n", captured.TimeEstimate)
		}

		return nil
	},
}

// --- ai categorize ---

var aiCategorizeCmd = &cobra.Command{
	Use:   "categorize",
	Short: "AI categorize all unprocessed inbox items",
	RunE: func(cmd *cobra.Command, args []string) error {
		d := db.Get()
		inbox := store.NewInboxStore(d)

		items, err := inbox.ListUnprocessed()
		if err != nil {
			return fmt.Errorf("failed to list inbox: %w", err)
		}

		if len(items) == 0 {
			ui.PrintInfo("Inbox is empty. Nothing to categorize!")
			return nil
		}

		ui.PrintSection(fmt.Sprintf("AI Categorizing %d inbox items", len(items)))

		ctx := context.Background()
		for i, item := range items {
			fmt.Println()
			ui.PrintInfo(fmt.Sprintf("[%d/%d] %s", i+1, len(items), item.Title))

			suggestion, err := ai.CategorizeItem(ctx, item.Title, item.Body)
			if err != nil {
				ui.PrintError(fmt.Sprintf("Failed to categorize: %v", err))
				continue
			}

			fmt.Printf("  Category:   %s (confidence: %.0f%%)\n", suggestion.Category, suggestion.Confidence*100)
			if suggestion.Reasoning != "" {
				fmt.Printf("  Reasoning:  %s\n", suggestion.Reasoning)
			}
			if suggestion.SuggestedProject != "" {
				fmt.Printf("  Project:    %s\n", suggestion.SuggestedProject)
			}
			if suggestion.SuggestedContext != "" {
				fmt.Printf("  Context:    %s\n", suggestion.SuggestedContext)
			}
			if suggestion.Priority > 0 {
				fmt.Printf("  Priority:   %d\n", suggestion.Priority)
			}
			if suggestion.Energy != "" {
				fmt.Printf("  Energy:     %s\n", suggestion.Energy)
			}

			confirmed, err := ui.Confirm("Accept this categorization?")
			if err != nil {
				ui.PrintWarn("Skipping remaining items.")
				break
			}
			if confirmed {
				ui.PrintSuccess(fmt.Sprintf("Accepted: %s -> %s", item.Title, suggestion.Category))
			} else {
				ui.PrintInfo("Skipped.")
			}
		}

		fmt.Println()
		ui.PrintSuccess("AI categorization complete!")
		return nil
	},
}

// --- ai insights ---

var aiInsightsCmd = &cobra.Command{
	Use:   "insights",
	Short: "AI weekly productivity insights",
	RunE: func(cmd *cobra.Command, args []string) error {
		d := db.Get()
		ts := store.NewTaskStore(d)
		is := store.NewInboxStore(d)

		// Gather weekly stats
		weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

		completed, err := ts.CompletedSince(weekAgo)
		if err != nil {
			return fmt.Errorf("failed to count completed tasks: %w", err)
		}

		created, err := ts.CreatedSince(weekAgo)
		if err != nil {
			return fmt.Errorf("failed to count created tasks: %w", err)
		}

		inboxCount, err := is.Count()
		if err != nil {
			return fmt.Errorf("failed to count inbox: %w", err)
		}

		statusMap, err := ts.CountByStatus()
		if err != nil {
			return fmt.Errorf("failed to count by status: %w", err)
		}

		// Count overdue tasks
		overdueTasks, err := ts.List(store.TaskFilter{
			DueBefore: time.Now().Format("2006-01-02"),
		})
		if err != nil {
			return fmt.Errorf("failed to list overdue: %w", err)
		}
		overdueCount := 0
		today := time.Now().Format("2006-01-02")
		for _, t := range overdueTasks {
			if t.DueDate.Valid && t.DueDate.String < today {
				overdueCount++
			}
		}

		// Count active projects
		ps := store.NewProjectStore(d)
		activeProjects, err := ps.List("active")
		projectCount := 0
		if err == nil {
			projectCount = len(activeProjects)
		}

		stats := ai.WeeklyStats{
			CompletedCount: completed,
			CreatedCount:   created,
			PendingCount:   statusMap["pending"],
			WaitingCount:   statusMap["waiting_for"],
			SomedayCount:   statusMap["someday_maybe"],
			ProjectCount:   projectCount,
			InboxCount:     inboxCount,
			OverdueCount:   overdueCount,
		}

		ui.PrintInfo("Generating weekly insights...")

		insights, err := ai.WeeklyInsights(context.Background(), stats)
		if err != nil {
			return fmt.Errorf("AI insights failed: %w", err)
		}

		ui.PrintSection("Weekly Stats")
		fmt.Printf("Tasks completed (7d): %d\n", completed)
		fmt.Printf("Tasks created (7d):   %d\n", created)
		fmt.Printf("Pending tasks:        %d\n", statusMap["pending"])
		fmt.Printf("Waiting for:          %d\n", statusMap["waiting_for"])
		fmt.Printf("Active projects:      %d\n", projectCount)
		fmt.Printf("Inbox items:          %d\n", inboxCount)
		fmt.Printf("Overdue tasks:        %d\n", overdueCount)

		ui.PrintSection("AI Analysis")
		fmt.Println(insights)

		return nil
	},
}

// --- ai schedule ---

var aiScheduleCmd = &cobra.Command{
	Use:   "schedule <task-id>",
	Short: "AI suggest best time to schedule a task",
	Long:  "Shortcut for 'gtd cal schedule'. Uses AI to find the optimal time slot.",
	Args:  cobra.ExactArgs(1),
	RunE:  calScheduleRun, // reuse the cal schedule implementation
}

// --- ai usage ---

var aiUsageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show today's AI token usage",
	RunE: func(cmd *cobra.Command, args []string) error {
		tokens, err := ai.TodayUsage()
		if err != nil {
			return fmt.Errorf("failed to get usage: %w", err)
		}

		ui.PrintSection("AI Token Usage")
		fmt.Printf("Today's usage: %d tokens\n", tokens)
		return nil
	},
}

func init() {
	aiCmd.AddCommand(aiSuggestCmd)
	aiCmd.AddCommand(aiCategorizeCmd)
	aiCmd.AddCommand(aiInsightsCmd)
	aiCmd.AddCommand(aiScheduleCmd)
	aiCmd.AddCommand(aiUsageCmd)

	rootCmd.AddCommand(aiCmd)
}
