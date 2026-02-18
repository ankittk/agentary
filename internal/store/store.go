package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func openSQLiteDSN(dsn string) (*sqliteStore, error) {
	if dsn == "" {
		return nil, errors.New("sqlite DSN required")
	}
	if !strings.HasPrefix(dsn, "file:") {
		dsn = "file:" + dsn + "?_pragma=busy_timeout(5000)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	s := &sqliteStore{DB: db}
	if err := s.initPragmas(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.Migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.prepareStatements(context.Background()); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

// sqliteStore is the SQLite implementation of Store (internal to this package).
type sqliteStore struct {
	DB *sql.DB
	// Prepared statements for hot paths (prepared at open, closed in Close).
	stmtGetTeamByName    *sql.Stmt
	stmtListTasks100     *sql.Stmt
	stmtCreateTask       *sql.Stmt
	stmtGetTaskByID      *sql.Stmt
	stmtNextRunnable     *sql.Stmt
	stmtClaimTask        *sql.Stmt
	stmtUpdateTaskStatus *sql.Stmt
	stmtUpdateTaskAssign *sql.Stmt
}

// OpenOptions configures how to open the store (driver and location).
type OpenOptions struct {
	Driver string // "sqlite" (default) or "postgres"
	Home   string // for sqlite: directory containing protected/db.sqlite
	DSN    string // for postgres: connection string; or env DATABASE_URL
}

// Open opens the default SQLite store at home/protected/db.sqlite.
func Open(home string) (Store, error) {
	return OpenWithOptions(OpenOptions{Driver: "sqlite", Home: home})
}

// OpenWithOptions opens a store based on driver and options. Driver "" or "sqlite" uses Home or DSN.
// For driver "postgres", the caller must use postgres.Open(dsn) from internal/store/postgres to avoid import cycles.
func OpenWithOptions(opts OpenOptions) (Store, error) {
	if opts.Driver == "postgres" {
		return nil, errors.New("for postgres use postgres.Open(dsn) from github.com/ankittk/agentary/internal/store/postgres")
	}
	if opts.Home == "" && opts.DSN != "" {
		return openSQLiteDSN(opts.DSN)
	}
	return openSQLite(opts.Home)
}

// openSQLite opens SQLite at home (used by Open and OpenWithOptions when driver is sqlite).
func openSQLite(home string) (*sqliteStore, error) {
	dbPath := filepath.Join(home, "protected", "db.sqlite")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}
	dsn := "file:" + dbPath + "?_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	s := &sqliteStore{DB: db}
	if err := s.initPragmas(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.Migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.prepareStatements(context.Background()); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

func (s *sqliteStore) prepareStatements(ctx context.Context) error {
	pairs := []struct {
		dest **sql.Stmt
		q    string
	}{
		{&s.stmtGetTeamByName, `SELECT name, team_id, created_at FROM teams WHERE name = ?`},
		{&s.stmtListTasks100, `SELECT task_id, title, status, assignee, dri, COALESCE(attempt_count,0), workflow_id, current_stage, worktree_path, branch_name, base_sha, repo_name, created_at, updated_at FROM tasks WHERE team_id = ? ORDER BY created_at DESC LIMIT 100`},
		{&s.stmtCreateTask, `INSERT INTO tasks(team_id, title, status, assignee, created_at, updated_at) VALUES(?, ?, ?, NULL, ?, ?)`},
		{&s.stmtGetTaskByID, `SELECT task_id, title, status, assignee, dri, COALESCE(attempt_count,0), workflow_id, current_stage, worktree_path, branch_name, base_sha, repo_name, created_at, updated_at FROM tasks WHERE task_id = ? AND team_id = ?`},
		{&s.stmtNextRunnable, `SELECT task_id, title, status, assignee, dri, COALESCE(attempt_count,0), workflow_id, current_stage, worktree_path, branch_name, base_sha, repo_name, created_at, updated_at FROM tasks WHERE team_id = ? AND status IN ('todo','in_progress') AND (current_stage IS NULL OR current_stage != 'Merging') ORDER BY updated_at ASC LIMIT 1`},
		{&s.stmtClaimTask, `UPDATE tasks SET status='in_progress', assignee=?, updated_at=?, dri=COALESCE(dri, ?) WHERE task_id=? AND team_id=? AND status='todo'`},
		{&s.stmtUpdateTaskStatus, `UPDATE tasks SET status=?, assignee=?, updated_at=? WHERE task_id=?`},
		{&s.stmtUpdateTaskAssign, `UPDATE tasks SET assignee=?, updated_at=? WHERE task_id=?`},
	}
	for _, p := range pairs {
		st, err := s.DB.PrepareContext(ctx, p.q)
		if err != nil {
			return err
		}
		*p.dest = st
	}
	return nil
}

// EnsureSchema creates the store at home, runs migrations, and closes it; used to bootstrap the DB.
func EnsureSchema(home string) error {
	s, err := Open(home)
	if err != nil {
		return err
	}
	return s.Close()
}

func (s *sqliteStore) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	for _, st := range []*sql.Stmt{s.stmtGetTeamByName, s.stmtListTasks100, s.stmtCreateTask, s.stmtGetTaskByID, s.stmtNextRunnable, s.stmtClaimTask, s.stmtUpdateTaskStatus, s.stmtUpdateTaskAssign} {
		if st != nil {
			_ = st.Close()
		}
	}
	return s.DB.Close()
}

func (s *sqliteStore) initPragmas(ctx context.Context) error {
	// WAL yields much better concurrency for read-heavy UI.
	stmts := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA temp_store=MEMORY;",
		// Negative cache_size means KB. Tune for small/medium local workloads.
		"PRAGMA cache_size=-20000;",
	}
	for _, q := range stmts {
		if _, err := s.DB.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func (s *sqliteStore) Migrate(ctx context.Context) error {
	if s == nil || s.DB == nil {
		return errors.New("store not initialized")
	}

	// Ensure migrations table exists even before we run migration files.
	if _, err := s.DB.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at INTEGER NOT NULL
);`); err != nil {
		return err
	}

	applied, err := s.appliedVersions(ctx)
	if err != nil {
		return err
	}

	files, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	var migs []migration
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		v, err := parseMigrationVersion(name)
		if err != nil {
			return err
		}
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		migs = append(migs, migration{Version: v, Name: name, SQL: string(body)})
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].Version < migs[j].Version })

	for _, m := range migs {
		if applied[m.Version] {
			continue
		}
		if err := s.applyMigration(ctx, m); err != nil {
			return fmt.Errorf("migration %s failed: %w", m.Name, err)
		}
	}

	return nil
}

type migration struct {
	Version int
	Name    string
	SQL     string
}

func (s *sqliteStore) appliedVersions(ctx context.Context) (map[int]bool, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func (s *sqliteStore) applyMigration(ctx context.Context, m migration) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`, m.Version, time.Now().Unix()); err != nil {
		return err
	}
	return tx.Commit()
}

func parseMigrationVersion(filename string) (int, error) {
	base := strings.TrimSuffix(filename, ".sql")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) < 1 {
		return 0, fmt.Errorf("invalid migration filename: %s", filename)
	}
	v, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid migration version in %s", filename)
	}
	return v, nil
}
