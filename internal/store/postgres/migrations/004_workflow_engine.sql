CREATE TABLE IF NOT EXISTS workflow_stages (
  workflow_id TEXT NOT NULL,
  stage_name TEXT NOT NULL,
  stage_type TEXT NOT NULL,
  outcomes TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (workflow_id, stage_name)
);

CREATE TABLE IF NOT EXISTS workflow_transitions (
  workflow_id TEXT NOT NULL,
  from_stage TEXT NOT NULL,
  outcome TEXT NOT NULL,
  to_stage TEXT NOT NULL,
  PRIMARY KEY (workflow_id, from_stage, outcome)
);

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS workflow_id TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS current_stage TEXT;

CREATE INDEX IF NOT EXISTS idx_tasks_workflow_stage ON tasks(workflow_id, current_stage);
