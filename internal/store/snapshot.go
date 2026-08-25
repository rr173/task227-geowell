package store

import (
	"database/sql"
	"errors"
	"fmt"

	"task227-geowell/internal/model"
)

// CreateSnapshot inserts a new comparison snapshot with a versioned code.
func (s *Store) CreateSnapshot(snap *model.ComparisonSnapshot) error {
	snap.CreatedAt = nowISO()
	snap.UpdatedAt = snap.CreatedAt
	if snap.State == "" {
		snap.State = model.SnapDraft
	}
	if snap.Detail == "" {
		snap.Detail = "[]"
	}
	const q = `INSERT INTO comparison_snapshots
		(code,well,version,state,baseline_run_id,target_run_id,detail,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?)`
	res, err := s.db.Exec(q, snap.Code, snap.Well, snap.Version, string(snap.State),
		snap.BaselineRunID, snap.TargetRunID, snap.Detail, snap.CreatedAt, snap.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	id, _ := res.LastInsertId()
	snap.ID = id
	return nil
}

// GetSnapshot returns a snapshot by id.
func (s *Store) GetSnapshot(id int64) (*model.ComparisonSnapshot, error) {
	const q = `SELECT id,code,well,version,state,baseline_run_id,target_run_id,detail,created_at,updated_at
		FROM comparison_snapshots WHERE id=?`
	var (
		snap                                                  model.ComparisonSnapshot
		state, detail, created, updated                      string
	)
	err := s.db.QueryRow(q, id).Scan(&snap.ID, &snap.Code, &snap.Well, &snap.Version,
		&state, &snap.BaselineRunID, &snap.TargetRunID, &detail, &created, &updated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	snap.State = model.SnapshotState(state)
	snap.Detail = detail
	snap.CreatedAt = created
	snap.UpdatedAt = updated
	return &snap, nil
}

// ListSnapshotsByWell returns snapshots for a well ordered by version desc.
func (s *Store) ListSnapshotsByWell(well string) ([]*model.ComparisonSnapshot, error) {
	rows, err := s.db.Query(`SELECT id,code,well,version,state,baseline_run_id,target_run_id,detail,created_at,updated_at
		FROM comparison_snapshots WHERE well=? ORDER BY version DESC`, well)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ComparisonSnapshot
	for rows.Next() {
		var (
			snap                                             model.ComparisonSnapshot
			state, detail, created, updated                 string
		)
		if err := rows.Scan(&snap.ID, &snap.Code, &snap.Well, &snap.Version, &state,
			&snap.BaselineRunID, &snap.TargetRunID, &detail, &created, &updated); err != nil {
			return nil, err
		}
		snap.State = model.SnapshotState(state)
		snap.Detail = detail
		snap.CreatedAt = created
		snap.UpdatedAt = updated
		out = append(out, &snap)
	}
	return out, rows.Err()
}

// SupersedePublished marks any previously published snapshot for this well as superseded.
func (s *Store) SupersedePublished(well string, exceptID int64) error {
	_, err := s.db.Exec(`UPDATE comparison_snapshots SET state=?, updated_at=? WHERE well=? AND state=? AND id!=?`,
		string(model.SnapSuperseded), nowISO(), well, string(model.SnapPublished), exceptID)
	return err
}

// SupersedeReviewingForRun marks every still-under-review snapshot that depends
// on the given run (as baseline or target) as superseded. Archiving a source
// run ends the unfinished publication those snapshots represent, so they cannot
// remain stuck in under_review. Only under_review snapshots are touched, which
// is the single legal transition to superseded.
func (s *Store) SupersedeReviewingForRun(runID int64) error {
	_, err := s.db.Exec(`UPDATE comparison_snapshots SET state=?, updated_at=?
		WHERE state=? AND (baseline_run_id=? OR target_run_id=?)`,
		string(model.SnapSuperseded), nowISO(), string(model.SnapUnderReview), runID, runID)
	return err
}

// UpdateSnapshotState persists a snapshot state transition.
func (s *Store) UpdateSnapshotState(id int64, st model.SnapshotState, detail string) error {
	if detail == "" {
		_, err := s.db.Exec(`UPDATE comparison_snapshots SET state=?, updated_at=? WHERE id=?`,
			string(st), nowISO(), id)
		return err
	}
	_, err := s.db.Exec(`UPDATE comparison_snapshots SET state=?, detail=?, updated_at=? WHERE id=?`,
		string(st), detail, nowISO(), id)
	return err
}

// LatestVersion returns the max version for a well (0 if none).
func (s *Store) LatestVersion(well string) (int, error) {
	var v sql.NullInt64
	err := s.db.QueryRow(`SELECT MAX(version) FROM comparison_snapshots WHERE well=?`, well).Scan(&v)
	if err != nil {
		return 0, err
	}
	if !v.Valid {
		return 0, nil
	}
	return int(v.Int64), nil
}
