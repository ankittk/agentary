package postgres

import (
	"context"
	"embed"
	"errors"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ankittk/agentary/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store is the PostgreSQL implementation of store.Store.
type Store struct {
	Pool *pgxpool.Pool
}

// Open opens a PostgreSQL connection pool and runs migrations. dsn may be empty to use DATABASE_URL env.
func Open(dsn string) (store.Store, error) {
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		return nil, errors.New("postgres DSN or DATABASE_URL required")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 20
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}
	s := &Store{Pool: pool}
	if err := s.Migrate(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the connection pool.
func (s *Store) Close() error {
	if s == nil || s.Pool == nil {
		return nil
	}
	s.Pool.Close()
	return nil
}

// Migrate runs pending migrations (only those not already in schema_migrations).
func (s *Store) Migrate(ctx context.Context) error {
	applied := make(map[int]bool)
	rows, err := s.Pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var v int
			if err := rows.Scan(&v); err != nil {
				break
			}
			applied[v] = true
		}
	}

	type mig struct{ version int; name, sql string }
	var migs []mig
	files, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		v, err := strconv.Atoi(strings.SplitN(strings.TrimSuffix(f.Name(), ".sql"), "_", 2)[0])
		if err != nil {
			continue
		}
		if applied[v] {
			continue
		}
		body, _ := migrationsFS.ReadFile("migrations/" + f.Name())
		migs = append(migs, mig{v, f.Name(), string(body)})
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })

	for _, m := range migs {
		if _, err := s.Pool.Exec(ctx, m.sql); err != nil && !strings.Contains(err.Error(), "already exists") {
			return err
		}
		if _, err := s.Pool.Exec(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES($1, $2) ON CONFLICT (version) DO NOTHING`, m.version, time.Now().Unix()); err != nil {
			return err
		}
	}
	return nil
}
