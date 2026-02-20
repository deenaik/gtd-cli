package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// TaskForSchedule contains the task properties relevant to scheduling.
type TaskForSchedule struct {
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	Priority     int    `json:"priority"`
	Energy       string `json:"energy"`
	TimeEstimate int    `json:"time_estimate"`
	DueDate      string `json:"due_date,omitempty"`
	Context      string `json:"context,omitempty"`
}

// BusySlot represents an existing calendar event that blocks a time range.
type BusySlot struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// ScheduleSuggestion holds the LLM's recommended time slot for a task.
type ScheduleSuggestion struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	Reasoning string `json:"reasoning"`
}

// scheduleSuggestionResponse is the raw LLM structured output wrapping an
// array of suggestions.
type scheduleSuggestionResponse struct {
	Suggestions []ScheduleSuggestion `json:"suggestions"`
}

// SuggestSchedule uses the LLM to recommend a time slot for the given task,
// taking into account the task's properties and existing busy slots. It returns
// the best (first) suggestion from the LLM.
func SuggestSchedule(ctx context.Context, task TaskForSchedule, busySlots []BusySlot) (*ScheduleSuggestion, error) {
	if task.Title == "" {
		return nil, fmt.Errorf("schedule: task title is empty")
	}

	est := task.TimeEstimate
	if est <= 0 {
		est = 30
	}

	// Build the user prompt with task details and calendar context.
	var sb strings.Builder
	sb.WriteString("Task to schedule:\n")
	sb.WriteString(fmt.Sprintf("  Title: %s\n", task.Title))
	if task.Description != "" {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", task.Description))
	}
	sb.WriteString(fmt.Sprintf("  Priority: %d (1=low, 2=medium, 3=high, 4=urgent)\n", task.Priority))
	sb.WriteString(fmt.Sprintf("  Energy: %s\n", task.Energy))
	sb.WriteString(fmt.Sprintf("  Estimated duration: %d minutes\n", est))
	if task.DueDate != "" {
		sb.WriteString(fmt.Sprintf("  Due date: %s\n", task.DueDate))
	}
	if task.Context != "" {
		sb.WriteString(fmt.Sprintf("  Context: %s\n", task.Context))
	}

	now := time.Now()
	sb.WriteString(fmt.Sprintf("\nCurrent date/time: %s\n", now.Format(time.RFC3339)))

	if len(busySlots) > 0 {
		sb.WriteString(fmt.Sprintf("\nExisting calendar events (%d):\n", len(busySlots)))
		for _, slot := range busySlots {
			sb.WriteString(fmt.Sprintf("  - %s to %s\n", slot.Start, slot.End))
		}
	} else {
		sb.WriteString("\nNo existing calendar events (calendar is free).\n")
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: SmartScheduleSystem,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: sb.String(),
		},
	}

	schema := ScheduleSuggestionSchema()
	temp := float32(0.4)

	raw, err := Complete(ctx, "smart_schedule", messages,
		WithTemperature(temp),
		WithMaxTokens(512),
		WithJSONSchema(schema.Name, schema),
	)
	if err != nil {
		return nil, fmt.Errorf("schedule: %w", err)
	}

	var resp scheduleSuggestionResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("schedule: parse response: %w", err)
	}

	if len(resp.Suggestions) == 0 {
		return nil, fmt.Errorf("schedule: LLM returned no suggestions")
	}

	// Return the best (first) suggestion.
	return &resp.Suggestions[0], nil
}
