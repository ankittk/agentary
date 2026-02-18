-- 006_git_worktrees.sql
-- Git worktree path, branch name, base SHA per task (for worktree create/delete and cancel cleanup).

ALTER TABLE tasks ADD COLUMN worktree_path TEXT;
ALTER TABLE tasks ADD COLUMN branch_name TEXT;
ALTER TABLE tasks ADD COLUMN base_sha TEXT;
ALTER TABLE tasks ADD COLUMN repo_name TEXT;
