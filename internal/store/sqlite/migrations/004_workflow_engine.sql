-- 004_workflow_engine.sql
-- Add workflow stages, transitions, and current_stage to tasks for minimal workflow engine.

-- Stages for a workflow: name, type (agent|human|auto|terminal), outcomes (comma-separated or empty for terminal)
CREATE TABLE IF NOT EXISTS workflow_stages (
  workflow_id TEXT NOT NULL,
  stage_name TEXT NOT NULL,
  stage_type TEXT NOT NULL,
  outcomes TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (workflow_id, stage_name)
);

-- Transitions: (from_stage, outcome) -> to_stage
CREATE TABLE IF NOT EXISTS workflow_transitions (
  workflow_id TEXT NOT NULL,
  from_stage TEXT NOT NULL,
  outcome TEXT NOT NULL,
  to_stage TEXT NOT NULL,
  PRIMARY KEY (workflow_id, from_stage, outcome)
);

-- Task workflow state: which workflow and current stage (NULL = legacy flat status)
ALTER TABLE tasks ADD COLUMN workflow_id TEXT;
ALTER TABLE tasks ADD COLUMN current_stage TEXT;

CREATE INDEX IF NOT EXISTS idx_tasks_workflow_stage ON tasks(workflow_id, current_stage);
