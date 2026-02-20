package ai

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// WeeklyStats contains the productivity metrics for a given week, used as
// input for generating weekly review insights. Fields are a superset to
// support multiple callers (weekly review command and AI insights command).
type WeeklyStats struct {
	// Used by cmd/review.go
	CompletedCount int `json:"completed_count"`
	CreatedCount   int `json:"created_count"`
	PendingCount   int `json:"pending_count"`
	WaitingCount   int `json:"waiting_count"`
	SomedayCount   int `json:"someday_count"`
	ProjectCount   int `json:"project_count"`
	InboxCount     int `json:"inbox_count"`
	OverdueCount   int `json:"overdue_count"`

	// Used by cmd/ai.go
	TasksCompleted  int            `json:"tasks_completed"`
	TasksCreated    int            `json:"tasks_created"`
	OverdueTasks    int            `json:"overdue_tasks"`
	StatusBreakdown map[string]int `json:"status_breakdown"`
}

// WeeklyInsights generates an AI-powered weekly review summary based on the
// provided productivity statistics.
func WeeklyInsights(ctx context.Context, stats WeeklyStats) (string, error) {
	// Merge fields: prefer the more specific field if set, fall back to the other.
	completed := stats.CompletedCount
	if stats.TasksCompleted > 0 {
		completed = stats.TasksCompleted
	}
	created := stats.CreatedCount
	if stats.TasksCreated > 0 {
		created = stats.TasksCreated
	}
	overdue := stats.OverdueCount
	if stats.OverdueTasks > 0 {
		overdue = stats.OverdueTasks
	}

	// Build a human-readable stats summary for the prompt.
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Tasks completed this week: %d\n", completed))
	sb.WriteString(fmt.Sprintf("Tasks created this week: %d\n", created))
	if stats.PendingCount > 0 {
		sb.WriteString(fmt.Sprintf("Pending (next actions): %d\n", stats.PendingCount))
	}
	if stats.WaitingCount > 0 {
		sb.WriteString(fmt.Sprintf("Waiting for: %d\n", stats.WaitingCount))
	}
	if stats.SomedayCount > 0 {
		sb.WriteString(fmt.Sprintf("Someday/maybe: %d\n", stats.SomedayCount))
	}
	if stats.ProjectCount > 0 {
		sb.WriteString(fmt.Sprintf("Active projects: %d\n", stats.ProjectCount))
	}
	sb.WriteString(fmt.Sprintf("Unprocessed inbox items: %d\n", stats.InboxCount))
	sb.WriteString(fmt.Sprintf("Overdue tasks: %d\n", overdue))
	if len(stats.StatusBreakdown) > 0 {
		sb.WriteString("\nTasks by status:\n")
		for status, count := range stats.StatusBreakdown {
			sb.WriteString(fmt.Sprintf("  - %s: %d\n", status, count))
		}
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: WeeklyInsightsSystem,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: sb.String(),
		},
	}

	temp := float32(0.5)

	insights, err := Complete(ctx, "weekly_insights", messages,
		WithTemperature(temp),
		WithMaxTokens(1024),
	)
	if err != nil {
		return "", fmt.Errorf("weekly insights: %w", err)
	}

	return strings.TrimSpace(insights), nil
}
