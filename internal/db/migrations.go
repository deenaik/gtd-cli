package db

const schema = `
CREATE TABLE IF NOT EXISTS contexts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    created_at  DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS projects (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    description TEXT DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'active'
                CHECK (status IN ('active','completed','on_hold','dropped')),
    area        TEXT DEFAULT '',
    due_date    DATE,
    created_at  DATETIME DEFAULT (datetime('now')),
    updated_at  DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS inbox_items (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT NOT NULL,
    body        TEXT DEFAULT '',
    source      TEXT DEFAULT 'manual' CHECK (source IN ('manual','email','chat','calendar')),
    source_ref  TEXT DEFAULT '',
    source_meta TEXT DEFAULT '{}',
    processed   INTEGER DEFAULT 0,
    created_at  DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS tasks (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    title           TEXT NOT NULL,
    description     TEXT DEFAULT '',
    category        TEXT NOT NULL DEFAULT 'next_action'
                    CHECK (category IN ('next_action','waiting_for','someday_maybe')),
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','active','done','delegated','deferred')),
    priority        INTEGER DEFAULT 0,
    energy          TEXT DEFAULT 'medium' CHECK (energy IN ('low','medium','high')),
    time_estimate   INTEGER DEFAULT 0,
    project_id      INTEGER REFERENCES projects(id) ON DELETE SET NULL,
    context_id      INTEGER REFERENCES contexts(id) ON DELETE SET NULL,
    delegated_to    TEXT DEFAULT '',
    due_date        DATE,
    defer_until     DATE,
    completed_at    DATETIME,
    source          TEXT DEFAULT 'manual',
    source_ref      TEXT DEFAULT '',
    calendar_event_id TEXT DEFAULT '',
    created_at      DATETIME DEFAULT (datetime('now')),
    updated_at      DATETIME DEFAULT (datetime('now'))
);

CREATE VIRTUAL TABLE IF NOT EXISTS tasks_fts USING fts5(title, description, content=tasks, content_rowid=id);

CREATE TRIGGER IF NOT EXISTS tasks_ai AFTER INSERT ON tasks BEGIN
    INSERT INTO tasks_fts(rowid, title, description) VALUES (new.id, new.title, new.description);
END;
CREATE TRIGGER IF NOT EXISTS tasks_ad AFTER DELETE ON tasks BEGIN
    INSERT INTO tasks_fts(tasks_fts, rowid, title, description) VALUES('delete', old.id, old.title, old.description);
END;
CREATE TRIGGER IF NOT EXISTS tasks_au AFTER UPDATE ON tasks BEGIN
    INSERT INTO tasks_fts(tasks_fts, rowid, title, description) VALUES('delete', old.id, old.title, old.description);
    INSERT INTO tasks_fts(rowid, title, description) VALUES (new.id, new.title, new.description);
END;

CREATE TABLE IF NOT EXISTS task_activity (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id    INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    action     TEXT NOT NULL,
    created_at DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS weekly_reviews (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    week_start      DATE NOT NULL,
    tasks_completed INTEGER DEFAULT 0,
    tasks_created   INTEGER DEFAULT 0,
    notes           TEXT DEFAULT '',
    ai_insights     TEXT DEFAULT '',
    completed_at    DATETIME,
    created_at      DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS processed_emails (
    thread_id    TEXT PRIMARY KEY,
    importance   TEXT,
    processed_at DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS notifications_sent (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_id   TEXT NOT NULL,
    notify_type TEXT NOT NULL,
    sent_at     DATETIME DEFAULT (datetime('now')),
    UNIQUE(entity_id, notify_type)
);

CREATE TABLE IF NOT EXISTS monitor_state (
    name       TEXT PRIMARY KEY,
    last_run   DATETIME,
    duration_ms INTEGER,
    success    BOOLEAN,
    last_error TEXT
);

CREATE TABLE IF NOT EXISTS token_usage (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    feature           TEXT NOT NULL,
    model             TEXT NOT NULL,
    prompt_tokens     INTEGER NOT NULL,
    completion_tokens INTEGER NOT NULL,
    total_tokens      INTEGER NOT NULL,
    created_at        DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS progress_metrics (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    completed_today     INTEGER,
    pending_count       INTEGER,
    overdue_count       INTEGER,
    inbox_count         INTEGER,
    avg_completion_rate REAL,
    created_at          DATETIME DEFAULT (datetime('now'))
);
`
