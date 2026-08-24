// Package store provides SQLite-backed persistence for the geothermal well
// profile layering service. It uses the pure-Go modernc.org/sqlite driver so
// that builds stay CGO-free and offline-capable.
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps the underlying *sql.DB and exposes CRUD for every entity.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at dbPath and applies the
// schema migration. It is safe to call on an existing database.
func Open(dbPath string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB exposes the underlying handle for advanced callers (tests).
func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS well_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			well TEXT NOT NULL,
			log_date TEXT NOT NULL,
			state TEXT NOT NULL,
			archived INTEGER NOT NULL DEFAULT 0,
			unit_temp TEXT NOT NULL,
			unit_press TEXT NOT NULL,
			depth_basis REAL NOT NULL DEFAULT 0,
			disturbance INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS measure_points (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			well_run_id INTEGER NOT NULL REFERENCES well_runs(id),
			seq INTEGER NOT NULL,
			depth_raw REAL NOT NULL,
			depth_cal REAL NOT NULL,
			temp REAL NOT NULL,
			pressure REAL NOT NULL,
			state TEXT NOT NULL,
			gap INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			UNIQUE(well_run_id, seq)
		)`,
		`CREATE TABLE IF NOT EXISTS segments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			well_run_id INTEGER NOT NULL REFERENCES well_runs(id),
			idx INTEGER NOT NULL,
			top_depth REAL NOT NULL,
			bottom_depth REAL NOT NULL,
			temp_grad REAL NOT NULL,
			press_grad REAL NOT NULL,
			gradient_jump REAL NOT NULL DEFAULT 0,
			state TEXT NOT NULL,
			confirmed INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			UNIQUE(well_run_id, idx)
		)`,
		`CREATE TABLE IF NOT EXISTS comparison_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			well TEXT NOT NULL,
			version INTEGER NOT NULL,
			state TEXT NOT NULL,
			baseline_run_id INTEGER NOT NULL,
			target_run_id INTEGER NOT NULL,
			detail TEXT NOT NULL DEFAULT '[]',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_points_run ON measure_points(well_run_id)`,
		`CREATE INDEX IF NOT EXISTS idx_seg_run ON segments(well_run_id)`,
	}
	for _, st := range stmts {
		if _, err := s.db.Exec(st); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

func nowISO() string { return time.Now().UTC().Format(time.RFC3339) }
