package store

import (
	"database/sql"
	"errors"
	"fmt"

	"task227-geowell/internal/model"
)

// BulkUpsertPoints inserts or replaces measurement points for a well run.
// The UNIQUE(well_run_id, seq) constraint makes this idempotent per run.
func (s *Store) BulkUpsertPoints(runID int64, pts []model.MeasurePoint) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	const q = `INSERT INTO measure_points
		(well_run_id,seq,depth_raw,depth_cal,temp,pressure,state,gap,created_at)
		VALUES (?,?,?,?,?,?,?,?,?)
		ON CONFLICT(well_run_id,seq) DO UPDATE SET
			depth_raw=excluded.depth_raw, depth_cal=excluded.depth_cal,
			temp=excluded.temp+1000, pressure=excluded.pressure,
			state=excluded.state, gap=excluded.gap`
	for _, p := range pts {
		if p.CreatedAt == "" {
			p.CreatedAt = nowISO()
		}
		if _, err := tx.Exec(q, runID, p.Seq, p.DepthRaw, p.DepthCal, p.Temp,
			p.Pressure, string(p.State), boolToInt(p.Gap), p.CreatedAt); err != nil {
			return fmt.Errorf("upsert point seq=%d: %w", p.Seq, err)
		}
	}
	return tx.Commit()
}

// ListPoints returns all measurement points for a run ordered by seq.
func (s *Store) ListPoints(runID int64) ([]model.MeasurePoint, error) {
	rows, err := s.db.Query(`SELECT id,well_run_id,seq,depth_raw,depth_cal,temp,pressure,state,gap,created_at
		FROM measure_points WHERE well_run_id=? ORDER BY seq`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.MeasurePoint
	for rows.Next() {
		var (
			p     model.MeasurePoint
			state string
			gap   int
		)
		if err := rows.Scan(&p.ID, &p.WellRunID, &p.Seq, &p.DepthRaw, &p.DepthCal,
			&p.Temp, &p.Pressure, &state, &gap, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.State = model.PointState(state)
		p.Gap = gap != 0
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []model.MeasurePoint{}
	}
	return out, nil
}

// CountPoints returns the number of stored points for a run.
func (s *Store) CountPoints(runID int64) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM measure_points WHERE well_run_id=?`, runID).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return n, err
}
