package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/deenaik/gtd-cli/internal/agent"
	"github.com/deenaik/gtd-cli/internal/config"
	"github.com/deenaik/gtd-cli/internal/ui"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Background agent management",
	Long:  "Start, stop, and monitor the GTD background agent that watches email, calendar, and deadlines.",
}

// --- agent start ---

var agentStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the background agent daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := agent.Start(); err != nil {
			return fmt.Errorf("failed to start agent: %w", err)
		}

		ui.PrintSuccess("Agent started.")
		return nil
	},
}

// --- agent stop ---

var agentStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the background agent daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := agent.Stop(); err != nil {
			return fmt.Errorf("failed to stop agent: %w", err)
		}

		ui.PrintSuccess("Agent stopped.")
		return nil
	},
}

// --- agent status ---

var agentStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show agent status",
	RunE: func(cmd *cobra.Command, args []string) error {
		status := agent.Status()

		ui.PrintSection("Agent Status")

		if status.Running {
			ui.PrintSuccess(fmt.Sprintf("Running (PID %d)", status.PID))
		} else {
			ui.PrintWarn("Stopped")
			return nil
		}

		if len(status.Monitors) > 0 {
			fmt.Println()
			table := ui.NewTable("Monitor", "Last Run", "Duration (ms)", "OK", "Last Error")
			for _, m := range status.Monitors {
				ok := "yes"
				if !m.Success {
					ok = "no"
				}
				table.AddRow(m.Name, m.LastRun, fmt.Sprintf("%d", m.DurationMs), ok, m.LastError)
			}
			table.Render()
		}

		return nil
	},
}

// --- agent run ---

var agentRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run agent in foreground (debug mode)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return agent.RunForeground()
	},
}

// --- agent logs ---

var agentLogsFollow bool
var agentLogsLines int

var agentLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Tail the agent log file",
	RunE: func(cmd *cobra.Command, args []string) error {
		logPath := config.Get().AgentLogPath

		// Check if log file exists
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			return fmt.Errorf("agent log file not found: %s", logPath)
		}

		// Build tail command
		tailArgs := []string{"-n", strconv.Itoa(agentLogsLines)}
		if agentLogsFollow {
			tailArgs = append(tailArgs, "-f")
		}
		tailArgs = append(tailArgs, logPath)

		tailCmd := exec.Command("tail", tailArgs...)
		tailCmd.Stdout = os.Stdout
		tailCmd.Stderr = os.Stderr

		// Forward signals to tail process for clean shutdown
		if agentLogsFollow {
			tailCmd.SysProcAttr = &syscall.SysProcAttr{
				Setpgid: false,
			}
		}

		if err := tailCmd.Run(); err != nil {
			return fmt.Errorf("failed to tail log: %w", err)
		}

		return nil
	},
}

func init() {
	// agent logs flags
	agentLogsCmd.Flags().BoolVarP(&agentLogsFollow, "follow", "f", false, "Follow log output")
	agentLogsCmd.Flags().IntVarP(&agentLogsLines, "lines", "n", 50, "Number of lines to show")

	// Register subcommands
	agentCmd.AddCommand(agentStartCmd)
	agentCmd.AddCommand(agentStopCmd)
	agentCmd.AddCommand(agentStatusCmd)
	agentCmd.AddCommand(agentRunCmd)
	agentCmd.AddCommand(agentLogsCmd)

	rootCmd.AddCommand(agentCmd)
}
