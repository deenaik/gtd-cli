package google

import (
	"encoding/json"
	"fmt"
)

// EventDateTime represents a Google Calendar event time which can be either
// a date (all-day event) or a dateTime (timed event).
type EventDateTime struct {
	Date     string `json:"date"`
	DateTime string `json:"dateTime"`
}

// String returns the most specific time representation available.
func (e EventDateTime) String() string {
	if e.DateTime != "" {
		return e.DateTime
	}
	return e.Date
}

// CalendarEvent represents a Google Calendar event returned by the gog CLI.
type CalendarEvent struct {
	ID          string        `json:"id"`
	Title       string        `json:"summary"`
	Start       EventDateTime `json:"start"`
	End         EventDateTime `json:"end"`
	Location    string        `json:"location"`
	Description string        `json:"description"`
	Status      string        `json:"status"`
	Organizer   json.RawMessage `json:"organizer"`
}

// FreeBusySlot represents a busy time range returned by a free/busy query.
type FreeBusySlot struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// ListEvents retrieves calendar events between the given dates.
// Dates should be in YYYY-MM-DD format. Either parameter can be empty to omit it.
func ListEvents(from, to string) ([]CalendarEvent, error) {
	args := []string{"calendar", "events"}
	if from != "" {
		args = append(args, "--from", from)
	}
	if to != "" {
		args = append(args, "--to", to)
	}
	var events []CalendarEvent
	if err := RunJSON(&events, args...); err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return events, nil
}

// GetEvent retrieves a single calendar event by its ID.
func GetEvent(id string) (*CalendarEvent, error) {
	var event CalendarEvent
	if err := RunJSON(&event, "calendar", "event", "primary", id); err != nil {
		return nil, fmt.Errorf("get event %s: %w", id, err)
	}
	return &event, nil
}

// CreateEvent creates a new calendar event with the given title and time range.
// The from and to parameters should be in datetime format (e.g. 2024-01-15T09:00:00).
func CreateEvent(title, from, to string) (*CalendarEvent, error) {
	var event CalendarEvent
	if err := RunJSON(&event, "calendar", "create", "primary", "--summary", title, "--from", from, "--to", to); err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}
	return &event, nil
}

// FreeBusy queries for busy time slots between the given datetimes.
func FreeBusy(from, to string) ([]FreeBusySlot, error) {
	var slots []FreeBusySlot
	if err := RunJSON(&slots, "calendar", "freebusy", "primary", "--from", from, "--to", to); err != nil {
		return nil, fmt.Errorf("freebusy query: %w", err)
	}
	return slots, nil
}
