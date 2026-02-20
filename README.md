# gtd — Getting Things Done CLI

A command-line productivity system implementing David Allen's [Getting Things Done](https://gettingthingsdone.com/) methodology, with Google Workspace integration, AI-powered features, and an autonomous background agent.

## Features

- **Full GTD workflow** — Capture, process, organize, review, and do
- **Google Workspace** — Gmail, Calendar, and Chat integration via the [`gog`](https://github.com/deenaik/gog) CLI
- **AI-powered** — Smart capture, auto-categorization, email summarization, scheduling suggestions, and weekly insights (OpenAI)
- **Background agent** — Daemon that monitors email, calendar, deadlines, and productivity metrics with macOS desktop notifications
- **Fast & portable** — Pure Go with embedded SQLite (no CGO), single binary

## Prerequisites

- **Go 1.21+**
- **[gog CLI](https://github.com/deenaik/gog)** — Authenticated with your Google account (`gog auth login`)
- **OpenAI API key** (optional) — Required only for AI features

## Installation

```bash
git clone https://github.com/deenaik/gtd-cli.git
cd gtd-cli
make build
```

The binary is built to `bin/gtd`. To install it to your PATH:

```bash
make install
```

## Quick Start

```bash
# Initialize config and database (~/.gtd/)
gtd config init

# Capture something to your inbox
gtd add "Call dentist to schedule cleaning"
gtd add "Review Q3 budget report by Friday"

# Process your inbox interactively (categorize each item)
gtd process

# View your dashboard
gtd
```

## Usage

Running `gtd` with no arguments shows your daily dashboard: overdue tasks, items due today, top next actions, and inbox count.

### Capture & Process

```bash
gtd add "Buy groceries for the week"      # Quick capture to inbox
gtd add --ai "Call Bob about Q3 on Monday" # AI-parsed structured task
gtd inbox                                  # List unprocessed items
gtd inbox --count                          # Just the count
gtd process                                # Interactive GTD processing
gtd process --ai                           # With AI categorization suggestions
```

### Organize

```bash
# Tasks by GTD category
gtd next                            # Next actions
gtd next --context office           # Filter by context
gtd next --project "Q1 Planning"    # Filter by project
gtd waiting                         # Waiting-for items
gtd waiting --who "Bob"             # Filter by delegated person
gtd someday                         # Someday/maybe list

# Manage tasks
gtd done 1 2 3                      # Mark tasks complete
gtd edit 5 --priority 3 --due 2025-03-15 --context phone
gtd edit 5 --delegate "Alice" --category waiting_for
gtd delete 7

# Projects & Contexts
gtd projects                        # List projects
gtd projects add "Website Redesign" --area Work --due 2025-04-01
gtd projects show 1                 # Project detail + its tasks
gtd projects archive 1              # Mark complete
gtd contexts add office             # Create context
gtd contexts list
```

### Search & Review

```bash
gtd search "budget report"          # Full-text search across tasks
gtd today                           # Daily dashboard
gtd today --tomorrow                # Tomorrow's view
gtd review start                    # Guided weekly review
gtd review start --ai               # With AI-generated insights
gtd review status                   # Review history
```

### Gmail Integration

```bash
gtd mail inbox                      # Recent emails
gtd mail inbox --max 20 --query "from:boss"
gtd mail get <message-id>           # Full email details
gtd mail capture <message-id>       # Create inbox item from email
gtd mail reply <message-id> --body "Thanks, will do."
gtd mail search "quarterly report"
gtd mail summarize <message-id>     # AI-powered thread summary
```

### Calendar Integration

```bash
gtd cal                             # Today's events (default)
gtd cal today                       # Same as above
gtd cal week                        # This week Mon-Sun
gtd cal create --title "Team Sync" --from 2025-03-10T09:00:00 --to 2025-03-10T10:00:00
gtd cal freebusy                    # Today 9am-5pm (default)
gtd cal freebusy --from 2025-03-10T09:00:00 --to 2025-03-10T17:00:00
gtd cal schedule 5                  # AI suggests best time for task #5
```

### Google Chat

```bash
gtd chat dm alice@company.com "Meeting moved to 3pm"
gtd chat notify alice@company.com 5   # Send task #5 details as DM
```

### AI Features

Requires `OPENAI_API_KEY` to be set (in `.env` or environment).

```bash
gtd ai suggest 5                    # AI suggestions for a task
gtd ai categorize                   # AI categorize all inbox items
gtd ai insights                     # Weekly productivity analysis
gtd ai schedule 5                   # AI suggest time slot for task
gtd ai usage                        # Today's token usage
```

### Background Agent

The agent runs as a daemon, monitoring your email, calendar, and deadlines autonomously.

```bash
gtd agent start                     # Launch background daemon
gtd agent status                    # Running state + monitor health
gtd agent logs                      # Tail log file
gtd agent logs -f                   # Follow mode
gtd agent logs -n 100               # Last 100 lines
gtd agent stop                      # Graceful shutdown
gtd agent run                       # Foreground mode (for debugging)
```

**What the agent does:**

| Monitor | Interval | Function |
|---------|----------|----------|
| Email | 5 min | Polls unread email, classifies importance (AI or rule-based), auto-captures high-priority emails to inbox |
| Calendar | 1 min | Sends 15-minute-before reminders, detects scheduling conflicts |
| Deadline | 15 min | Alerts for overdue tasks (re-notifies every 4h) and approaching deadlines (24h warning) |
| Progress | 1 hour | Records productivity metrics, sends daily summary at ~5-6 PM |

## Configuration

Configuration is stored at `~/.gtd/config.yaml`. Values can be overridden by environment variables.

| Setting | Config Key | Env Var | Default |
|---------|-----------|---------|---------|
| Data directory | `data_dir` | `GTD_DATA_DIR` | `~/.gtd` |
| OpenAI API key | — | `OPENAI_API_KEY` | — |
| OpenAI model | `openai_model` | — | `gpt-4o-mini` |
| Daily token budget | `token_budget` | `GTD_TOKEN_BUDGET` | `100000` |
| gog CLI path | `gog_path` | `GTD_GOG_PATH` | `gog` |
| Chat notifications | `chat_notify` | — | `false` |

You can also create a `.env` file in your working directory:

```
OPENAI_API_KEY=sk-...
```

## Data Storage

All data lives in `~/.gtd/`:

| File | Purpose |
|------|---------|
| `config.yaml` | User configuration |
| `gtd.db` | SQLite database (WAL mode) |
| `agent.pid` | Background agent PID file |
| `agent.log` | Agent structured logs (JSON) |

---

## Developer Guide

### Project Structure

```
├── main.go                     # Entry point → cmd.Execute()
├── cmd/                        # Cobra CLI commands (one file per command group)
├── internal/
│   ├── config/                 # Config loading (YAML + .env + env vars)
│   ├── db/                     # SQLite connection + schema migrations
│   ├── model/                  # Domain structs (Task, InboxItem, Project, Context)
│   ├── store/                  # Data access layer (one store per entity)
│   ├── google/                 # gog CLI wrapper (gmail, calendar, chat)
│   ├── ai/                     # OpenAI integration (client, prompts, schemas, features)
│   ├── agent/                  # Background daemon + monitors + notifications
│   └── ui/                     # Terminal rendering (tables, prompts)
├── Makefile
├── go.mod
└── .env                        # Local env vars (gitignored)
```

### Build & Test

```bash
make build          # Build → bin/gtd
make clean build    # Clean rebuild
make test           # go test ./...
make tidy           # go mod tidy
go vet ./...        # Static analysis
```

### Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `modernc.org/sqlite` | Pure-Go SQLite driver (no CGO) |
| `github.com/sashabaranov/go-openai` | OpenAI client with structured output |
| `github.com/charmbracelet/huh` | Interactive terminal prompts |
| `github.com/charmbracelet/lipgloss` | Terminal styling and tables |
| `github.com/joho/godotenv` | `.env` file loading |
| `gopkg.in/yaml.v3` | YAML config parsing |
| `golang.org/x/time` | Rate limiter for OpenAI API |

### Architecture Notes

**Singletons**: Both `config.Get()` and `db.Get()` use `sync.Once` for lazy initialization. The config singleton is loaded in `PersistentPreRunE` before any command runs.

**Database**: Schema is defined as a single idempotent SQL block in `internal/db/migrations.go` using `CREATE TABLE IF NOT EXISTS`. It runs on every command invocation via `db.Migrate()`. FTS5 is kept in sync via SQLite triggers.

**Google integration**: All Google API calls go through the `gog` CLI via shell exec. `RunJSON()` adds `-j --results-only` flags for machine-readable output. Note: `gmail get` uses a different JSON structure than `gmail list` — see `gogEmailGetResponse` in `gmail.go`.

**AI features**: All LLM calls go through `ai.Complete()` which enforces rate limiting (10 req/min), exponential backoff retries (3 attempts), daily token budget, and usage tracking. Structured output uses OpenAI's JSON schema mode. Each AI feature has a fallback path (rule-based or graceful error) when the API key is missing or budget is exceeded.

**Agent daemon**: Uses a re-exec pattern (`os/exec` with `Setpgid: true`) since Go cannot fork safely. The parent writes the child PID to `~/.gtd/agent.pid` and exits. Each monitor runs in its own goroutine with independent `time.Ticker` intervals. Health is tracked in the `monitor_state` DB table; consecutive failures trigger escalating responses (notification at 3, pause at 10).

**Notifications**: macOS desktop via `osascript`, optional Google Chat DM. Deduplication via `notifications_sent` table with `(entity_id, notify_type)` uniqueness constraint.

### Adding a New Command

1. Create `cmd/<name>.go`
2. Define a `cobra.Command` and register it in an `init()` function via `rootCmd.AddCommand()`
3. Use `db.Get()` and `store.New<Entity>Store(db)` for data access
4. Use `ui.PrintSection()`, `ui.PrintSuccess()`, `ui.NewTable()` for output

### Adding a New AI Feature

1. Add the system prompt to `internal/ai/prompts.go`
2. Add a JSON schema function to `internal/ai/schemas.go` (if using structured output)
3. Create `internal/ai/<feature>.go` with a public function that calls `ai.Complete()`
4. Wire it into a CLI command in `cmd/ai.go` or the relevant command file

### Adding a New Agent Monitor

1. Create `internal/agent/monitor_<name>.go` with a `Run<Name>Monitor(ctx context.Context) error` function
2. Register it in the `monitors` slice in `scheduler.go:RunForeground()`
3. Add a config interval field in `internal/config/config.go` if needed

## License

MIT
