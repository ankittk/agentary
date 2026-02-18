-- 001_init.sql (PostgreSQL)
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS teams (
  team_id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  created_at BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS agents (
  agent_id TEXT PRIMARY KEY,
  team_id TEXT NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  role TEXT NOT NULL,
  created_at BIGINT NOT NULL,
  UNIQUE(team_id, name)
);

CREATE TABLE IF NOT EXISTS repos (
  repo_id TEXT PRIMARY KEY,
  team_id TEXT NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  source TEXT NOT NULL,
  approval TEXT NOT NULL DEFAULT 'manual',
  test_cmd TEXT,
  created_at BIGINT NOT NULL,
  UNIQUE(team_id, name)
);

CREATE TABLE IF NOT EXISTS workflows (
  workflow_id TEXT PRIMARY KEY,
  team_id TEXT NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  version INTEGER NOT NULL,
  source_path TEXT NOT NULL,
  created_at BIGINT NOT NULL,
  UNIQUE(team_id, name, version)
);

CREATE TABLE IF NOT EXISTS tasks (
  task_id BIGSERIAL PRIMARY KEY,
  team_id TEXT NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  assignee TEXT,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS messages (
  message_id BIGSERIAL PRIMARY KEY,
  team_id TEXT NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
  sender TEXT NOT NULL,
  recipient TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at BIGINT NOT NULL,
  processed_at BIGINT
);

CREATE TABLE IF NOT EXISTS activity (
  activity_id BIGSERIAL PRIMARY KEY,
  team_id TEXT NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
  agent TEXT NOT NULL,
  type TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  created_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_team_status ON tasks(team_id, status);
CREATE INDEX IF NOT EXISTS idx_tasks_team_created ON tasks(team_id, created_at);
CREATE INDEX IF NOT EXISTS idx_messages_team_created ON messages(team_id, created_at);
CREATE INDEX IF NOT EXISTS idx_activity_team_created ON activity(team_id, created_at);
