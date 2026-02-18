package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

func (s *sqliteStore) ListTeams(ctx context.Context) ([]Team, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT
  t.name, t.team_id, t.created_at,
  (SELECT COUNT(*) FROM agents a WHERE a.team_id = t.team_id) AS agent_count,
  (SELECT COUNT(*) FROM tasks k WHERE k.team_id = t.team_id) AS task_count
FROM teams t
ORDER BY t.created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []Team
	for rows.Next() {
		var (
			name      string
			teamID    string
			createdAt int64
			agentCnt  int
			taskCnt   int
		)
		if err := rows.Scan(&name, &teamID, &createdAt, &agentCnt, &taskCnt); err != nil {
			return nil, err
		}
		out = append(out, Team{
			Name:       name,
			TeamID:     teamID,
			CreatedAt:  time.Unix(createdAt, 0).UTC(),
			AgentCount: agentCnt,
			TaskCount:  taskCnt,
		})
	}
	return out, rows.Err()
}

func (s *sqliteStore) GetTeamByName(ctx context.Context, name string) (Team, error) {
	var t Team
	var createdAt int64
	err := s.stmtGetTeamByName.QueryRowContext(ctx, name).Scan(&t.Name, &t.TeamID, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Team{}, fmt.Errorf("team not found: %s", name)
		}
		return Team{}, err
	}
	t.CreatedAt = time.Unix(createdAt, 0).UTC()
	return t, nil
}

func (s *sqliteStore) CreateTeam(ctx context.Context, name string) (Team, error) {
	if name == "" {
		return Team{}, errors.New("team name required")
	}
	id := randomID()
	now := time.Now().UTC().Unix()
	_, err := s.DB.ExecContext(ctx, `INSERT INTO teams(team_id, name, created_at) VALUES(?, ?, ?)`, id, name, now)
	if err != nil {
		return Team{}, err
	}
	return Team{TeamID: id, Name: name, CreatedAt: time.Unix(now, 0).UTC()}, nil
}

func (s *sqliteStore) DeleteTeam(ctx context.Context, name string) error {
	team, err := s.GetTeamByName(ctx, name)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `DELETE FROM teams WHERE team_id = ?`, team.TeamID)
	return err
}

func (s *sqliteStore) ListAgents(ctx context.Context, teamName string) ([]Agent, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT name, role, created_at FROM agents WHERE team_id = ? ORDER BY created_at ASC`, team.TeamID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []Agent
	for rows.Next() {
		var (
			name      string
			role      string
			createdAt int64
		)
		if err := rows.Scan(&name, &role, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, Agent{Name: name, Role: role, CreatedAt: time.Unix(createdAt, 0).UTC()})
	}
	return out, rows.Err()
}

func (s *sqliteStore) CreateAgent(ctx context.Context, teamName, name, role string) error {
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
	_, err = s.DB.ExecContext(ctx, `INSERT INTO agents(agent_id, team_id, name, role, created_at) VALUES(?, ?, ?, ?, ?)`,
		randomID(), team.TeamID, name, role, time.Now().UTC().Unix())
	return err
}

func (s *sqliteStore) ListTasks(ctx context.Context, teamName string, limit int) ([]Task, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	usePrepared := limit == 0 || limit >= 100
	if usePrepared {
		rows, err := s.stmtListTasks100.QueryContext(ctx, team.TeamID)
		if err != nil {
			return nil, err
		}
		defer func() { _ = rows.Close() }()
		var out []Task
		for rows.Next() {
			task, err := scanTaskRow(rows)
			if err != nil {
				return nil, err
			}
			out = append(out, *task)
		}
		return out, rows.Err()
	}
	q := `SELECT task_id, title, status, assignee, dri, COALESCE(attempt_count,0), workflow_id, current_stage, worktree_path, branch_name, base_sha, repo_name, created_at, updated_at FROM tasks WHERE team_id = ? ORDER BY created_at DESC LIMIT ?`
	rows, err := s.DB.QueryContext(ctx, q, team.TeamID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Task
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *task)
	}
	return out, rows.Err()
}

func (s *sqliteStore) ListTasksInStage(ctx context.Context, teamName, stage string, limit int) ([]Task, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	q := `SELECT task_id, title, status, assignee, dri, COALESCE(attempt_count,0), workflow_id, current_stage, worktree_path, branch_name, base_sha, repo_name, created_at, updated_at FROM tasks WHERE team_id = ? AND current_stage = ? ORDER BY updated_at ASC`
	args := []any{team.TeamID, stage}
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Task
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *task)
	}
	return out, rows.Err()
}

// scanTaskRow scans the current row of rows (must have task columns in order: task_id, title, status, assignee, dri, attempt_count, workflow_id, current_stage, worktree_path, branch_name, base_sha, repo_name, created_at, updated_at).
func scanTaskRow(rows interface{ Scan(dest ...any) error }) (*Task, error) {
	var (
		id           int64
		title        string
		status       string
		assignee     sql.NullString
		dri          sql.NullString
		attemptCount int
		workflowID   sql.NullString
		currentStage sql.NullString
		worktreePath sql.NullString
		branchName   sql.NullString
		baseSHA      sql.NullString
		repoName     sql.NullString
		createdAt    int64
		updatedAt    int64
	)
	err := rows.Scan(&id, &title, &status, &assignee, &dri, &attemptCount, &workflowID, &currentStage, &worktreePath, &branchName, &baseSHA, &repoName, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	var a, d, wfID, curStage, wtPath, brName, bSHA, rName *string
	if assignee.Valid {
		a = &assignee.String
	}
	if dri.Valid {
		d = &dri.String
	}
	if workflowID.Valid {
		wfID = &workflowID.String
	}
	if currentStage.Valid {
		curStage = &currentStage.String
	}
	if worktreePath.Valid {
		wtPath = &worktreePath.String
	}
	if branchName.Valid {
		brName = &branchName.String
	}
	if baseSHA.Valid {
		bSHA = &baseSHA.String
	}
	if repoName.Valid {
		rName = &repoName.String
	}
	return &Task{
		TaskID:       id,
		Title:        title,
		Status:       status,
		Assignee:     a,
		DRI:          d,
		AttemptCount: attemptCount,
		WorkflowID:   wfID,
		CurrentStage: curStage,
		WorktreePath: wtPath,
		BranchName:   brName,
		BaseSHA:      bSHA,
		RepoName:     rName,
		CreatedAt:    time.Unix(createdAt, 0).UTC(),
		UpdatedAt:    time.Unix(updatedAt, 0).UTC(),
	}, nil
}

func (s *sqliteStore) CreateTask(ctx context.Context, teamName, title, status string, workflowID *string) (int64, error) {
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
	res, err := s.stmtCreateTask.ExecContext(ctx, team.TeamID, title, status, now, now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if workflowID != nil && *workflowID != "" {
		initial, err := s.GetWorkflowInitialStage(ctx, *workflowID)
		if err == nil {
			_, _ = s.DB.ExecContext(ctx, `UPDATE tasks SET workflow_id=?, current_stage=?, updated_at=? WHERE task_id=?`, *workflowID, initial, now, id)
		}
	}
	return id, nil
}

// UpdateTask updates status and/or assignee for a task by ID. Pass nil assignee to clear. Empty status leaves status unchanged.
func (s *sqliteStore) UpdateTask(ctx context.Context, taskID int64, status string, assignee *string) error {
	now := time.Now().UTC().Unix()
	var assigneeVal interface{}
	if assignee != nil {
		assigneeVal = *assignee
	}
	if status != "" {
		_, err := s.stmtUpdateTaskStatus.ExecContext(ctx, status, assigneeVal, now, taskID)
		return err
	}
	_, err := s.stmtUpdateTaskAssign.ExecContext(ctx, assigneeVal, now, taskID)
	return err
}

// ClaimTask sets status to in_progress and assignee if the task is still todo (optimistic lock). Returns true if claimed.
func (s *sqliteStore) ClaimTask(ctx context.Context, teamName string, taskID int64, assignee string) (bool, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC().Unix()
	res, err := s.stmtClaimTask.ExecContext(ctx, assignee, now, assignee, taskID, team.TeamID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// SetTaskFailed sets status to failed and increments attempt_count.
func (s *sqliteStore) SetTaskFailed(ctx context.Context, taskID int64) error {
	now := time.Now().UTC().Unix()
	_, err := s.DB.ExecContext(ctx, `UPDATE tasks SET status='failed', updated_at=?, attempt_count=COALESCE(attempt_count,0)+1 WHERE task_id=?`, now, taskID)
	return err
}

// RequeueTask sets status to todo and clears assignee.
func (s *sqliteStore) RequeueTask(ctx context.Context, teamName string, taskID int64) error {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	_, err = s.DB.ExecContext(ctx, `UPDATE tasks SET status='todo', assignee=NULL, updated_at=? WHERE task_id=? AND team_id=?`, now, taskID, team.TeamID)
	return err
}

// SetTaskCancelled sets status to cancelled (terminal). Does not clean up worktree; caller should clear git fields and delete worktree if needed.
func (s *sqliteStore) SetTaskCancelled(ctx context.Context, teamName string, taskID int64) error {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	_, err = s.DB.ExecContext(ctx, `UPDATE tasks SET status='cancelled', assignee=NULL, updated_at=? WHERE task_id=? AND team_id=?`, now, taskID, team.TeamID)
	return err
}

// ClearTaskGitFields clears worktree_path, branch_name, base_sha, repo_name for a task.
func (s *sqliteStore) ClearTaskGitFields(ctx context.Context, taskID int64) error {
	now := time.Now().UTC().Unix()
	_, err := s.DB.ExecContext(ctx, `UPDATE tasks SET worktree_path=NULL, branch_name=NULL, base_sha=NULL, repo_name=NULL, updated_at=? WHERE task_id=?`, now, taskID)
	return err
}

// UpdateTaskGitFields sets git-related fields for a task.
func (s *sqliteStore) UpdateTaskGitFields(ctx context.Context, taskID int64, worktreePath, branchName, baseSHA, repoName *string) error {
	now := time.Now().UTC().Unix()
	_, err := s.DB.ExecContext(ctx, `UPDATE tasks SET worktree_path=?, branch_name=?, base_sha=?, repo_name=?, updated_at=? WHERE task_id=?`,
		toNull(worktreePath), toNull(branchName), toNull(baseSHA), toNull(repoName), now, taskID)
	return err
}

func toNull(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

// RewindTask sets status to todo, clears assignee, and resets current_stage to workflow initial (or null if no workflow).
func (s *sqliteStore) RewindTask(ctx context.Context, teamName string, taskID int64) error {
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
		_, err = s.DB.ExecContext(ctx, `UPDATE tasks SET status='todo', assignee=NULL, current_stage=?, updated_at=? WHERE task_id=? AND team_id=?`, initial, now, taskID, team.TeamID)
		return err
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE tasks SET status='todo', assignee=NULL, current_stage=NULL, updated_at=? WHERE task_id=? AND team_id=?`, now, taskID, team.TeamID)
	return err
}

// CreateTaskComment adds a comment to a task.
func (s *sqliteStore) CreateTaskComment(ctx context.Context, teamName string, taskID int64, author, body string) (int64, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC().Unix()
	res, err := s.DB.ExecContext(ctx, `INSERT INTO task_comments(task_id, team_id, author, body, created_at) VALUES(?, ?, ?, ?, ?)`, taskID, team.TeamID, author, body, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListTaskComments returns comments for a task (newest first).
func (s *sqliteStore) ListTaskComments(ctx context.Context, teamName string, taskID int64) ([]TaskComment, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT comment_id, task_id, team_id, author, body, created_at FROM task_comments WHERE task_id=? AND team_id=? ORDER BY created_at DESC`, taskID, team.TeamID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []TaskComment
	for rows.Next() {
		var c TaskComment
		var createdAt int64
		if err := rows.Scan(&c.CommentID, &c.TaskID, &c.TeamID, &c.Author, &c.Body, &createdAt); err != nil {
			return nil, err
		}
		c.CreatedAt = time.Unix(createdAt, 0).UTC()
		out = append(out, c)
	}
	return out, rows.Err()
}

// AddTaskAttachment records an attachment path for a task.
func (s *sqliteStore) AddTaskAttachment(ctx context.Context, teamName string, taskID int64, filePath string) error {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	_, err = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO task_attachments(task_id, team_id, file_path, created_at) VALUES(?, ?, ?, ?)`, taskID, team.TeamID, filePath, now)
	return err
}

// RemoveTaskAttachment removes an attachment by path.
func (s *sqliteStore) RemoveTaskAttachment(ctx context.Context, teamName string, taskID int64, filePath string) error {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `DELETE FROM task_attachments WHERE task_id=? AND team_id=? AND file_path=?`, taskID, team.TeamID, filePath)
	return err
}

// ListTaskAttachments returns attachment paths for a task.
func (s *sqliteStore) ListTaskAttachments(ctx context.Context, teamName string, taskID int64) ([]TaskAttachment, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT attachment_id, task_id, file_path, created_at FROM task_attachments WHERE task_id=? AND team_id=? ORDER BY created_at ASC`, taskID, team.TeamID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []TaskAttachment
	for rows.Next() {
		var a TaskAttachment
		var createdAt int64
		if err := rows.Scan(&a.AttachmentID, &a.TaskID, &a.FilePath, &createdAt); err != nil {
			return nil, err
		}
		a.CreatedAt = time.Unix(createdAt, 0).UTC()
		out = append(out, a)
	}
	return out, rows.Err()
}

// AddTaskDependency records that taskID depends on dependsOnTaskID.
// Both tasks must belong to the same team.
func (s *sqliteStore) AddTaskDependency(ctx context.Context, teamName string, taskID, dependsOnTaskID int64) error {
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
	_, err = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO task_dependencies(task_id, depends_on_task_id) VALUES(?, ?)`, taskID, dependsOnTaskID)
	return err
}

// ListTaskDependencies returns task IDs that this task depends on.
func (s *sqliteStore) ListTaskDependencies(ctx context.Context, teamName string, taskID int64) ([]int64, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT depends_on_task_id FROM task_dependencies WHERE task_id=?`, taskID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
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

// NextRunnableTaskForTeam returns one task with status todo or in_progress for the team (oldest updated first), or nil if none.
func (s *sqliteStore) NextRunnableTaskForTeam(ctx context.Context, teamName string) (*Task, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	row := s.stmtNextRunnable.QueryRowContext(ctx, team.TeamID)
	task, err := scanTaskRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return task, nil
}

// GetTaskByIDAndTeam returns the task if it belongs to the given team, or nil/error if not found.
func (s *sqliteStore) GetTaskByIDAndTeam(ctx context.Context, teamName string, taskID int64) (*Task, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	row := s.stmtGetTaskByID.QueryRowContext(ctx, taskID, team.TeamID)
	task, err := scanTaskRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return task, nil
}

func (s *sqliteStore) ListRepos(ctx context.Context, teamName string) ([]Repo, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT name, source, approval, test_cmd, created_at
FROM repos
WHERE team_id = ?
ORDER BY created_at ASC`, team.TeamID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []Repo
	for rows.Next() {
		var (
			name      string
			source    string
			approval  string
			testCmd   sql.NullString
			createdAt int64
		)
		if err := rows.Scan(&name, &source, &approval, &testCmd, &createdAt); err != nil {
			return nil, err
		}
		var tc *string
		if testCmd.Valid {
			tc = &testCmd.String
		}
		out = append(out, Repo{
			Name:      name,
			Source:    source,
			Approval:  approval,
			TestCmd:   tc,
			CreatedAt: time.Unix(createdAt, 0).UTC(),
		})
	}
	return out, rows.Err()
}

func (s *sqliteStore) CreateRepo(ctx context.Context, teamName, name, source, approval string, testCmd *string) error {
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
	now := time.Now().UTC().Unix()
	_, err = s.DB.ExecContext(ctx, `
INSERT INTO repos(repo_id, team_id, name, source, approval, test_cmd, created_at)
VALUES(?, ?, ?, ?, ?, ?, ?)`,
		randomID(), team.TeamID, name, source, approval, testCmd, now)
	return err
}

// SetRepoApproval sets the approval mode (auto or manual) for a repo.
func (s *sqliteStore) SetRepoApproval(ctx context.Context, teamName, repoName, approval string) error {
	if approval != "auto" && approval != "manual" {
		return errors.New("approval must be auto or manual")
	}
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return err
	}
	res, err := s.DB.ExecContext(ctx, `UPDATE repos SET approval=? WHERE team_id=? AND name=?`, approval, team.TeamID, repoName)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("repo not found")
	}
	return nil
}

func (s *sqliteStore) ListWorkflows(ctx context.Context, teamName string) ([]Workflow, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT name, version, source_path, created_at
FROM workflows
WHERE team_id = ?
ORDER BY name ASC, version DESC`, team.TeamID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []Workflow
	for rows.Next() {
		var (
			name       string
			version    int
			sourcePath string
			createdAt  int64
		)
		if err := rows.Scan(&name, &version, &sourcePath, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, Workflow{
			Name:       name,
			Version:    version,
			SourcePath: sourcePath,
			CreatedAt:  time.Unix(createdAt, 0).UTC(),
		})
	}
	return out, rows.Err()
}

func (s *sqliteStore) CreateWorkflow(ctx context.Context, teamName, name string, version int, sourcePath string) (string, error) {
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
	_, err = s.DB.ExecContext(ctx, `
INSERT INTO workflows(workflow_id, team_id, name, version, source_path, created_at)
VALUES(?, ?, ?, ?, ?, ?)`,
		wfID, team.TeamID, name, version, sourcePath, time.Now().UTC().Unix())
	if err != nil {
		return "", err
	}
	if name == "default" && version == 1 {
		_ = s.seedDefaultWorkflowStages(ctx, wfID)
	}
	return wfID, nil
}

// CreateWorkflowWithStages creates a workflow and inserts the given stages and transitions (e.g. from Python script output).
func (s *sqliteStore) CreateWorkflowWithStages(ctx context.Context, teamName, name string, version int, sourcePath string, stages []WorkflowStage, transitions []WorkflowTransition) (string, error) {
	wfID, err := s.CreateWorkflow(ctx, teamName, name, version, sourcePath)
	if err != nil {
		return "", err
	}
	// Replace any existing stages/transitions (e.g. default seed) with the script output.
	_, _ = s.DB.ExecContext(ctx, `DELETE FROM workflow_stages WHERE workflow_id = ?`, wfID)
	_, _ = s.DB.ExecContext(ctx, `DELETE FROM workflow_transitions WHERE workflow_id = ?`, wfID)
	for _, st := range stages {
		st.WorkflowID = wfID
		_, _ = s.DB.ExecContext(ctx, `INSERT INTO workflow_stages(workflow_id, stage_name, stage_type, outcomes, candidate_agents) VALUES(?, ?, ?, ?, ?)`,
			wfID, st.StageName, st.StageType, st.Outcomes, st.CandidateAgents)
	}
	for _, tr := range transitions {
		tr.WorkflowID = wfID
		_, _ = s.DB.ExecContext(ctx, `INSERT INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES(?, ?, ?, ?)`,
			wfID, tr.FromStage, tr.Outcome, tr.ToStage)
	}
	return wfID, nil
}

func (s *sqliteStore) seedDefaultWorkflowStages(ctx context.Context, workflowID string) error {
	// Enhanced default: Coding -> InReview -> InApproval -> Merging -> Done
	_, _ = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO workflow_stages(workflow_id, stage_name, stage_type, outcomes) VALUES(?, 'Coding', 'agent', 'submit_for_review,done')`, workflowID)
	_, _ = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO workflow_stages(workflow_id, stage_name, stage_type, outcomes) VALUES(?, 'InReview', 'agent', 'approved,changes_requested')`, workflowID)
	_, _ = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO workflow_stages(workflow_id, stage_name, stage_type, outcomes) VALUES(?, 'InApproval', 'human', 'approved,changes_requested')`, workflowID)
	_, _ = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO workflow_stages(workflow_id, stage_name, stage_type, outcomes) VALUES(?, 'Merging', 'merge', 'done')`, workflowID)
	_, _ = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO workflow_stages(workflow_id, stage_name, stage_type, outcomes) VALUES(?, 'Done', 'terminal', '')`, workflowID)
	_, _ = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES(?, 'Coding', 'submit_for_review', 'InReview')`, workflowID)
	_, _ = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES(?, 'Coding', 'done', 'Done')`, workflowID)
	_, _ = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES(?, 'InReview', 'approved', 'InApproval')`, workflowID)
	_, _ = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES(?, 'InReview', 'changes_requested', 'Coding')`, workflowID)
	_, _ = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES(?, 'InApproval', 'approved', 'Merging')`, workflowID)
	_, _ = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES(?, 'InApproval', 'changes_requested', 'Coding')`, workflowID)
	_, _ = s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO workflow_transitions(workflow_id, from_stage, outcome, to_stage) VALUES(?, 'Merging', 'done', 'Done')`, workflowID)
	return nil
}

func (s *sqliteStore) GetWorkflowStages(ctx context.Context, workflowID string) ([]WorkflowStage, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT workflow_id, stage_name, stage_type, outcomes, COALESCE(candidate_agents,'') FROM workflow_stages WHERE workflow_id = ? ORDER BY stage_name`, workflowID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []WorkflowStage
	for rows.Next() {
		var w WorkflowStage
		if err := rows.Scan(&w.WorkflowID, &w.StageName, &w.StageType, &w.Outcomes, &w.CandidateAgents); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *sqliteStore) GetWorkflowTransitions(ctx context.Context, workflowID string) ([]WorkflowTransition, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT workflow_id, from_stage, outcome, to_stage FROM workflow_transitions WHERE workflow_id = ?`, workflowID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []WorkflowTransition
	for rows.Next() {
		var t WorkflowTransition
		if err := rows.Scan(&t.WorkflowID, &t.FromStage, &t.Outcome, &t.ToStage); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *sqliteStore) GetWorkflowIDByTeamAndName(ctx context.Context, teamName, name string, version int) (string, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return "", err
	}
	var wfID string
	err = s.DB.QueryRowContext(ctx, `SELECT workflow_id FROM workflows WHERE team_id = ? AND name = ? AND version = ? LIMIT 1`, team.TeamID, name, version).Scan(&wfID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return wfID, nil
}

func (s *sqliteStore) GetWorkflowInitialStage(ctx context.Context, workflowID string) (string, error) {
	toStages, err := s.DB.QueryContext(ctx, `SELECT DISTINCT to_stage FROM workflow_transitions WHERE workflow_id = ?`, workflowID)
	if err != nil {
		return "", err
	}
	defer func() { _ = toStages.Close() }()
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
	rows, err := s.DB.QueryContext(ctx, `SELECT stage_name FROM workflow_stages WHERE workflow_id = ? ORDER BY stage_name`, workflowID)
	if err != nil {
		return "", err
	}
	defer func() { _ = rows.Close() }()
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

func (s *sqliteStore) UpdateTaskStage(ctx context.Context, taskID int64, stage string) error {
	now := time.Now().UTC().Unix()
	_, err := s.DB.ExecContext(ctx, `UPDATE tasks SET current_stage=?, updated_at=? WHERE task_id=?`, stage, now, taskID)
	return err
}

func (s *sqliteStore) SetTaskWorkflowAndStage(ctx context.Context, taskID int64, workflowID, stage string) error {
	now := time.Now().UTC().Unix()
	_, err := s.DB.ExecContext(ctx, `UPDATE tasks SET workflow_id=?, current_stage=?, updated_at=? WHERE task_id=?`, workflowID, stage, now, taskID)
	return err
}

func (s *sqliteStore) CreateTaskReview(ctx context.Context, teamName string, taskID int64, reviewerAgent, outcome, comments string) (int64, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC().Unix()
	res, err := s.DB.ExecContext(ctx, `INSERT INTO task_reviews(task_id, team_id, reviewer_agent, outcome, comments, created_at) VALUES(?, ?, ?, ?, ?, ?)`,
		taskID, team.TeamID, reviewerAgent, outcome, comments, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *sqliteStore) ListTaskReviews(ctx context.Context, teamName string, taskID int64) ([]TaskReview, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT review_id, task_id, team_id, reviewer_agent, outcome, comments, created_at FROM task_reviews WHERE task_id=? AND team_id=? ORDER BY created_at DESC`, taskID, team.TeamID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []TaskReview
	for rows.Next() {
		var r TaskReview
		var createdAt int64
		if err := rows.Scan(&r.ReviewID, &r.TaskID, &r.TeamID, &r.ReviewerAgent, &r.Outcome, &r.Comments, &createdAt); err != nil {
			return nil, err
		}
		r.CreatedAt = time.Unix(createdAt, 0).UTC()
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *sqliteStore) ListAllowedDomains(ctx context.Context) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT domain FROM network_allowlist ORDER BY domain ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
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

func (s *sqliteStore) ResetAllowlist(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, `DELETE FROM network_allowlist`); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO network_allowlist(domain) VALUES('*')`)
	return err
}

func (s *sqliteStore) AllowDomain(ctx context.Context, domain string) error {
	if domain == "" {
		return errors.New("domain required")
	}
	if domain == "*" {
		return s.ResetAllowlist(ctx)
	}
	// If currently unrestricted, remove wildcard.
	_, _ = s.DB.ExecContext(ctx, `DELETE FROM network_allowlist WHERE domain = '*'`)
	_, err := s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO network_allowlist(domain) VALUES(?)`, domain)
	return err
}

func (s *sqliteStore) DisallowDomain(ctx context.Context, domain string) error {
	if domain == "" {
		return errors.New("domain required")
	}
	_, err := s.DB.ExecContext(ctx, `DELETE FROM network_allowlist WHERE domain = ?`, domain)
	if err != nil {
		return err
	}
	// Never allow empty set; default to unrestricted.
	domains, err := s.ListAllowedDomains(ctx)
	if err != nil {
		return nil
	}
	if len(domains) == 0 {
		return s.ResetAllowlist(ctx)
	}
	return nil
}

// SeedDemo ensures there's at least one team with a few agents and a task
// so the fresh UI isn't empty.
func (s *sqliteStore) SeedDemo(ctx context.Context) error {
	teams, err := s.ListTeams(ctx)
	if err != nil {
		return err
	}
	if len(teams) == 0 {
		if _, err := s.CreateTeam(ctx, "default"); err != nil {
			return err
		}
	}

	// Ensure a manager and a couple of agents exist for default team.
	agents, err := s.ListAgents(ctx, "default")
	if err != nil {
		// default might not exist if user renamed; ignore.
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

// CreateMessage adds a message to the mailbox (sender -> recipient).
func (s *sqliteStore) CreateMessage(ctx context.Context, teamName, sender, recipient, content string) (int64, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC().Unix()
	res, err := s.DB.ExecContext(ctx, `INSERT INTO messages(team_id, sender, recipient, content, created_at) VALUES(?, ?, ?, ?, ?)`,
		team.TeamID, sender, recipient, content, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListMessages returns messages for a team, optionally filtered by recipient (inbox for that recipient).
func (s *sqliteStore) ListMessages(ctx context.Context, teamName string, recipient string, limit int) ([]Message, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	q := `SELECT message_id, team_id, sender, recipient, content, created_at, processed_at FROM messages WHERE team_id = ?`
	args := []any{team.TeamID}
	if recipient != "" {
		q += ` AND recipient = ?`
		args = append(args, recipient)
	}
	q += ` ORDER BY created_at DESC`
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Message
	for rows.Next() {
		var m Message
		var createdAt int64
		var processedAt sql.NullInt64
		if err := rows.Scan(&m.MessageID, &m.TeamID, &m.Sender, &m.Recipient, &m.Content, &createdAt, &processedAt); err != nil {
			return nil, err
		}
		m.CreatedAt = time.Unix(createdAt, 0).UTC()
		if processedAt.Valid {
			t := time.Unix(processedAt.Int64, 0).UTC()
			m.ProcessedAt = &t
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListUnprocessedMessages returns messages for a recipient that have processed_at IS NULL (oldest first), for daemon to drive turns.
func (s *sqliteStore) ListUnprocessedMessages(ctx context.Context, teamName string, recipient string, limit int) ([]Message, error) {
	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		return nil, err
	}
	if recipient == "" {
		return nil, nil
	}
	q := `SELECT message_id, team_id, sender, recipient, content, created_at, processed_at FROM messages WHERE team_id = ? AND recipient = ? AND processed_at IS NULL ORDER BY created_at ASC`
	args := []any{team.TeamID, recipient}
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Message
	for rows.Next() {
		var m Message
		var createdAt int64
		var processedAt sql.NullInt64
		if err := rows.Scan(&m.MessageID, &m.TeamID, &m.Sender, &m.Recipient, &m.Content, &createdAt, &processedAt); err != nil {
			return nil, err
		}
		m.CreatedAt = time.Unix(createdAt, 0).UTC()
		if processedAt.Valid {
			t := time.Unix(processedAt.Int64, 0).UTC()
			m.ProcessedAt = &t
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// MarkMessageProcessed sets processed_at for a message.
func (s *sqliteStore) MarkMessageProcessed(ctx context.Context, messageID int64) error {
	now := time.Now().UTC().Unix()
	_, err := s.DB.ExecContext(ctx, `UPDATE messages SET processed_at=? WHERE message_id=?`, now, messageID)
	return err
}

func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("t-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
