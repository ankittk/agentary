-- 008_task_reviews.sql
-- Agent-to-agent code review: store review submissions per task.

CREATE TABLE IF NOT EXISTS task_reviews (
  review_id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id INTEGER NOT NULL,
  team_id TEXT NOT NULL,
  reviewer_agent TEXT NOT NULL,
  outcome TEXT NOT NULL,
  comments TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks(task_id) ON DELETE CASCADE,
  FOREIGN KEY (team_id) REFERENCES teams(team_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_task_reviews_task ON task_reviews(task_id);
