package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// CategorySuggestion represents the LLM's categorization of an inbox item.
type CategorySuggestion struct {
	Category         string  `json:"category"`
	Confidence       float64 `json:"confidence"`
	Reasoning        string  `json:"reasoning"`
	SuggestedContext string  `json:"suggested_context"`
	SuggestedProject string  `json:"suggested_project"`
	Priority         int     `json:"priority"`
	Energy           string  `json:"energy"`

	// Convenience aliases populated after unmarshalling.
	Project string `json:"-"`
	Context string `json:"-"`
}

// validCategories is the set of allowed GTD categories.
var validCategories = map[string]bool{
	"next_action":  true,
	"waiting_for":  true,
	"someday_maybe": true,
	"reference":    true,
	"trash":        true,
}

// CategorizeItem takes an inbox item's title and body and uses the LLM to
// suggest a GTD category along with context and project recommendations.
func CategorizeItem(ctx context.Context, title, body string) (*CategorySuggestion, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("categorize: title is empty")
	}

	userContent := fmt.Sprintf("Title: %s", title)
	if body = strings.TrimSpace(body); body != "" {
		userContent += fmt.Sprintf("\n\nBody:\n%s", body)
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: CategorizeSystem,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userContent,
		},
	}

	schema := CategorizeSchema()
	temp := float32(0.2)

	raw, err := Complete(ctx, "categorize", messages,
		WithTemperature(temp),
		WithMaxTokens(256),
		WithJSONSchema(schema.Name, schema),
	)
	if err != nil {
		return nil, fmt.Errorf("categorize: %w", err)
	}

	var suggestion CategorySuggestion
	if err := json.Unmarshal([]byte(raw), &suggestion); err != nil {
		return nil, fmt.Errorf("categorize: parse response: %w", err)
	}

	// Validate the category.
	if !validCategories[suggestion.Category] {
		return nil, fmt.Errorf("categorize: invalid category %q from LLM", suggestion.Category)
	}

	// Populate convenience aliases from the schema fields.
	suggestion.Project = suggestion.SuggestedProject
	suggestion.Context = suggestion.SuggestedContext

	return &suggestion, nil
}
