package sqlite

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

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store is the SQLite implementation of store.Store.
type Store struct {
	DB *sql.DB
}

// Open opens a SQLite database at home/protected/db.sqlite and runs migrations.
func Open(home string) (*Store, error) {
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

	s := &Store{DB: db}
	if err := s.initPragmas(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.Migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

func (s *Store) initPragmas(ctx context.Context) error {
	stmts := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA temp_store=MEMORY;",
		"PRAGMA cache_size=-20000;",
	}
	for _, q := range stmts {
		if _, err := s.DB.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Migrate(ctx context.Context) error {
	if s == nil || s.DB == nil {
		return errors.New("store not initialized")
	}
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

func (s *Store) appliedVersions(ctx context.Context) (map[int]bool, error) {
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

func (s *Store) applyMigration(ctx context.Context, m migration) error {
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
