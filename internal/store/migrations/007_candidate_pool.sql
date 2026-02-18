-- 007_candidate_pool.sql
-- Optional candidate_agents per stage (comma-separated names); scheduler/manager picks assignee from this pool.

ALTER TABLE workflow_stages ADD COLUMN candidate_agents TEXT DEFAULT '';
