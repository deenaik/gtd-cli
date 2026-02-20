package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/deenaik/gtd-cli/internal/config"
	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/google"
	"github.com/deenaik/gtd-cli/internal/model"
	"github.com/deenaik/gtd-cli/internal/store"
)

// vipSenders is a default list of email addresses/domains that are always
// treated as high-importance. This could be made configurable in the future.
var vipSenders = []string{
	"@google.com",
	"@github.com",
}

// RunEmailMonitor polls for unread emails, classifies their importance, and
// auto-creates inbox items for high-importance messages.
func RunEmailMonitor(ctx context.Context) error {
	emails, err := google.ListUnread(10)
	if err != nil {
		return fmt.Errorf("list unread: %w", err)
	}

	if len(emails) == 0 {
		slog.Debug("email monitor: no unread emails")
		return nil
	}

	d := db.Get()

	// Filter out already-processed threads.
	var newEmails []google.Email
	for _, e := range emails {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		var exists int
		err := d.QueryRow(`SELECT COUNT(*) FROM processed_emails WHERE thread_id = ?`, e.ThreadID).Scan(&exists)
		if err != nil {
			slog.Warn("email monitor: check processed", "thread_id", e.ThreadID, "error", err)
			continue
		}
		if exists > 0 {
			continue
		}
		newEmails = append(newEmails, e)
	}

	if len(newEmails) == 0 {
		slog.Debug("email monitor: all emails already processed")
		return nil
	}

	slog.Info("email monitor: processing new emails", "count", len(newEmails))

	// Classify importance.
	classifications := classifyEmails(ctx, newEmails)

	// Process each classified email.
	inbox := store.NewInboxStore(d)
	for i, e := range newEmails {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		importance := classifications[i]

		// Record as processed.
		_, err := d.Exec(`INSERT OR IGNORE INTO processed_emails (thread_id, importance) VALUES (?, ?)`,
			e.ThreadID, importance)
		if err != nil {
			slog.Error("email monitor: record processed email", "thread_id", e.ThreadID, "error", err)
			continue
		}

		if importance == "high" {
			// Auto-create an inbox item.
			meta, _ := json.Marshal(map[string]string{
				"from":      e.From,
				"thread_id": e.ThreadID,
				"date":      e.Date,
			})
			item := &model.InboxItem{
				Title:      fmt.Sprintf("[Email] %s", e.Subject),
				Body:       truncate(e.Snippet, 500),
				Source:     "email",
				SourceRef:  e.ThreadID,
				SourceMeta: string(meta),
			}
			if err := inbox.Create(item); err != nil {
				slog.Error("email monitor: create inbox item", "subject", e.Subject, "error", err)
				continue
			}

			slog.Info("email monitor: created inbox item for high-importance email",
				"subject", e.Subject, "from", e.From)

			// Desktop notification for high-importance emails.
			Notify(
				"Important Email",
				fmt.Sprintf("From: %s\n%s", e.From, e.Subject),
			)
		}
	}

	return nil
}

// classifyEmails determines the importance level of each email. If the OpenAI
// API key is configured, it uses LLM-based batch classification. Otherwise,
// it falls back to rule-based heuristics.
func classifyEmails(ctx context.Context, emails []google.Email) []string {
	cfg := config.Get()

	if cfg.OpenAIKey != "" {
		result, err := classifyWithAI(ctx, emails)
		if err != nil {
			slog.Warn("email monitor: AI classification failed, falling back to rules", "error", err)
			return classifyWithRules(emails)
		}
		return result
	}

	return classifyWithRules(emails)
}

// classifyWithAI sends a batch of email summaries to the LLM and asks it to
// classify each as "high", "medium", or "low" importance.
func classifyWithAI(ctx context.Context, emails []google.Email) ([]string, error) {
	cfg := config.Get()

	// Build a summary payload for the LLM.
	var sb strings.Builder
	sb.WriteString("Classify each of the following emails as 'high', 'medium', or 'low' importance.\n")
	sb.WriteString("Consider: sender reputation, urgency cues in subject/snippet, action required, deadlines.\n")
	sb.WriteString("Respond with a JSON array of strings, one per email, in the same order.\n\n")

	for i, e := range emails {
		fmt.Fprintf(&sb, "Email %d:\n  From: %s\n  Subject: %s\n  Snippet: %s\n\n",
			i+1, e.From, e.Subject, truncate(e.Snippet, 200))
	}

	// Build the OpenAI API request.
	type chatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type chatRequest struct {
		Model    string        `json:"model"`
		Messages []chatMessage `json:"messages"`
	}
	type chatChoice struct {
		Message chatMessage `json:"message"`
	}
	type chatResponse struct {
		Choices []chatChoice `json:"choices"`
	}

	reqBody := chatRequest{
		Model: cfg.OpenAIModel,
		Messages: []chatMessage{
			{Role: "system", Content: "You are a helpful assistant that classifies emails by importance. Always respond with valid JSON."},
			{Role: "user", Content: sb.String()},
		},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in API response")
	}

	content := chatResp.Choices[0].Message.Content

	// Parse the JSON array response.
	var classifications []string
	if err := json.Unmarshal([]byte(content), &classifications); err != nil {
		// Try to extract from a code block.
		cleaned := extractJSON(content)
		if err := json.Unmarshal([]byte(cleaned), &classifications); err != nil {
			return nil, fmt.Errorf("parse AI response: %w", err)
		}
	}

	// Ensure we have the right number of results.
	if len(classifications) != len(emails) {
		return nil, fmt.Errorf("AI returned %d classifications for %d emails", len(classifications), len(emails))
	}

	// Normalize values.
	for i, c := range classifications {
		c = strings.ToLower(strings.TrimSpace(c))
		switch c {
		case "high", "medium", "low":
			classifications[i] = c
		default:
			classifications[i] = "medium"
		}
	}

	return classifications, nil
}

// classifyWithRules applies simple heuristic rules to determine email
// importance.
func classifyWithRules(emails []google.Email) []string {
	results := make([]string, len(emails))
	for i, e := range emails {
		results[i] = classifyEmailByRule(e)
	}
	return results
}

// classifyEmailByRule applies rule-based heuristics to a single email.
func classifyEmailByRule(e google.Email) string {
	from := strings.ToLower(e.From)
	subject := strings.ToLower(e.Subject)
	snippet := strings.ToLower(e.Snippet)

	// VIP sender check.
	for _, vip := range vipSenders {
		if strings.Contains(from, vip) {
			return "high"
		}
	}

	// Check the user's own email - replies to self are usually important.
	cfg := config.Get()
	if cfg.UserEmail != "" && strings.Contains(from, strings.ToLower(cfg.UserEmail)) {
		return "high"
	}

	// Urgency keywords in subject.
	urgentKeywords := []string{"urgent", "asap", "critical", "emergency", "action required", "immediate", "deadline"}
	for _, kw := range urgentKeywords {
		if strings.Contains(subject, kw) || strings.Contains(snippet, kw) {
			return "high"
		}
	}

	// Calendar/meeting related.
	meetingKeywords := []string{"meeting", "calendar", "invite", "rsvp", "agenda"}
	for _, kw := range meetingKeywords {
		if strings.Contains(subject, kw) {
			return "medium"
		}
	}

	// Newsletters and notifications tend to be low importance.
	lowKeywords := []string{"unsubscribe", "newsletter", "no-reply", "noreply", "digest", "notification"}
	for _, kw := range lowKeywords {
		if strings.Contains(from, kw) || strings.Contains(subject, kw) || strings.Contains(snippet, kw) {
			return "low"
		}
	}

	return "medium"
}

// truncate shortens a string to the given max length, appending "..." if
// truncation occurred.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// extractJSON attempts to extract a JSON array from a string that may be
// wrapped in markdown code fences.
func extractJSON(s string) string {
	// Look for content between ```json and ``` or between ``` and ```.
	if idx := strings.Index(s, "```json"); idx != -1 {
		s = s[idx+7:]
		if end := strings.Index(s, "```"); end != -1 {
			return strings.TrimSpace(s[:end])
		}
	}
	if idx := strings.Index(s, "```"); idx != -1 {
		s = s[idx+3:]
		if end := strings.Index(s, "```"); end != -1 {
			return strings.TrimSpace(s[:end])
		}
	}
	// Try to find a JSON array directly.
	if start := strings.Index(s, "["); start != -1 {
		if end := strings.LastIndex(s, "]"); end != -1 {
			return s[start : end+1]
		}
	}
	return s
}
