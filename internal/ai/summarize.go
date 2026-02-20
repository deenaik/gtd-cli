package ai

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// EmailForSummary represents a single email within a thread to be summarized.
type EmailForSummary struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Date    string `json:"date"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// SummarizeThread takes a slice of emails in a thread and returns a concise,
// actionable summary produced by the LLM.
func SummarizeThread(ctx context.Context, emails []EmailForSummary) (string, error) {
	if len(emails) == 0 {
		return "", fmt.Errorf("summarize: no emails provided")
	}

	// Build the thread representation for the prompt.
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Email thread with %d message(s):\n\n", len(emails)))
	for i, e := range emails {
		sb.WriteString(fmt.Sprintf("--- Message %d ---\n", i+1))
		sb.WriteString(fmt.Sprintf("From: %s\n", e.From))
		sb.WriteString(fmt.Sprintf("To: %s\n", e.To))
		sb.WriteString(fmt.Sprintf("Date: %s\n", e.Date))
		sb.WriteString(fmt.Sprintf("Subject: %s\n", e.Subject))
		sb.WriteString(fmt.Sprintf("\n%s\n\n", e.Body))
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: EmailSummarizeSystem,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: sb.String(),
		},
	}

	temp := float32(0.3)

	summary, err := Complete(ctx, "email_summarize", messages,
		WithTemperature(temp),
		WithMaxTokens(1024),
	)
	if err != nil {
		return "", fmt.Errorf("summarize: %w", err)
	}

	return strings.TrimSpace(summary), nil
}
