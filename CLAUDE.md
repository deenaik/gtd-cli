# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build              # Build binary to bin/gtd
make clean build        # Clean rebuild
make tidy               # go mod tidy
make test               # go test ./...
go vet ./...            # Lint check
make init               # Build + run `gtd config init` (creates ~/.gtd/)
```

The binary is at `bin/gtd`. First-time setup requires `gtd config init` which creates `~/.gtd/` with `config.yaml` and `gtd.db`.

## Architecture

This is a Go CLI implementing the Getting Things Done methodology. It has four major subsystems:

### 1. CLI Layer (`cmd/`)
Cobra-based commands registered via `init()` functions to `rootCmd`. Running bare `gtd` delegates to `todayRun` (the dashboard). The `PersistentPreRunE` on `rootCmd` ensures `~/.gtd/` exists and runs DB migrations before every command (except `config init`).

### 2. Data Layer (`internal/model/`, `internal/store/`, `internal/db/`)
- **SQLite** via `modernc.org/sqlite` (pure Go, no CGO). Single connection with WAL mode, foreign keys, 5s busy timeout.
- **Schema** lives in `internal/db/migrations.go` as a single `CREATE TABLE IF NOT EXISTS` block — idempotent, runs on every command.
- **Models** are plain structs with `sql.NullInt64`/`sql.NullString` for optional FK/date fields. `Task` has joined fields `ProjectName`/`ContextName` populated by store queries.
- **Stores** take `*sql.DB` via constructor (`NewTaskStore(db)`) and own all SQL. `TaskStore.List()` uses a builder pattern with `TaskFilter` struct.
- **FTS**: `tasks_fts` (FTS5 content table) is kept in sync via triggers on `tasks`. Search via `TaskStore.Search()`.

### 3. Google Workspace Integration (`internal/google/`)
Shells out to the `gog` CLI (path from `config.GogPath`). Two executor functions in `gog.go`:
- `Run(args...)` — raw execution, returns stdout bytes
- `RunJSON(result, args...)` — prepends `-j --results-only`, unmarshals JSON into `result`

Key gotchas with `gog` CLI syntax:
- Gmail: `gmail list <query>` (query is positional), `gmail get <id>` (returns nested `{headers, message, body}` structure — different from list), `gmail send --reply-to-message-id <id> --body <body>` (no `gmail reply` command)
- Calendar: `calendar events` (not `list`), `calendar event primary <id>`, `calendar create primary --summary <title>`, `calendar freebusy primary`; events have nested start/end as `EventDateTime{Date, DateTime}` not plain strings
- Chat: `chat dm send <email> --text <message>`

### 4. Background Agent (`internal/agent/`)
- **Daemon** (`daemon.go`): `Start()` re-execs the binary with `agent run` in a detached process group (Go can't fork). PID file at `~/.gtd/agent.pid`, logs to `~/.gtd/agent.log`.
- **Scheduler** (`scheduler.go`): Each monitor runs in its own goroutine with `time.Ticker`. Health tracked per-monitor: 3 consecutive failures → notification, 10 → 1-hour pause.
- **Monitors**: `monitor_email.go` (polls unread, classifies importance via LLM with rule-based fallback, auto-captures high-priority), `monitor_calendar.go` (15-min reminders, conflict detection), `monitor_deadline.go` (overdue/approaching alerts with tiered renotification), `monitor_progress.go` (hourly metrics snapshot, daily summary at 5-6 PM).
- **Notification dedup**: `notifications_sent` table keyed on `(entity_id, notify_type)`. Helpers `hasNotificationBeenSent()`/`recordNotificationSent()` in `scheduler.go`.

### 5. AI/LLM Integration (`internal/ai/`)
- OpenAI client via `github.com/sashabaranov/go-openai` with structured output (JSON schema mode).
- Singleton client with rate limiter (10 req/min) and exponential backoff retry (3 attempts).
- Daily token budget tracked in `token_usage` table; returns `ErrBudgetExceeded` when exceeded.
- All features use `ai.Complete()` which handles rate limiting, retries, budget checks, and usage tracking.
- Each feature (capture, categorize, summarize, insights, schedule) has its own file with prompt in `prompts.go` and schema in `schemas.go`.

## Configuration

Config loaded in order: defaults → `~/.gtd/config.yaml` → `.env` file → environment variables.

Key env vars: `OPENAI_API_KEY`, `GTD_DATA_DIR`, `GTD_GOG_PATH`, `GTD_TOKEN_BUDGET`.

## Data Flow

GTD workflow: `gtd add` → `inbox_items` → `gtd process` (interactive) → `tasks` (with category: `next_action`/`waiting_for`/`someday_maybe`). The agent's email monitor also creates `inbox_items` with `source='email'`. Tasks have optional FK links to `projects` and `contexts`.
