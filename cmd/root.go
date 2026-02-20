package cmd

import (
	"fmt"
	"os"

	"github.com/deenaik/gtd-cli/internal/config"
	"github.com/deenaik/gtd-cli/internal/db"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gtd",
	Short: "Getting Things Done — CLI productivity system",
	Long:  "A GTD methodology CLI with Google Workspace integration, AI-powered features, and a background agent.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip init for config init command
		if cmd.Name() == "init" {
			return nil
		}
		// Ensure data dir and DB exist
		if err := config.EnsureDataDir(); err != nil {
			return fmt.Errorf("failed to create data dir: %w", err)
		}
		return db.Migrate()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default: show today dashboard
		return todayRun(cmd, args)
	},
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize GTD configuration and database",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.WriteDefault(); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
		if err := config.EnsureDataDir(); err != nil {
			return err
		}
		if err := db.Migrate(); err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		fmt.Printf("Initialized GTD at %s\n", config.DataDir())
		return nil
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
}
