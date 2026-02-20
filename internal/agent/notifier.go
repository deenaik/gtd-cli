package agent

import (
	"log/slog"
	"os/exec"
	"strings"

	"github.com/deenaik/gtd-cli/internal/config"
	"github.com/deenaik/gtd-cli/internal/google"
)

// Notify sends a macOS desktop notification using osascript. On non-macOS
// systems this is a no-op that logs the notification.
func Notify(title, message string) {
	// Escape double quotes and backslashes for AppleScript strings.
	safeTitle := escapeAppleScript(title)
	safeMessage := escapeAppleScript(message)

	script := `display notification "` + safeMessage + `" with title "` + safeTitle + `"`

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		slog.Warn("notify: osascript failed, logging instead",
			"title", title, "message", message, "error", err)
	}

	slog.Info("notification sent", "title", title)
}

// NotifyChat sends a direct message to the user via Google Chat if
// ChatNotify is enabled in the configuration.
func NotifyChat(message string) {
	cfg := config.Get()
	if !cfg.ChatNotify {
		return
	}

	if cfg.UserEmail == "" {
		slog.Warn("notify chat: user_email not configured, skipping")
		return
	}

	if err := google.SendDM(cfg.UserEmail, message); err != nil {
		slog.Error("notify chat: send DM failed", "error", err)
	} else {
		slog.Info("chat notification sent")
	}
}

// NotifyAll sends both a macOS desktop notification and a Google Chat message
// (if chat notifications are enabled).
func NotifyAll(title, message string) {
	Notify(title, message)
	NotifyChat(title + "\n" + message)
}

// escapeAppleScript escapes characters that are special in AppleScript string
// literals (double quotes and backslashes).
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	// Replace newlines with spaces for the notification text since
	// AppleScript display notification does not handle literal newlines well.
	s = strings.ReplaceAll(s, "\n", " - ")
	return s
}
