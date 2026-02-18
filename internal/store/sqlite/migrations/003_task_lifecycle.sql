-- 003_task_lifecycle.sql
-- Add attempt_count and support failed status; no schema change required for 'failed' (status is TEXT).

ALTER TABLE tasks ADD COLUMN attempt_count INTEGER NOT NULL DEFAULT 0;
