// Package store persists scan snapshots for historical drift analysis.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/daemon-blockint-tech/ouroboros/internal/model"
)

// SQLite keeps component observations keyed by stable ID per host.
type SQLite struct {
	db *sql.DB
}

// Open opens or creates a SQLite database at path.
func Open(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &SQLite{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS scans (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id TEXT NOT NULL,
  host_id TEXT NOT NULL,
  profile TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  scanner_version TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS components (
  stable_id TEXT NOT NULL,
  host_id TEXT NOT NULL,
  ecosystem TEXT NOT NULL,
  package_name TEXT NOT NULL,
  version TEXT NOT NULL,
  first_seen TEXT NOT NULL,
  last_seen TEXT NOT NULL,
  last_run_id TEXT NOT NULL,
  record_json TEXT NOT NULL,
  PRIMARY KEY (host_id, stable_id)
);
CREATE INDEX IF NOT EXISTS idx_components_host ON components(host_id);
`)
	return err
}

// Close closes the database.
func (s *SQLite) Close() error {
	return s.db.Close()
}

// BeginScan inserts a scan row and returns its id.
func (s *SQLite) BeginScan(ctx context.Context, runID, hostID, profile, scannerVersion string, started time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO scans(run_id, host_id, profile, started_at, scanner_version) VALUES (?,?,?,?,?)`,
		runID, hostID, profile, started.UTC().Format(time.RFC3339Nano), scannerVersion,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// FinishScan marks scan completion.
func (s *SQLite) FinishScan(ctx context.Context, scanID int64, finished time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE scans SET finished_at=? WHERE id=?`,
		finished.UTC().Format(time.RFC3339Nano), scanID,
	)
	return err
}

// UpsertComponent records or updates one package observation.
func (s *SQLite) UpsertComponent(ctx context.Context, hostID, runID string, r model.Record, seen time.Time) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	ts := seen.UTC().Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(ctx, `
INSERT INTO components(stable_id, host_id, ecosystem, package_name, version, first_seen, last_seen, last_run_id, record_json)
VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT(host_id, stable_id) DO UPDATE SET
  version=excluded.version,
  last_seen=excluded.last_seen,
  last_run_id=excluded.last_run_id,
  record_json=excluded.record_json,
  ecosystem=excluded.ecosystem,
  package_name=excluded.package_name
`, r.StableID(), hostID, r.Ecosystem, r.PackageName, r.Version, ts, ts, runID, string(b))
	return err
}

// DriftRow is a component whose version changed between first and last observation.
type DriftRow struct {
	StableID    string
	Ecosystem   string
	PackageName string
	Version     string
	FirstSeen   string
	LastSeen    string
}

// VersionDrift returns rows where the same stable_id was seen with different versions
// (approximated by re-reading record_json is not done — callers compare version column
// across runs in future; for now lists components updated in last N days).
func (s *SQLite) VersionDrift(ctx context.Context, hostID string, since time.Time) ([]DriftRow, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT stable_id, ecosystem, package_name, version, first_seen, last_seen
FROM components
WHERE host_id=? AND first_seen != last_seen AND last_seen >= ?
ORDER BY last_seen DESC
`, hostID, since.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DriftRow
	for rows.Next() {
		var d DriftRow
		if err := rows.Scan(&d.StableID, &d.Ecosystem, &d.PackageName, &d.Version, &d.FirstSeen, &d.LastSeen); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// PathDefault returns the default agent database path under home.
func PathDefault(home string) string {
	return fmt.Sprintf("%s/.ouroboros/agent.db", home)
}
