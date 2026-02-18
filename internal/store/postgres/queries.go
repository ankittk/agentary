package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ankittk/agentary/internal/store"
	"github.com/jackc/pgx/v5"
)

func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("t-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func (s *Store) SeedDemo(ctx context.Context) error {
	teams, err := s.ListTeams(ctx)
	if err != nil {
		return err
	}
	if len(teams) == 0 {
		if _, err := s.CreateTeam(ctx, "default"); err != nil {
			return err
		}
	}
	agents, err := s.ListAgents(ctx, "default")
	if err != nil {
		return nil
	}
	has := map[string]bool{}
	for _, a := range agents {
		has[a.Name] = true
	}
	if !has["agentary"] {
		_ = s.CreateAgent(ctx, "default", "agentary", "manager")
	}
	if !has["alice"] {
		_ = s.CreateAgent(ctx, "default", "alice", "engineer")
	}
	if !has["bob"] {
		_ = s.CreateAgent(ctx, "default", "bob", "engineer")
	}
	tasks, err := s.ListTasks(ctx, "default", 0)
	if err != nil {
		return nil
	}
	if len(tasks) == 0 {
		_, _ = s.CreateTask(ctx, "default", "Welcome to Agentary", "todo", nil)
	}
	return nil
}

func (s *Store) ListTeams(ctx context.Context) ([]store.Team, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT t.name, t.team_id, t.created_at,
  (SELECT COUNT(*) FROM agents a WHERE a.team_id = t.team_id) AS agent_count,
  (SELECT COUNT(*) FROM tasks k WHERE k.team_id = t.team_id) AS task_count
FROM teams t ORDER BY t.created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Team
	for rows.Next() {
		var name, teamID string
		var createdAt int64
		var agentCnt, taskCnt int
		if err := rows.Scan(&name, &teamID, &createdAt, &agentCnt, &taskCnt); err != nil {
			return nil, err
		}
		out = append(out, store.Team{
			Name: name, TeamID: teamID,
			CreatedAt: time.Unix(createdAt, 0).UTC(),
			AgentCount: agentCnt, TaskCount: taskCnt,
		})
	}
	return out, rows.Err()
}

func (s *Store) GetTeamByName(ctx context.Context, name string) (store.Team, error) {
	var t store.Team
	var createdAt int64
	err := s.Pool.QueryRow(ctx, `SELECT name, team_id, created_at FROM teams WHERE name = $1`, name).
		Scan(&t.Name, &t.TeamID, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.Team{}, fmt.Errorf("team not found: %s", name)
		}
		return store.Team{}, err
	}
	t.CreatedAt = time.Unix(createdAt, 0).UTC()
	return t, nil
}

func (s *Store) CreateTeam(ctx context.Context, name string) (store.Team, error) {
	if name == "" {
		return store.Team{}, errors.New("team name required")
	}
	id := randomID()
	now := time.Now().UTC().Unix()
	_, err := s.Pool.Exec(ctx, `INSERT INTO teams(team_id, name, created_at) VALUES($1, $2, $3)`, id, name, now)
	if err != nil {
		return store.Team{}, err
	}
	return store.Team{TeamID: id, Name: name, CreatedAt: time.Unix(now, 0).UTC()}, nil
}

func (s *Store) DeleteTeam(ctx context.Context, name string) error {
	team, err := s.GetTeamByName(ctx, name)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `DELETE FROM teams WHERE team_id = $1`, team.TeamID)
	return err
}

func (s *Store) ListAgents(ctx context.Context, teamName string) ([]store.Agent, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT name, role, created_at FROM agents WHERE team_id = $1 ORDER BY created_at ASC`, team.TeamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Agent
	for rows.Next() {
		var name, role string
		var createdAt int64
		if err := rows.Scan(&name, &role, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, store.Agent{Name: name, Role: role, CreatedAt: time.Unix(createdAt, 0).UTC()})
	}
	return out, rows.Err()
}

func (s *Store) CreateAgent(ctx context.Context, teamName, name, role string) error {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("agent name required")
	}
	if role == "" {
		role = "engineer"
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO agents(agent_id, team_id, name, role, created_at) VALUES($1, $2, $3, $4, $5)`,
		randomID(), team.TeamID, name, role, time.Now().UTC().Unix())
	return err
}

// scanTaskRow scans a row with task columns into *store.Task (used by NextRunnableTaskForTeam, GetTaskByIDAndTeam).
func scanTaskRow(row interface{ Scan(dest ...any) error }) (*store.Task, error) {
	var id int64
	var title, status string
	var assignee, dri, workflowID, currentStage, worktreePath, branchName, baseSHA, repoName *string
	var attemptCount int
	var createdAt, updatedAt int64
	err := row.Scan(&id, &title, &status, &assignee, &dri, &attemptCount, &workflowID, &currentStage, &worktreePath, &branchName, &baseSHA, &repoName, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return &store.Task{
		TaskID: id, Title: title, Status: status, Assignee: assignee, DRI: dri,
		AttemptCount: attemptCount, WorkflowID: workflowID, CurrentStage: currentStage,
		WorktreePath: worktreePath, BranchName: branchName, BaseSHA: baseSHA, RepoName: repoName,
		CreatedAt: time.Unix(createdAt, 0).UTC(), UpdatedAt: time.Unix(updatedAt, 0).UTC(),
	}, nil
}

func toNull(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func (s *Store) ListTasks(ctx context.Context, teamName string, limit int) ([]store.Task, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	q := `SELECT task_id, title, status, assignee, dri, COALESCE(attempt_count,0), workflow_id, current_stage, worktree_path, branch_name, base_sha, repo_name, created_at, updated_at FROM tasks WHERE team_id = $1 ORDER BY created_at DESC`
	args := []any{team.TeamID}
	if limit > 0 {
		q += ` LIMIT $2`
		args = append(args, limit)
	}
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Task
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *task)
	}
	return out, rows.Err()
}

func (s *Store) ListTasksInStage(ctx context.Context, teamName, stage string, limit int) ([]store.Task, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	q := `SELECT task_id, title, status, assignee, dri, COALESCE(attempt_count,0), workflow_id, current_stage, worktree_path, branch_name, base_sha, repo_name, created_at, updated_at FROM tasks WHERE team_id = $1 AND current_stage = $2 ORDER BY updated_at ASC`
	args := []any{team.TeamID, stage}
	if limit > 0 {
		q += ` LIMIT $3`
		args = append(args, limit)
	}
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Task
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *task)
	}
	return out, rows.Err()
}

func (s *Store) CreateTask(ctx context.Context, teamName, title, status string, workflowID *string) (int64, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return 0, err
	}
	if title == "" {
		return 0, errors.New("title required")
	}
	if status == "" {
		status = "todo"
	}
	now := time.Now().UTC().Unix()
	var id int64
	err = s.Pool.QueryRow(ctx, `INSERT INTO tasks(team_id, title, status, created_at, updated_at) VALUES($1, $2, $3, $4, $5) RETURNING task_id`,
		team.TeamID, title, status, now, now).Scan(&id)
	if err != nil {
		return 0, err
	}
	if workflowID != nil && *workflowID != "" {
		initial, err := s.GetWorkflowInitialStage(ctx, *workflowID)
		if err == nil {
			_, _ = s.Pool.Exec(ctx, `UPDATE tasks SET workflow_id=$1, current_stage=$2, updated_at=$3 WHERE task_id=$4`, *workflowID, initial, now, id)
		}
	}
	return id, nil
}

func (s *Store) UpdateTask(ctx context.Context, taskID int64, status string, assignee *string) error {
	now := time.Now().UTC().Unix()
	if status != "" {
		_, err := s.Pool.Exec(ctx, `UPDATE tasks SET status=$1, assignee=$2, updated_at=$3 WHERE task_id=$4`, status, toNull(assignee), now, taskID)
		return err
	}
	_, err := s.Pool.Exec(ctx, `UPDATE tasks SET assignee=$1, updated_at=$2 WHERE task_id=$3`, toNull(assignee), now, taskID)
	return err
}

func (s *Store) ClaimTask(ctx context.Context, teamName string, taskID int64, assignee string) (bool, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC().Unix()
	res, err := s.Pool.Exec(ctx, `UPDATE tasks SET status='in_progress', assignee=$1, updated_at=$2, dri=COALESCE(dri, $3) WHERE task_id=$4 AND team_id=$5 AND status='todo'`,
		assignee, now, assignee, taskID, team.TeamID)
	if err != nil {
		return false, err
	}
	return res.RowsAffected() > 0, nil
}

func (s *Store) SetTaskFailed(ctx context.Context, taskID int64) error {
	now := time.Now().UTC().Unix()
	_, err := s.Pool.Exec(ctx, `UPDATE tasks SET status='failed', updated_at=$1, attempt_count=COALESCE(attempt_count,0)+1 WHERE task_id=$2`, now, taskID)
	return err
}

func (s *Store) RequeueTask(ctx context.Context, teamName string, taskID int64) error {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	_, err = s.Pool.Exec(ctx, `UPDATE tasks SET status='todo', assignee=NULL, updated_at=$1 WHERE task_id=$2 AND team_id=$3`, now, taskID, team.TeamID)
	return err
}

func (s *Store) SetTaskCancelled(ctx context.Context, teamName string, taskID int64) error {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	_, err = s.Pool.Exec(ctx, `UPDATE tasks SET status='cancelled', assignee=NULL, updated_at=$1 WHERE task_id=$2 AND team_id=$3`, now, taskID, team.TeamID)
	return err
}

func (s *Store) ClearTaskGitFields(ctx context.Context, taskID int64) error {
	now := time.Now().UTC().Unix()
	_, err := s.Pool.Exec(ctx, `UPDATE tasks SET worktree_path=NULL, branch_name=NULL, base_sha=NULL, repo_name=NULL, updated_at=$1 WHERE task_id=$2`, now, taskID)
	return err
}

func (s *Store) UpdateTaskGitFields(ctx context.Context, taskID int64, worktreePath, branchName, baseSHA, repoName *string) error {
	now := time.Now().UTC().Unix()
	_, err := s.Pool.Exec(ctx, `UPDATE tasks SET worktree_path=$1, branch_name=$2, base_sha=$3, repo_name=$4, updated_at=$5 WHERE task_id=$6`,
		toNull(worktreePath), toNull(branchName), toNull(baseSHA), toNull(repoName), now, taskID)
	return err
}

func (s *Store) RewindTask(ctx context.Context, teamName string, taskID int64) error {
	task, err := s.GetTaskByIDAndTeam(ctx, teamName, taskID)
	if err != nil || task == nil {
		if err != nil {
			return err
		}
		return errors.New("task not found")
	}
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	if task.WorkflowID != nil && *task.WorkflowID != "" {
		initial, err := s.GetWorkflowInitialStage(ctx, *task.WorkflowID)
		if err != nil {
			return err
		}
		_, err = s.Pool.Exec(ctx, `UPDATE tasks SET status='todo', assignee=NULL, current_stage=$1, updated_at=$2 WHERE task_id=$3 AND team_id=$4`, initial, now, taskID, team.TeamID)
		return err
	}
	_, err = s.Pool.Exec(ctx, `UPDATE tasks SET status='todo', assignee=NULL, current_stage=NULL, updated_at=$1 WHERE task_id=$2 AND team_id=$3`, now, taskID, team.TeamID)
	return err
}

func (s *Store) CreateTaskComment(ctx context.Context, teamName string, taskID int64, author, body string) (int64, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.Pool.QueryRow(ctx, `INSERT INTO task_comments(task_id, team_id, author, body, created_at) VALUES($1, $2, $3, $4, $5) RETURNING comment_id`,
		taskID, team.TeamID, author, body, time.Now().UTC().Unix()).Scan(&id)
	return id, err
}

func (s *Store) ListTaskComments(ctx context.Context, teamName string, taskID int64) ([]store.TaskComment, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT comment_id, task_id, team_id, author, body, created_at FROM task_comments WHERE task_id=$1 AND team_id=$2 ORDER BY created_at DESC`, taskID, team.TeamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.TaskComment
	for rows.Next() {
		var c store.TaskComment
		var createdAt int64
		if err := rows.Scan(&c.CommentID, &c.TaskID, &c.TeamID, &c.Author, &c.Body, &createdAt); err != nil {
			return nil, err
		}
		c.CreatedAt = time.Unix(createdAt, 0).UTC()
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) AddTaskAttachment(ctx context.Context, teamName string, taskID int64, filePath string) error {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	_, err = s.Pool.Exec(ctx, `INSERT INTO task_attachments(task_id, team_id, file_path, created_at) VALUES($1, $2, $3, $4) ON CONFLICT DO NOTHING`, taskID, team.TeamID, filePath, now)
	return err
}

func (s *Store) RemoveTaskAttachment(ctx context.Context, teamName string, taskID int64, filePath string) error {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `DELETE FROM task_attachments WHERE task_id=$1 AND team_id=$2 AND file_path=$3`, taskID, team.TeamID, filePath)
	return err
}

func (s *Store) ListTaskAttachments(ctx context.Context, teamName string, taskID int64) ([]store.TaskAttachment, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT attachment_id, task_id, file_path, created_at FROM task_attachments WHERE task_id=$1 AND team_id=$2 ORDER BY created_at ASC`, taskID, team.TeamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.TaskAttachment
	for rows.Next() {
		var a store.TaskAttachment
		var createdAt int64
		if err := rows.Scan(&a.AttachmentID, &a.TaskID, &a.FilePath, &createdAt); err != nil {
			return nil, err
		}
		a.CreatedAt = time.Unix(createdAt, 0).UTC()
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) AddTaskDependency(ctx context.Context, teamName string, taskID, dependsOnTaskID int64) error {
	if _, err := s.GetTaskByIDAndTeam(ctx, teamName, taskID); err != nil {
		return err
	}
	target, err := s.GetTaskByIDAndTeam(ctx, teamName, dependsOnTaskID)
	if err != nil {
		return err
	}
	if target == nil {
		return fmt.Errorf("dependency task %d not found in team", dependsOnTaskID)
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO task_dependencies(task_id, depends_on_task_id) VALUES($1, $2) ON CONFLICT DO NOTHING`, taskID, dependsOnTaskID)
	return err
}

func (s *Store) ListTaskDependencies(ctx context.Context, teamName string, taskID int64) ([]int64, error) {
	rows, err := s.Pool.Query(ctx, `SELECT depends_on_task_id FROM task_dependencies WHERE task_id=$1`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var dep int64
		if err := rows.Scan(&dep); err != nil {
			return nil, err
		}
		out = append(out, dep)
	}
	return out, rows.Err()
}

func (s *Store) NextRunnableTaskForTeam(ctx context.Context, teamName string) (*store.Task, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	row := s.Pool.QueryRow(ctx, `
SELECT task_id, title, status, assignee, dri, COALESCE(attempt_count,0), workflow_id, current_stage, worktree_path, branch_name, base_sha, repo_name, created_at, updated_at
FROM tasks WHERE team_id = $1 AND status IN ('todo','in_progress') AND (current_stage IS NULL OR current_stage != 'Merging') ORDER BY updated_at ASC LIMIT 1`, team.TeamID)
	task, err := scanTaskRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return task, nil
}

func (s *Store) GetTaskByIDAndTeam(ctx context.Context, teamName string, taskID int64) (*store.Task, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	row := s.Pool.QueryRow(ctx, `
SELECT task_id, title, status, assignee, dri, COALESCE(attempt_count,0), workflow_id, current_stage, worktree_path, branch_name, base_sha, repo_name, created_at, updated_at
FROM tasks WHERE task_id = $1 AND team_id = $2`, taskID, team.TeamID)
	task, err := scanTaskRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return task, nil
}

func (s *Store) UpdateTaskStage(ctx context.Context, taskID int64, stage string) error {
	now := time.Now().UTC().Unix()
	_, err := s.Pool.Exec(ctx, `UPDATE tasks SET current_stage=$1, updated_at=$2 WHERE task_id=$3`, stage, now, taskID)
	return err
}

func (s *Store) SetTaskWorkflowAndStage(ctx context.Context, taskID int64, workflowID, stage string) error {
	now := time.Now().UTC().Unix()
	_, err := s.Pool.Exec(ctx, `UPDATE tasks SET workflow_id=$1, current_stage=$2, updated_at=$3 WHERE task_id=$4`, workflowID, stage, now, taskID)
	return err
}

func (s *Store) CreateTaskReview(ctx context.Context, teamName string, taskID int64, reviewerAgent, outcome, comments string) (int64, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return 0, err
	}
	var reviewID int64
	err = s.Pool.QueryRow(ctx, `INSERT INTO task_reviews(task_id, team_id, reviewer_agent, outcome, comments, created_at) VALUES($1, $2, $3, $4, $5, $6) RETURNING review_id`,
		taskID, team.TeamID, reviewerAgent, outcome, comments, time.Now().UTC().Unix()).Scan(&reviewID)
	return reviewID, err
}

func (s *Store) ListTaskReviews(ctx context.Context, teamName string, taskID int64) ([]store.TaskReview, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT review_id, task_id, team_id, reviewer_agent, outcome, comments, created_at FROM task_reviews WHERE task_id=$1 AND team_id=$2 ORDER BY created_at DESC`, taskID, team.TeamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.TaskReview
	for rows.Next() {
		var r store.TaskReview
		var createdAt int64
		if err := rows.Scan(&r.ReviewID, &r.TaskID, &r.TeamID, &r.ReviewerAgent, &r.Outcome, &r.Comments, &createdAt); err != nil {
			return nil, err
		}
		r.CreatedAt = time.Unix(createdAt, 0).UTC()
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) ListRepos(ctx context.Context, teamName string) ([]store.Repo, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT name, source, approval, test_cmd, created_at FROM repos WHERE team_id = $1 ORDER BY created_at ASC`, team.TeamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Repo
	for rows.Next() {
		var name, source, approval string
		var testCmd *string
		var createdAt int64
		if err := rows.Scan(&name, &source, &approval, &testCmd, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, store.Repo{Name: name, Source: source, Approval: approval, TestCmd: testCmd, CreatedAt: time.Unix(createdAt, 0).UTC()})
	}
	return out, rows.Err()
}

func (s *Store) CreateRepo(ctx context.Context, teamName, name, source, approval string, testCmd *string) error {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("repo name required")
	}
	if source == "" {
		return errors.New("repo source required")
	}
	if approval == "" {
		approval = "manual"
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO repos(repo_id, team_id, name, source, approval, test_cmd, created_at) VALUES($1, $2, $3, $4, $5, $6, $7)`,
		randomID(), team.TeamID, name, source, approval, testCmd, time.Now().UTC().Unix())
	return err
}

func (s *Store) SetRepoApproval(ctx context.Context, teamName, repoName, approval string) error {
	if approval != "auto" && approval != "manual" {
		return errors.New("approval must be auto or manual")
	}
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return err
	}
	res, err := s.Pool.Exec(ctx, `UPDATE repos SET approval=$1 WHERE team_id=$2 AND name=$3`, approval, team.TeamID, repoName)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("repo not found")
	}
	return nil
}

func (s *Store) ListWorkflows(ctx context.Context, teamName string) ([]store.Workflow, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT name, version, source_path, created_at FROM workflows WHERE team_id = $1 ORDER BY name ASC, version DESC`, team.TeamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Workflow
	for rows.Next() {
		var w store.Workflow
		var createdAt int64
		if err := rows.Scan(&w.Name, &w.Version, &w.SourcePath, &createdAt); err != nil {
			return nil, err
		}
		w.CreatedAt = time.Unix(createdAt, 0).UTC()
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) CreateWorkflow(ctx context.Context, teamName, name string, version int, sourcePath string) (string, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return "", err
	}
	if name == "" {
		return "", errors.New("workflow name required")
	}
	if version <= 0 {
		return "", errors.New("workflow version must be > 0")
	}
	if sourcePath == "" {
		return "", errors.New("workflow source path required")
	}
	wfID := randomID()
	_, err = s.Pool.Exec(ctx, `INSERT INTO workflows(workflow_id, team_id, name, version, source_path, created_at) VALUES($1, $2, $3, $4, $5, $6)`,
		wfID, team.TeamID, name, version, sourcePath, time.Now().UTC().Unix())
	if err != nil {
		return "", err
	}
	if name == "default" && version == 1 {
		_ = s.seedDefaultWorkflowStages(ctx, wfID)
	}
	return wfID, nil
}

func (s *Store) CreateWorkflowWithStages(ctx context.Context, teamName, name string, version int, sourcePath string, stages []store.WorkflowStage, transitions []store.WorkflowTransition) (string, error) {
	wfID, err := s.CreateWorkflow(ctx, teamName, name, version, sourcePath)
	if err != nil {
		return "", err
	}
	_, _ = s.Pool.Exec(ctx, `DELETE FROM workflow_stages WHERE workflow_id = $1`, wfID)
	_, _ = s.Pool.Exec(ctx, `DELETE FROM workflow_transitions WHERE workflow_id = $1`, wfID)
	for _, st := range stages {
		_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_stages(workflow_id, stage_name, stage_type, outcomes, candidate_agents) VALUES($1, $2, $3, $4, $5)`,
			wfID, st.StageName, st.StageType, st.Outcomes, st.CandidateAgents)
	}
	for _, tr := range transitions {
		_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES($1, $2, $3, $4)`,
			wfID, tr.FromStage, tr.Outcome, tr.ToStage)
	}
	return wfID, nil
}

func (s *Store) seedDefaultWorkflowStages(ctx context.Context, workflowID string) error {
	// Enhanced default: Coding -> InReview -> InApproval -> Merging -> Done
	_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_stages(workflow_id, stage_name, stage_type, outcomes) VALUES($1, 'Coding', 'agent', 'submit_for_review,done') ON CONFLICT DO NOTHING`, workflowID)
	_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_stages(workflow_id, stage_name, stage_type, outcomes) VALUES($1, 'InReview', 'agent', 'approved,changes_requested') ON CONFLICT DO NOTHING`, workflowID)
	_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_stages(workflow_id, stage_name, stage_type, outcomes) VALUES($1, 'InApproval', 'human', 'approved,changes_requested') ON CONFLICT DO NOTHING`, workflowID)
	_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_stages(workflow_id, stage_name, stage_type, outcomes) VALUES($1, 'Merging', 'merge', 'done') ON CONFLICT DO NOTHING`, workflowID)
	_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_stages(workflow_id, stage_name, stage_type, outcomes) VALUES($1, 'Done', 'terminal', '') ON CONFLICT DO NOTHING`, workflowID)
	_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES($1, 'Coding', 'submit_for_review', 'InReview') ON CONFLICT DO NOTHING`, workflowID)
	_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES($1, 'Coding', 'done', 'Done') ON CONFLICT DO NOTHING`, workflowID)
	_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES($1, 'InReview', 'approved', 'InApproval') ON CONFLICT DO NOTHING`, workflowID)
	_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES($1, 'InReview', 'changes_requested', 'Coding') ON CONFLICT DO NOTHING`, workflowID)
	_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES($1, 'InApproval', 'approved', 'Merging') ON CONFLICT DO NOTHING`, workflowID)
	_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES($1, 'InApproval', 'changes_requested', 'Coding') ON CONFLICT DO NOTHING`, workflowID)
	_, _ = s.Pool.Exec(ctx, `INSERT INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES($1, 'Merging', 'done', 'Done') ON CONFLICT DO NOTHING`, workflowID)
	return nil
}

func (s *Store) GetWorkflowStages(ctx context.Context, workflowID string) ([]store.WorkflowStage, error) {
	rows, err := s.Pool.Query(ctx, `SELECT workflow_id, stage_name, stage_type, outcomes, COALESCE(candidate_agents,'') FROM workflow_stages WHERE workflow_id = $1 ORDER BY stage_name`, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.WorkflowStage
	for rows.Next() {
		var w store.WorkflowStage
		if err := rows.Scan(&w.WorkflowID, &w.StageName, &w.StageType, &w.Outcomes, &w.CandidateAgents); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) GetWorkflowTransitions(ctx context.Context, workflowID string) ([]store.WorkflowTransition, error) {
	rows, err := s.Pool.Query(ctx, `SELECT workflow_id, from_stage, outcome, to_stage FROM workflow_transitions WHERE workflow_id = $1`, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.WorkflowTransition
	for rows.Next() {
		var t store.WorkflowTransition
		if err := rows.Scan(&t.WorkflowID, &t.FromStage, &t.Outcome, &t.ToStage); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetWorkflowIDByTeamAndName(ctx context.Context, teamName, name string, version int) (string, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return "", err
	}
	var wfID string
	err = s.Pool.QueryRow(ctx, `SELECT workflow_id FROM workflows WHERE team_id = $1 AND name = $2 AND version = $3 LIMIT 1`, team.TeamID, name, version).Scan(&wfID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return wfID, nil
}

func (s *Store) GetWorkflowInitialStage(ctx context.Context, workflowID string) (string, error) {
	toStages, err := s.Pool.Query(ctx, `SELECT DISTINCT to_stage FROM workflow_transitions WHERE workflow_id = $1`, workflowID)
	if err != nil {
		return "", err
	}
	defer toStages.Close()
	toSet := make(map[string]struct{})
	for toStages.Next() {
		var to string
		if err := toStages.Scan(&to); err != nil {
			return "", err
		}
		toSet[to] = struct{}{}
	}
	if err := toStages.Err(); err != nil {
		return "", err
	}
	rows, err := s.Pool.Query(ctx, `SELECT stage_name FROM workflow_stages WHERE workflow_id = $1 ORDER BY stage_name`, workflowID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", err
		}
		if _, ok := toSet[name]; !ok {
			return name, nil
		}
	}
	return "", errors.New("workflow has no initial stage")
}

func (s *Store) ListAllowedDomains(ctx context.Context) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT domain FROM network_allowlist ORDER BY domain ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) ResetAllowlist(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM network_allowlist`)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO network_allowlist(domain) VALUES('*') ON CONFLICT (domain) DO NOTHING`)
	return err
}

func (s *Store) AllowDomain(ctx context.Context, domain string) error {
	if domain == "" {
		return errors.New("domain required")
	}
	if domain == "*" {
		return s.ResetAllowlist(ctx)
	}
	_, _ = s.Pool.Exec(ctx, `DELETE FROM network_allowlist WHERE domain = '*'`)
	_, err := s.Pool.Exec(ctx, `INSERT INTO network_allowlist(domain) VALUES($1) ON CONFLICT (domain) DO NOTHING`, domain)
	return err
}

func (s *Store) DisallowDomain(ctx context.Context, domain string) error {
	if domain == "" {
		return errors.New("domain required")
	}
	_, err := s.Pool.Exec(ctx, `DELETE FROM network_allowlist WHERE domain = $1`, domain)
	if err != nil {
		return err
	}
	domains, err := s.ListAllowedDomains(ctx)
	if err != nil {
		return nil
	}
	if len(domains) == 0 {
		return s.ResetAllowlist(ctx)
	}
	return nil
}

func (s *Store) CreateMessage(ctx context.Context, teamName, sender, recipient, content string) (int64, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.Pool.QueryRow(ctx, `INSERT INTO messages(team_id, sender, recipient, content, created_at) VALUES($1, $2, $3, $4, $5) RETURNING message_id`,
		team.TeamID, sender, recipient, content, time.Now().UTC().Unix()).Scan(&id)
	return id, err
}

func (s *Store) ListMessages(ctx context.Context, teamName string, recipient string, limit int) ([]store.Message, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	var q string
	var args []any
	if recipient != "" && limit > 0 {
		q = `SELECT message_id, team_id, sender, recipient, content, created_at, processed_at FROM messages WHERE team_id = $1 AND recipient = $2 ORDER BY created_at DESC LIMIT $3`
		args = []any{team.TeamID, recipient, limit}
	} else if recipient != "" {
		q = `SELECT message_id, team_id, sender, recipient, content, created_at, processed_at FROM messages WHERE team_id = $1 AND recipient = $2 ORDER BY created_at DESC`
		args = []any{team.TeamID, recipient}
	} else if limit > 0 {
		q = `SELECT message_id, team_id, sender, recipient, content, created_at, processed_at FROM messages WHERE team_id = $1 ORDER BY created_at DESC LIMIT $2`
		args = []any{team.TeamID, limit}
	} else {
		q = `SELECT message_id, team_id, sender, recipient, content, created_at, processed_at FROM messages WHERE team_id = $1 ORDER BY created_at DESC`
		args = []any{team.TeamID}
	}
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Message
	for rows.Next() {
		var m store.Message
		var createdAt int64
		var processedAt *int64
		if err := rows.Scan(&m.MessageID, &m.TeamID, &m.Sender, &m.Recipient, &m.Content, &createdAt, &processedAt); err != nil {
			return nil, err
		}
		m.CreatedAt = time.Unix(createdAt, 0).UTC()
		if processedAt != nil {
			t := time.Unix(*processedAt, 0).UTC()
			m.ProcessedAt = &t
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) ListUnprocessedMessages(ctx context.Context, teamName string, recipient string, limit int) ([]store.Message, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	if recipient == "" {
		return nil, nil
	}
	q := `SELECT message_id, team_id, sender, recipient, content, created_at, processed_at FROM messages WHERE team_id = $1 AND recipient = $2 AND processed_at IS NULL ORDER BY created_at ASC`
	args := []any{team.TeamID, recipient}
	if limit > 0 {
		q += ` LIMIT $3`
		args = append(args, limit)
	}
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Message
	for rows.Next() {
		var m store.Message
		var createdAt int64
		var processedAt *int64
		if err := rows.Scan(&m.MessageID, &m.TeamID, &m.Sender, &m.Recipient, &m.Content, &createdAt, &processedAt); err != nil {
			return nil, err
		}
		m.CreatedAt = time.Unix(createdAt, 0).UTC()
		if processedAt != nil {
			t := time.Unix(*processedAt, 0).UTC()
			m.ProcessedAt = &t
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) MarkMessageProcessed(ctx context.Context, messageID int64) error {
	now := time.Now().UTC().Unix()
	_, err := s.Pool.Exec(ctx, `UPDATE messages SET processed_at=$1 WHERE message_id=$2`, now, messageID)
	return err
}
