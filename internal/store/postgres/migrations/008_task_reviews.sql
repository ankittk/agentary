CREATE TABLE IF NOT EXISTS task_reviews (
  review_id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
  team_id TEXT NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
  reviewer_agent TEXT NOT NULL,
  outcome TEXT NOT NULL,
  comments TEXT NOT NULL DEFAULT '',
  created_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_reviews_task ON task_reviews(task_id);
