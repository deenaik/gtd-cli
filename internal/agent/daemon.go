package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/deenaik/gtd-cli/internal/config"
	"github.com/deenaik/gtd-cli/internal/db"
)

// AgentStatus describes the current state of the background agent.
type AgentStatus struct {
	Running  bool
	PID      int
	Monitors []MonitorStatus
}

// MonitorStatus captures the health state of a single monitor as persisted
// in the monitor_state table.
type MonitorStatus struct {
	Name       string
	Status     string // "ok", "error", "unknown"
	LastRun    string
	DurationMs int64
	Success    bool
	LastError  string
	Message    string // human-readable summary
}

// PIDFile returns the absolute path to the agent PID file.
func PIDFile() string {
	return filepath.Join(config.Get().DataDir, "agent.pid")
}

// Start launches the agent as a detached background process by re-executing
// the current binary with "agent run" arguments. It writes the child PID to
// the PID file and redirects stdout/stderr to the configured agent log.
func Start() error {
	if running, _ := IsRunning(); running {
		return fmt.Errorf("agent is already running (pid file: %s)", PIDFile())
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	cfg := config.Get()

	// Ensure data directory exists.
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	logFile, err := os.OpenFile(cfg.AgentLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open agent log: %w", err)
	}

	cmd := exec.Command(exe, "agent", "run")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil

	// Detach the child into its own process group so it survives the parent
	// exiting.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Inherit a minimal environment so config and credentials are available.
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("start agent process: %w", err)
	}

	// We deliberately do NOT call cmd.Wait() -- the child is detached.
	pid := cmd.Process.Pid

	if err := os.WriteFile(PIDFile(), []byte(strconv.Itoa(pid)), 0o644); err != nil {
		// Best-effort kill if we cannot record the PID.
		_ = cmd.Process.Kill()
		logFile.Close()
		return fmt.Errorf("write pid file: %w", err)
	}

	// Release the process handle so it is not kept alive by this parent.
	_ = cmd.Process.Release()
	logFile.Close()

	return nil
}

// Stop gracefully shuts down the running agent. It sends SIGTERM, waits up to
// 10 seconds, then force-kills the process if it is still alive. The PID file
// is removed on success.
func Stop() error {
	pid, err := readPID()
	if err != nil {
		return fmt.Errorf("agent not running: %w", err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		removePIDFile()
		return fmt.Errorf("find process %d: %w", pid, err)
	}

	// Send SIGTERM for graceful shutdown.
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Process already gone.
		removePIDFile()
		return nil
	}

	// Poll for up to 10 seconds waiting for the process to exit.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			removePIDFile()
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}

	// Force kill.
	_ = proc.Signal(syscall.SIGKILL)
	time.Sleep(500 * time.Millisecond)
	removePIDFile()

	return nil
}

// Status returns a detailed AgentStatus including whether the agent is
// running, its PID, and the state of each registered monitor.
func Status() AgentStatus {
	status := AgentStatus{}

	pid, err := readPID()
	if err != nil {
		return status
	}

	if !processAlive(pid) {
		// Stale PID file.
		removePIDFile()
		return status
	}

	status.Running = true
	status.PID = pid

	// Read monitor states from the database.
	status.Monitors = readMonitorStates()

	return status
}

// IsRunning performs a quick check to determine whether the agent process is
// alive.
func IsRunning() (bool, error) {
	pid, err := readPID()
	if err != nil {
		return false, nil
	}
	if !processAlive(pid) {
		removePIDFile()
		return false, nil
	}
	return true, nil
}

// readPID reads and parses the PID from the PID file.
func readPID() (int, error) {
	data, err := os.ReadFile(PIDFile())
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid pid file contents: %w", err)
	}
	return pid, nil
}

// processAlive checks whether a process with the given PID exists by sending
// signal 0.
func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 does not send a signal but performs error checking -- if the
	// process does not exist the call returns an error.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// removePIDFile removes the PID file, ignoring errors.
func removePIDFile() {
	_ = os.Remove(PIDFile())
}

// readMonitorStates queries the monitor_state table for all monitor health
// records.
func readMonitorStates() []MonitorStatus {
	d := db.Get()
	rows, err := d.Query(`SELECT name, COALESCE(last_run, ''), COALESCE(duration_ms, 0), COALESCE(success, 0), COALESCE(last_error, '') FROM monitor_state ORDER BY name`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var states []MonitorStatus
	for rows.Next() {
		var s MonitorStatus
		if err := rows.Scan(&s.Name, &s.LastRun, &s.DurationMs, &s.Success, &s.LastError); err != nil {
			continue
		}

		// Derive human-readable Status and Message fields.
		if s.Success {
			s.Status = "ok"
			s.Message = fmt.Sprintf("completed in %dms", s.DurationMs)
		} else if s.LastError != "" {
			s.Status = "error"
			s.Message = s.LastError
		} else {
			s.Status = "unknown"
			s.Message = ""
		}

		states = append(states, s)
	}
	return states
}
