package store

import (
	"fmt"

	"task227-geowell/internal/model"
)

// ReplaceSegments deletes existing segments for a run and inserts the new set.
// Layering is serial per run so a full replace keeps the set consistent.
func (s *Store) ReplaceSegments(runID int64, segs []model.Segment) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM segments WHERE well_run_id=?`, runID); err != nil {
		return err
	}
	const q = `INSERT INTO segments
		(well_run_id,idx,top_depth,bottom_depth,temp_grad,press_grad,gradient_jump,state,confirmed,created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`
	for _, sg := range segs {
		if sg.CreatedAt == "" {
			sg.CreatedAt = nowISO()
		}
		if _, err := tx.Exec(q, runID, sg.Index, sg.TopDepth, sg.BottomDepth,
			sg.TempGrad, sg.PressGrad, sg.GradientJump, string(sg.State),
			boolToInt(sg.Confirmed), sg.CreatedAt); err != nil {
			return fmt.Errorf("insert segment %d: %w", sg.Index, err)
		}
	}
	return tx.Commit()
}

// ListSegments returns all segments for a run ordered by index.
func (s *Store) ListSegments(runID int64) ([]model.Segment, error) {
	rows, err := s.db.Query(`SELECT id,well_run_id,idx,top_depth,bottom_depth,temp_grad,press_grad,gradient_jump,state,confirmed,created_at
		FROM segments WHERE well_run_id=? ORDER BY idx`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Segment
	for rows.Next() {
		var (
			sg                          model.Segment
			state                       string
			confirmed                   int
		)
		if err := rows.Scan(&sg.ID, &sg.WellRunID, &sg.Index, &sg.TopDepth, &sg.BottomDepth,
			&sg.TempGrad, &sg.PressGrad, &sg.GradientJump, &state, &confirmed, &sg.CreatedAt); err != nil {
			return nil, err
		}
		sg.State = model.SegmentState(state)
		sg.Confirmed = confirmed != 0
		out = append(out, sg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []model.Segment{}
	}
	return out, nil
}

// UpdateSegmentState persists a segment state transition.
func (s *Store) UpdateSegmentState(segID int64, st model.SegmentState, confirmed bool) error {
	_, err := s.db.Exec(`UPDATE segments SET state=?, confirmed=? WHERE id=?`,
		string(st), boolToInt(confirmed), segID)
	return err
}
