package cmd

import (
	"fmt"
	"strings"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/model"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var addAI bool

var addCmd = &cobra.Command{
	Use:   "add [text]",
	Short: "Quick capture to inbox",
	Long:  "Add an item to the inbox for later processing. Pass the text as arguments or it will be joined.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if addAI {
			ui.PrintWarn("AI mode not yet available")
		}

		title := strings.Join(args, " ")

		inbox := store.NewInboxStore(db.Get())
		item := &model.InboxItem{
			Title:  title,
			Source: "manual",
		}
		if err := inbox.Create(item); err != nil {
			return fmt.Errorf("failed to add inbox item: %w", err)
		}

		ui.PrintSuccess(fmt.Sprintf("Added to inbox: %q (id: %d)", item.Title, item.ID))
		return nil
	},
}

func init() {
	addCmd.Flags().BoolVar(&addAI, "ai", false, "Use AI to parse and categorize (placeholder)")
	rootCmd.AddCommand(addCmd)
}
