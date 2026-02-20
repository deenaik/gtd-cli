package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// CapturedTask represents a structured task parsed from natural language input.
type CapturedTask struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Project      string `json:"project"`
	Context      string `json:"context"`
	Priority     int    `json:"priority"`
	DueDate      string `json:"due_date"`
	Energy       string `json:"energy"`
	TimeEstimate int    `json:"time_estimate"`
}

// SmartCapture takes natural language text and uses the LLM to parse it into
// a structured CapturedTask. If the LLM call fails, it falls back to using
// the raw text as the task title with sensible defaults.
func SmartCapture(ctx context.Context, text string) (*CapturedTask, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("smart capture: input text is empty")
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: SmartCaptureSystem,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: text,
		},
	}

	schema := SmartCaptureSchema()
	temp := float32(0.3)

	raw, err := Complete(ctx, "smart_capture", messages,
		WithTemperature(temp),
		WithMaxTokens(512),
		WithJSONSchema(schema.Name, schema),
	)
	if err != nil {
		// Fall back to simple capture: use the input text as the title.
		return fallbackCapture(text), nil
	}

	var task CapturedTask
	if err := json.Unmarshal([]byte(raw), &task); err != nil {
		return fallbackCapture(text), nil
	}

	// Validate required field.
	if task.Title == "" {
		task.Title = text
	}

	// Clamp priority to valid range.
	if task.Priority < 1 || task.Priority > 4 {
		task.Priority = 2
	}

	// Validate energy.
	switch task.Energy {
	case "low", "medium", "high":
		// valid
	default:
		task.Energy = "medium"
	}

	// Ensure a reasonable time estimate.
	if task.TimeEstimate <= 0 {
		task.TimeEstimate = 30
	}

	return &task, nil
}

// fallbackCapture creates a CapturedTask with sensible defaults when the LLM
// is unavailable or returns an unparseable response.
func fallbackCapture(text string) *CapturedTask {
	// Truncate excessively long input for the title.
	title := text
	if len(title) > 200 {
		title = title[:200] + "..."
	}
	return &CapturedTask{
		Title:               title,
		Description:         "",
		Project:             "",
		Context:             "",
		Priority:            2,
		DueDate:             "",
		Energy:              "medium",
		TimeEstimate: 30,
	}
}
