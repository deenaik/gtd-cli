package google

import (
	"encoding/json"
	"fmt"
)

// Email represents a Gmail message returned by the gog CLI.
// For search/list results, the top-level fields (ID, From, Subject, Date) are
// populated directly. For get results, the headers sub-object is used.
type Email struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
	Subject  string `json:"subject"`
	From     string `json:"from"`
	To       string `json:"to"`
	Date     string `json:"date"`
	Snippet  string `json:"snippet"`
	Body     string `json:"body"`
	IsUnread bool   `json:"isUnread"`

	// MessageCount is returned by gog gmail search (thread list).
	MessageCount int `json:"messageCount"`
}

// gogEmailGetResponse is the raw JSON structure from gog gmail get.
type gogEmailGetResponse struct {
	Body    string `json:"body"`
	Headers struct {
		From    string `json:"from"`
		To      string `json:"to"`
		Subject string `json:"subject"`
		Date    string `json:"date"`
		CC      string `json:"cc"`
		BCC     string `json:"bcc"`
	} `json:"headers"`
	Message struct {
		ID       string `json:"id"`
		ThreadID string `json:"threadId"`
	} `json:"message"`
}

// ListInbox retrieves inbox messages. Use max to limit the number of results
// and query to filter. Either parameter can be zero-valued to omit it.
func ListInbox(max int, query string) ([]Email, error) {
	if query == "" {
		query = "in:inbox"
	}
	args := []string{"gmail", "list", query}
	if max > 0 {
		args = append(args, "--max", fmt.Sprintf("%d", max))
	}
	var emails []Email
	if err := RunJSON(&emails, args...); err != nil {
		return nil, fmt.Errorf("list inbox: %w", err)
	}
	return emails, nil
}

// GetMessage retrieves a single email by its message ID.
func GetMessage(id string) (*Email, error) {
	// gog gmail get returns a different structure than list.
	out, err := Run("-j", "gmail", "get", id)
	if err != nil {
		return nil, fmt.Errorf("get message %s: %w", id, err)
	}

	var raw gogEmailGetResponse
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("get message %s: parse: %w", id, err)
	}

	return &Email{
		ID:       raw.Message.ID,
		ThreadID: raw.Message.ThreadID,
		Subject:  raw.Headers.Subject,
		From:     raw.Headers.From,
		To:       raw.Headers.To,
		Date:     raw.Headers.Date,
		Body:     raw.Body,
	}, nil
}

// Reply sends a reply to the specified message.
func Reply(messageID string, body string) error {
	_, err := Run("gmail", "send",
		"--reply-to-message-id", messageID,
		"--reply-all",
		"--body", body,
		"--quote",
	)
	if err != nil {
		return fmt.Errorf("reply to %s: %w", messageID, err)
	}
	return nil
}

// SearchEmails searches for emails matching the given query. Use max to limit
// the number of results (0 means no limit).
func SearchEmails(query string, max int) ([]Email, error) {
	args := []string{"gmail", "search", query}
	if max > 0 {
		args = append(args, "--max", fmt.Sprintf("%d", max))
	}
	var emails []Email
	if err := RunJSON(&emails, args...); err != nil {
		return nil, fmt.Errorf("search emails: %w", err)
	}
	return emails, nil
}

// ListUnread retrieves unread messages from the inbox.
func ListUnread(max int) ([]Email, error) {
	return ListInbox(max, "is:unread")
}
