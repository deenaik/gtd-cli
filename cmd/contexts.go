package cmd

import (
	"fmt"
	"strconv"

	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/deenaik/gtd-cli/internal/store"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var contextsCmd = &cobra.Command{
	Use:   "contexts",
	Short: "Manage contexts",
	Long:  "List, add, and delete contexts (e.g. @home, @office, @errands).",
	RunE: func(cmd *cobra.Command, args []string) error {
		return contextsListRun(cmd, args)
	},
}

var contextsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contexts",
	RunE:  contextsListRun,
}

var contextsAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new context",
	Args:  cobra.ExactArgs(1),
	RunE:  contextsAddRun,
}

var contextsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a context",
	Args:  cobra.ExactArgs(1),
	RunE:  contextsDeleteRun,
}

func init() {
	rootCmd.AddCommand(contextsCmd)

	contextsCmd.AddCommand(contextsListCmd)
	contextsCmd.AddCommand(contextsAddCmd)
	contextsCmd.AddCommand(contextsDeleteCmd)
}

func contextsListRun(cmd *cobra.Command, args []string) error {
	cs := store.NewContextStore(db.Get())
	contexts, err := cs.List()
	if err != nil {
		return fmt.Errorf("failed to list contexts: %w", err)
	}

	if len(contexts) == 0 {
		ui.PrintInfo("No contexts found.")
		return nil
	}

	t := ui.NewTable("ID", "Name", "Created")
	for _, c := range contexts {
		t.AddRow(
			strconv.FormatInt(c.ID, 10),
			c.Name,
			c.CreatedAt.Format("2006-01-02"),
		)
	}
	t.Render()
	return nil
}

func contextsAddRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	cs := store.NewContextStore(db.Get())
	c, err := cs.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create context: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Context created: %s (#%d)", c.Name, c.ID))
	return nil
}

func contextsDeleteRun(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid context ID: %s", args[0])
	}

	cs := store.NewContextStore(db.Get())
	c, err := cs.GetByID(id)
	if err != nil {
		return fmt.Errorf("context not found: %w", err)
	}

	confirmed, err := ui.Confirm(fmt.Sprintf("Delete context '%s' (#%d)?", c.Name, c.ID))
	if err != nil {
		return err
	}
	if !confirmed {
		ui.PrintWarn("Cancelled.")
		return nil
	}

	if err := cs.Delete(id); err != nil {
		return fmt.Errorf("failed to delete context: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Context deleted: %s (#%d)", c.Name, c.ID))
	return nil
}
