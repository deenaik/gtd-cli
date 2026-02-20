package cmd

import (
	"fmt"
	"strconv"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var inboxCount bool

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "List unprocessed inbox items",
	Long:  "Show all unprocessed items in the inbox. Use --count to just display the count.",
	RunE: func(cmd *cobra.Command, args []string) error {
		inbox := store.NewInboxStore(db.Get())

		if inboxCount {
			count, err := inbox.Count()
			if err != nil {
				return fmt.Errorf("failed to count inbox items: %w", err)
			}
			fmt.Printf("%d unprocessed inbox items\n", count)
			return nil
		}

		items, err := inbox.ListUnprocessed()
		if err != nil {
			return fmt.Errorf("failed to list inbox items: %w", err)
		}

		if len(items) == 0 {
			ui.PrintInfo("Inbox is empty. Well done!")
			return nil
		}

		table := ui.NewTable("ID", "Title", "Source", "Created")
		for _, item := range items {
			table.AddRow(
				strconv.FormatInt(item.ID, 10),
				item.Title,
				item.Source,
				item.CreatedAt.Format("2006-01-02 15:04"),
			)
		}
		table.Render()

		return nil
	},
}

func init() {
	inboxCmd.Flags().BoolVar(&inboxCount, "count", false, "Only print the count of unprocessed items")
	rootCmd.AddCommand(inboxCmd)
}
