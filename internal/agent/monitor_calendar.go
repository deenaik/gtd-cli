package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/deenaik/gtd-cli/internal/google"
)

// RunCalendarMonitor checks upcoming calendar events and sends reminders for
// events starting within 15 minutes. It also detects time conflicts between
// overlapping events.
func RunCalendarMonitor(ctx context.Context) error {
	now := time.Now()

	// Look 20 minutes ahead to catch events that need a 15-minute warning.
	from := now.Format(time.RFC3339)
	to := now.Add(20 * time.Minute).Format(time.RFC3339)

	events, err := google.ListEvents(from, to)
	if err != nil {
		return fmt.Errorf("list events: %w", err)
	}

	if len(events) == 0 {
		slog.Debug("calendar monitor: no upcoming events")
		return nil
	}

	slog.Info("calendar monitor: checking events", "count", len(events))

	// Send 15-minute reminders for upcoming events.
	for _, event := range events {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		startTime, err := parseEventTime(event.Start.String())
		if err != nil {
			slog.Warn("calendar monitor: parse event start time",
				"event", event.Title, "start", event.Start.String(), "error", err)
			continue
		}

		untilStart := time.Until(startTime)

		// Send reminder if event starts within 15 minutes (and hasn't
		// started yet).
		if untilStart > 0 && untilStart <= 15*time.Minute {
			notifyKey := fmt.Sprintf("cal-reminder-%s", event.ID)

			if hasNotificationBeenSent(notifyKey, "calendar_reminder", 0) {
				continue
			}

			minutesLeft := int(untilStart.Minutes())
			if minutesLeft < 1 {
				minutesLeft = 1
			}

			location := ""
			if event.Location != "" {
				location = fmt.Sprintf("\nLocation: %s", event.Location)
			}

			NotifyAll(
				fmt.Sprintf("Calendar: %s in %d min", event.Title, minutesLeft),
				fmt.Sprintf("Starting at %s%s", startTime.Format("3:04 PM"), location),
			)
			recordNotificationSent(notifyKey, "calendar_reminder")

			slog.Info("calendar monitor: sent reminder",
				"event", event.Title, "minutes_left", minutesLeft)
		}
	}

	// Detect time conflicts among the upcoming events.
	detectConflicts(ctx, events)

	return nil
}

// detectConflicts checks for overlapping events and sends a notification if
// any are found.
func detectConflicts(ctx context.Context, events []google.CalendarEvent) {
	if len(events) < 2 {
		return
	}

	type parsedEvent struct {
		event google.CalendarEvent
		start time.Time
		end   time.Time
	}

	var parsed []parsedEvent
	for _, e := range events {
		start, err := parseEventTime(e.Start.String())
		if err != nil {
			continue
		}
		end, err := parseEventTime(e.End.String())
		if err != nil {
			// Default to 30 minutes if end time is unparseable.
			end = start.Add(30 * time.Minute)
		}
		parsed = append(parsed, parsedEvent{event: e, start: start, end: end})
	}

	// O(n^2) comparison is fine for small event counts.
	for i := 0; i < len(parsed); i++ {
		for j := i + 1; j < len(parsed); j++ {
			if ctx.Err() != nil {
				return
			}

			a := parsed[i]
			b := parsed[j]

			// Two events overlap if one starts before the other ends.
			if a.start.Before(b.end) && b.start.Before(a.end) {
				// Build a dedup key from sorted event IDs.
				idA, idB := a.event.ID, b.event.ID
				if idA > idB {
					idA, idB = idB, idA
				}
				conflictKey := fmt.Sprintf("cal-conflict-%s-%s", idA, idB)

				if hasNotificationBeenSent(conflictKey, "calendar_conflict", 0) {
					continue
				}

				NotifyAll(
					"Calendar Conflict Detected",
					fmt.Sprintf("%q (%s) overlaps with %q (%s)",
						a.event.Title, a.start.Format("3:04 PM"),
						b.event.Title, b.start.Format("3:04 PM"),
					),
				)
				recordNotificationSent(conflictKey, "calendar_conflict")

				slog.Warn("calendar monitor: conflict detected",
					"event_a", a.event.Title, "event_b", b.event.Title)
			}
		}
	}
}

// parseEventTime attempts to parse an event time string in several common
// formats returned by Google Calendar.
func parseEventTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	// Try RFC3339 first (most common for datetime events).
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try RFC3339 without timezone offset (some APIs return this).
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local); err == nil {
		return t, nil
	}

	// Date-only format (all-day events).
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t, nil
	}

	// Try a few more common variations.
	formats := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z",
		time.RFC1123Z,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized time format: %q", s)
}
