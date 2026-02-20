package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/deenaik/gtd-cli/internal/ai"
	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/google"
	"github.com/deenaik/gtd-cli/internal/model"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Gmail integration commands",
	Long:  "Interact with Gmail: list inbox, read emails, capture to GTD inbox, reply, search, and summarize threads.",
}

// --- mail inbox ---

var mailInboxMax int
var mailInboxQuery string

var mailInboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "List recent emails",
	RunE: func(cmd *cobra.Command, args []string) error {
		emails, err := google.ListInbox(mailInboxMax, mailInboxQuery)
		if err != nil {
			return fmt.Errorf("failed to list inbox: %w", err)
		}

		if len(emails) == 0 {
			ui.PrintInfo("No emails found.")
			return nil
		}

		table := ui.NewTable("From", "Subject", "Date", "ID")
		for _, e := range emails {
			id := e.ID
			if len(id) > 8 {
				id = id[:8]
			}
			table.AddRow(e.From, e.Subject, e.Date, id)
		}
		table.Render()
		return nil
	},
}

// --- mail get ---

var mailGetCmd = &cobra.Command{
	Use:   "get <msg-id>",
	Short: "Show email details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email, err := google.GetMessage(args[0])
		if err != nil {
			return fmt.Errorf("failed to get message: %w", err)
		}

		ui.PrintSection("Email Details")
		fmt.Printf("From:    %s\n", email.From)
		fmt.Printf("To:      %s\n", email.To)
		fmt.Printf("Subject: %s\n", email.Subject)
		fmt.Printf("Date:    %s\n", email.Date)
		fmt.Printf("ID:      %s\n", email.ID)
		fmt.Printf("Thread:  %s\n", email.ThreadID)
		fmt.Println()
		fmt.Println(email.Body)
		return nil
	},
}

// --- mail capture ---

var mailCaptureCmd = &cobra.Command{
	Use:   "capture <msg-id>",
	Short: "Create inbox item from email",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email, err := google.GetMessage(args[0])
		if err != nil {
			return fmt.Errorf("failed to get message: %w", err)
		}

		inbox := store.NewInboxStore(db.Get())
		item := &model.InboxItem{
			Title:     email.Subject,
			Body:      email.Snippet,
			Source:    "email",
			SourceRef: email.ID,
		}
		if err := inbox.Create(item); err != nil {
			return fmt.Errorf("failed to create inbox item: %w", err)
		}

		ui.PrintSuccess(fmt.Sprintf("Captured to inbox: %q (id: %d)", item.Title, item.ID))
		return nil
	},
}

// --- mail reply ---

var mailReplyBody string

var mailReplyCmd = &cobra.Command{
	Use:   "reply <msg-id>",
	Short: "Reply to an email",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		msgID := args[0]
		body := mailReplyBody

		if body == "" {
			var err error
			body, err = ui.TextArea("Reply body", "Type your reply...")
			if err != nil {
				return fmt.Errorf("input cancelled: %w", err)
			}
			if strings.TrimSpace(body) == "" {
				return fmt.Errorf("reply body cannot be empty")
			}
		}

		if err := google.Reply(msgID, body); err != nil {
			return fmt.Errorf("failed to send reply: %w", err)
		}

		ui.PrintSuccess("Reply sent.")
		return nil
	},
}

// --- mail search ---

var mailSearchMax int

var mailSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search emails",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		emails, err := google.SearchEmails(query, mailSearchMax)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if len(emails) == 0 {
			ui.PrintInfo(fmt.Sprintf("No emails found for '%s'.", query))
			return nil
		}

		ui.PrintInfo(fmt.Sprintf("Found %d result(s) for '%s':", len(emails), query))
		fmt.Println()

		table := ui.NewTable("From", "Subject", "Date", "ID")
		for _, e := range emails {
			id := e.ID
			if len(id) > 8 {
				id = id[:8]
			}
			table.AddRow(e.From, e.Subject, e.Date, id)
		}
		table.Render()
		return nil
	},
}

// --- mail summarize ---

var mailSummarizeCmd = &cobra.Command{
	Use:   "summarize <thread-id>",
	Short: "AI summarize an email thread",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		threadID := args[0]

		// Search for messages in this thread
		emails, err := google.SearchEmails(fmt.Sprintf("thread:%s", threadID), 50)
		if err != nil {
			// Fallback: try to get single message
			msg, getErr := google.GetMessage(threadID)
			if getErr != nil {
				return fmt.Errorf("failed to get thread messages: %w", err)
			}
			emails = []google.Email{*msg}
		}

		if len(emails) == 0 {
			return fmt.Errorf("no messages found for thread %s", threadID)
		}

		// Convert to AI format
		var aiEmails []ai.EmailForSummary
		for _, e := range emails {
			aiEmails = append(aiEmails, ai.EmailForSummary{
				From:    e.From,
				To:      e.To,
				Date:    e.Date,
				Subject: e.Subject,
				Body:    e.Body,
			})
		}

		ui.PrintInfo(fmt.Sprintf("Summarizing %d message(s)...", len(aiEmails)))

		summary, err := ai.SummarizeThread(context.Background(), aiEmails)
		if err != nil {
			return fmt.Errorf("AI summarization failed: %w", err)
		}

		ui.PrintSection("Thread Summary")
		fmt.Println(summary)
		fmt.Println()
		fmt.Printf("Messages: %s\n", strconv.Itoa(len(aiEmails)))
		return nil
	},
}

func init() {
	// mail inbox flags
	mailInboxCmd.Flags().IntVar(&mailInboxMax, "max", 10, "Maximum number of emails to show")
	mailInboxCmd.Flags().StringVar(&mailInboxQuery, "query", "", "Gmail search query filter")

	// mail reply flags
	mailReplyCmd.Flags().StringVar(&mailReplyBody, "body", "", "Reply body (prompts if not provided)")

	// mail search flags
	mailSearchCmd.Flags().IntVar(&mailSearchMax, "max", 10, "Maximum number of results")

	// Register subcommands
	mailCmd.AddCommand(mailInboxCmd)
	mailCmd.AddCommand(mailGetCmd)
	mailCmd.AddCommand(mailCaptureCmd)
	mailCmd.AddCommand(mailReplyCmd)
	mailCmd.AddCommand(mailSearchCmd)
	mailCmd.AddCommand(mailSummarizeCmd)

	rootCmd.AddCommand(mailCmd)
}
