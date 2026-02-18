-- 001_init.sql
-- Initial schema for Agentary (v1)

PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS teams (
  team_id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS agents (
  agent_id TEXT PRIMARY KEY,
  team_id TEXT NOT NULL,
  name TEXT NOT NULL,
  role TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  UNIQUE(team_id, name),
  FOREIGN KEY(team_id) REFERENCES teams(team_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS repos (
  repo_id TEXT PRIMARY KEY,
  team_id TEXT NOT NULL,
  name TEXT NOT NULL,
  source TEXT NOT NULL,
  approval TEXT NOT NULL DEFAULT 'manual',
  test_cmd TEXT,
  created_at INTEGER NOT NULL,
  UNIQUE(team_id, name),
  FOREIGN KEY(team_id) REFERENCES teams(team_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS workflows (
  workflow_id TEXT PRIMARY KEY,
  team_id TEXT NOT NULL,
  name TEXT NOT NULL,
  version INTEGER NOT NULL,
  source_path TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  UNIQUE(team_id, name, version),
  FOREIGN KEY(team_id) REFERENCES teams(team_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tasks (
  task_id INTEGER PRIMARY KEY AUTOINCREMENT,
  team_id TEXT NOT NULL,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  assignee TEXT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  FOREIGN KEY(team_id) REFERENCES teams(team_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS messages (
  message_id INTEGER PRIMARY KEY AUTOINCREMENT,
  team_id TEXT NOT NULL,
  sender TEXT NOT NULL,
  recipient TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  processed_at INTEGER,
  FOREIGN KEY(team_id) REFERENCES teams(team_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS activity (
  activity_id INTEGER PRIMARY KEY AUTOINCREMENT,
  team_id TEXT NOT NULL,
  agent TEXT NOT NULL,
  type TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  FOREIGN KEY(team_id) REFERENCES teams(team_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tasks_team_status ON tasks(team_id, status);
CREATE INDEX IF NOT EXISTS idx_tasks_team_created ON tasks(team_id, created_at);
CREATE INDEX IF NOT EXISTS idx_messages_team_created ON messages(team_id, created_at);
CREATE INDEX IF NOT EXISTS idx_activity_team_created ON activity(team_id, created_at);

