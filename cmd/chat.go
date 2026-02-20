package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/google"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Google Chat integration",
	Long:  "Send direct messages and task notifications via Google Chat.",
}

// --- chat dm ---

var chatDMCmd = &cobra.Command{
	Use:   "dm <email> <message>",
	Short: "Send a direct message",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]
		message := strings.Join(args[1:], " ")

		if err := google.SendDM(email, message); err != nil {
			return fmt.Errorf("failed to send DM: %w", err)
		}

		ui.PrintSuccess(fmt.Sprintf("Message sent to %s", email))
		return nil
	},
}

// --- chat notify ---

var chatNotifyCmd = &cobra.Command{
	Use:   "notify <email> <task-id>",
	Short: "Notify someone about a task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]
		taskID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}

		ts := store.NewTaskStore(db.Get())
		task, err := ts.GetByID(taskID)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}

		// Format the notification message
		message := fmt.Sprintf("GTD Task Notification\n\nTask: %s\nStatus: %s\nPriority: %s",
			task.Title, task.Status, task.PriorityLabel())

		if task.Description != "" {
			message += fmt.Sprintf("\nDescription: %s", task.Description)
		}
		if task.DueDate.Valid {
			message += fmt.Sprintf("\nDue: %s", task.DueDate.String)
		}
		if task.ProjectName != "" {
			message += fmt.Sprintf("\nProject: %s", task.ProjectName)
		}

		if err := google.SendDM(email, message); err != nil {
			return fmt.Errorf("failed to send notification: %w", err)
		}

		ui.PrintSuccess(fmt.Sprintf("Notified %s about task #%d: %s", email, task.ID, task.Title))
		return nil
	},
}

func init() {
	chatCmd.AddCommand(chatDMCmd)
	chatCmd.AddCommand(chatNotifyCmd)

	rootCmd.AddCommand(chatCmd)
}
