ALTER TABLE tasks ADD COLUMN IF NOT EXISTS dri TEXT;

CREATE TABLE IF NOT EXISTS task_comments (
  comment_id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
  team_id TEXT NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
  author TEXT NOT NULL,
  body TEXT NOT NULL,
  created_at BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS task_attachments (
  attachment_id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
  team_id TEXT NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
  file_path TEXT NOT NULL,
  created_at BIGINT NOT NULL,
  UNIQUE(task_id, file_path)
);

CREATE TABLE IF NOT EXISTS task_dependencies (
  task_id BIGINT NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
  depends_on_task_id BIGINT NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
  PRIMARY KEY (task_id, depends_on_task_id)
);

CREATE INDEX IF NOT EXISTS idx_task_comments_task ON task_comments(task_id);
CREATE INDEX IF NOT EXISTS idx_task_attachments_task ON task_attachments(task_id);
CREATE INDEX IF NOT EXISTS idx_task_dependencies_task ON task_dependencies(task_id);
