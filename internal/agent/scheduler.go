package agent

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/deenaik/gtd-cli/internal/config"
	"github.com/deenaik/gtd-cli/internal/db"
)

// Monitor represents a single periodic task that the scheduler runs.
type Monitor struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context) error
}

// consecutiveFailures tracks how many times in a row a monitor has failed.
type monitorHealth struct {
	mu          sync.Mutex
	failures    int
	pausedUntil time.Time
}

const (
	failureNotifyThreshold = 3
	failurePauseThreshold  = 10
	pauseDuration          = 1 * time.Hour
)

// RunForeground is the main event loop for the agent. It is intended to be
// called by "agent run" in the detached child process. It configures
// structured logging, registers all monitors, starts their ticker loops, and
// blocks until SIGTERM or SIGINT is received.
func RunForeground() error {
	cfg := config.Get()

	// Ensure the database schema is up to date.
	if err := db.Migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	// Set up structured logging to both the log file and stderr.
	logFile, err := os.OpenFile(cfg.AgentLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(logFile, os.Stderr)
	logger := slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("agent starting", "pid", os.Getpid())

	// Build the list of monitors from configuration.
	monitors := []Monitor{
		{
			Name:     "email",
			Interval: time.Duration(cfg.EmailInterval) * time.Second,
			Run:      RunEmailMonitor,
		},
		{
			Name:     "calendar",
			Interval: time.Duration(cfg.CalendarInterval) * time.Second,
			Run:      RunCalendarMonitor,
		},
		{
			Name:     "deadline",
			Interval: time.Duration(cfg.DeadlineInterval) * time.Second,
			Run:      RunDeadlineMonitor,
		},
		{
			Name:     "progress",
			Interval: time.Duration(cfg.ProgressInterval) * time.Second,
			Run:      RunProgressMonitor,
		},
	}

	// Create a cancellable context for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Catch termination signals.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	var wg sync.WaitGroup

	// Health trackers keyed by monitor name.
	healthMap := make(map[string]*monitorHealth)
	for _, m := range monitors {
		healthMap[m.Name] = &monitorHealth{}
	}

	// Launch a goroutine per monitor.
	for _, m := range monitors {
		m := m // capture loop variable
		health := healthMap[m.Name]
		wg.Add(1)
		go func() {
			defer wg.Done()
			runMonitorLoop(ctx, m, health)
		}()
	}

	slog.Info("agent running", "monitors", len(monitors))

	// Block until a termination signal is received.
	sig := <-sigCh
	slog.Info("received signal, shutting down", "signal", sig)
	cancel()

	// Wait for all monitor goroutines to finish.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("all monitors stopped")
	case <-time.After(15 * time.Second):
		slog.Warn("shutdown timed out, some monitors may not have stopped cleanly")
	}

	slog.Info("agent stopped")
	return nil
}

// runMonitorLoop drives a single monitor on its ticker interval.
func runMonitorLoop(ctx context.Context, m Monitor, health *monitorHealth) {
	slog.Info("monitor started", "name", m.Name, "interval", m.Interval)

	// Run once immediately at startup.
	executeMonitor(ctx, m, health)

	ticker := time.NewTicker(m.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("monitor stopping", "name", m.Name)
			return
		case <-ticker.C:
			executeMonitor(ctx, m, health)
		}
	}
}

// executeMonitor runs the monitor function once, records its health in the
// database, and handles consecutive failure logic.
func executeMonitor(ctx context.Context, m Monitor, health *monitorHealth) {
	health.mu.Lock()
	if time.Now().Before(health.pausedUntil) {
		health.mu.Unlock()
		slog.Warn("monitor paused due to repeated failures", "name", m.Name, "until", health.pausedUntil)
		return
	}
	health.mu.Unlock()

	start := time.Now()
	err := m.Run(ctx)
	durationMs := time.Since(start).Milliseconds()

	success := err == nil
	var lastError string
	if err != nil {
		lastError = err.Error()
		slog.Error("monitor failed", "name", m.Name, "error", err, "duration_ms", durationMs)
	} else {
		slog.Info("monitor completed", "name", m.Name, "duration_ms", durationMs)
	}

	// Persist state to the database.
	recordMonitorState(m.Name, success, durationMs, lastError)

	// Track consecutive failures.
	health.mu.Lock()
	defer health.mu.Unlock()

	if success {
		health.failures = 0
		return
	}

	health.failures++

	if health.failures == failureNotifyThreshold {
		slog.Warn("monitor has failed multiple times", "name", m.Name, "consecutive", health.failures)
		Notify(
			fmt.Sprintf("GTD Agent: %s monitor failing", m.Name),
			fmt.Sprintf("The %s monitor has failed %d times in a row. Last error: %s", m.Name, health.failures, lastError),
		)
	}

	if health.failures >= failurePauseThreshold {
		health.pausedUntil = time.Now().Add(pauseDuration)
		slog.Warn("pausing monitor due to excessive failures",
			"name", m.Name,
			"consecutive", health.failures,
			"paused_until", health.pausedUntil,
		)
		Notify(
			fmt.Sprintf("GTD Agent: %s monitor paused", m.Name),
			fmt.Sprintf("The %s monitor has been paused for 1 hour after %d consecutive failures.", m.Name, health.failures),
		)
	}
}

// recordMonitorState upserts the monitor_state table with the latest run info.
func recordMonitorState(name string, success bool, durationMs int64, lastError string) {
	d := db.Get()
	_, err := d.Exec(`
		INSERT INTO monitor_state (name, last_run, duration_ms, success, last_error)
		VALUES (?, datetime('now'), ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			last_run = datetime('now'),
			duration_ms = excluded.duration_ms,
			success = excluded.success,
			last_error = excluded.last_error`,
		name, durationMs, success, lastError,
	)
	if err != nil {
		slog.Error("failed to record monitor state", "name", name, "error", err)
	}
}

// hasNotificationBeenSent checks whether a notification with the given entity
// ID and type has already been sent. If maxAge is positive, only
// notifications sent within that duration are considered.
func hasNotificationBeenSent(entityID, notifyType string, maxAge time.Duration) bool {
	d := db.Get()
	var sentAt string
	var err error

	if maxAge > 0 {
		cutoff := time.Now().Add(-maxAge).UTC().Format("2006-01-02 15:04:05")
		err = d.QueryRow(`
			SELECT sent_at FROM notifications_sent
			WHERE entity_id = ? AND notify_type = ? AND sent_at >= ?`,
			entityID, notifyType, cutoff,
		).Scan(&sentAt)
	} else {
		err = d.QueryRow(`
			SELECT sent_at FROM notifications_sent
			WHERE entity_id = ? AND notify_type = ?`,
			entityID, notifyType,
		).Scan(&sentAt)
	}

	return err == nil && sentAt != ""
}

// recordNotificationSent inserts or updates a record in the notifications_sent
// table so we can deduplicate alerts.
func recordNotificationSent(entityID, notifyType string) {
	d := db.Get()
	_, err := d.Exec(`
		INSERT INTO notifications_sent (entity_id, notify_type, sent_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(entity_id, notify_type) DO UPDATE SET
			sent_at = datetime('now')`,
		entityID, notifyType,
	)
	if err != nil {
		slog.Error("failed to record notification", "entity_id", entityID, "notify_type", notifyType, "error", err)
	}
}

// queryScalarInt is a convenience helper that runs a query expected to return
// a single integer value.
func queryScalarInt(d *sql.DB, query string, args ...any) (int, error) {
	var n int
	err := d.QueryRow(query, args...).Scan(&n)
	return n, err
}
