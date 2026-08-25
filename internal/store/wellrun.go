package store

import (
	"database/sql"
	"errors"
	"fmt"

	"task227-geowell/internal/model"
)

// CreateWellRun inserts a new well run. code must be unique (idempotent key).
func (s *Store) CreateWellRun(r *model.WellRun) error {
	r.CreatedAt = nowISO()
	r.UpdatedAt = r.CreatedAt
	if r.State == "" {
		r.State = model.WellRunCollecting
	}
	const q = `INSERT INTO well_runs
		(code,name,well,log_date,state,archived,unit_temp,unit_press,depth_basis,disturbance,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`
	res, err := s.db.Exec(q, r.Code, r.Name, r.Well, r.LogDate, string(r.State),
		boolToInt(r.Archived), string(r.UnitTemp), string(r.UnitPress), r.DepthBasis,
		boolToInt(r.Disturbance), r.CreatedAt, r.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert well run: %w", err)
	}
	id, _ := res.LastInsertId()
	r.ID = id
	return nil
}

// GetWellRun fetches a well run by id.
func (s *Store) GetWellRun(id int64) (*model.WellRun, error) {
	const q = `SELECT id,code,name,well,log_date,state,archived,unit_temp,unit_press,depth_basis,disturbance,created_at,updated_at
		FROM well_runs WHERE id=?`
	return scanWellRun(s.db.QueryRow(q, id))
}

// GetWellRunByCode fetches a well run by business code (idempotent lookup).
func (s *Store) GetWellRunByCode(code string) (*model.WellRun, error) {
	const q = `SELECT id,code,name,well,log_date,state,archived,unit_temp,unit_press,depth_basis,disturbance,created_at,updated_at
		FROM well_runs WHERE code=?`
	return scanWellRun(s.db.QueryRow(q, code))
}

// ListWellRuns returns all well runs, optionally filtered by well name.
func (s *Store) ListWellRuns(well string) ([]*model.WellRun, error) {
	var rows *sql.Rows
	var err error
	if well == "" {
		rows, err = s.db.Query(`SELECT id,code,name,well,log_date,state,archived,unit_temp,unit_press,depth_basis,disturbance,created_at,updated_at FROM well_runs ORDER BY id`)
	} else {
		rows, err = s.db.Query(`SELECT id,code,name,well,log_date,state,archived,unit_temp,unit_press,depth_basis,disturbance,created_at,updated_at FROM well_runs WHERE well=? ORDER BY id`, well)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.WellRun
	for rows.Next() {
		wr, err := scanWellRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, wr)
	}
	return out, rows.Err()
}

// UpdateWellRunState persists a state transition.
func (s *Store) UpdateWellRunState(id int64, st model.WellRunState, archived bool) error {
	_, err := s.db.Exec(`UPDATE well_runs SET state=?, archived=?, updated_at=? WHERE id=?`,
		string(st), boolToInt(archived), nowISO(), id)
	return err
}

// SetDepthBasis and disturbance flag persisted after calibration.
func (s *Store) SetWellRunMeta(id int64, depthBasis float64, disturbance bool) error {
	_, err := s.db.Exec(`UPDATE well_runs SET depth_basis=?, disturbance=?, updated_at=? WHERE id=?`,
		depthBasis, boolToInt(disturbance), nowISO(), id)
	return err
}

func scanWellRun(r rowScanner) (*model.WellRun, error) {
	var (
		wr                              model.WellRun
		state, unitT, unitP, created, updated string
		archived, disturbance          int
	)
	if err := r.Scan(&wr.ID, &wr.Code, &wr.Name, &wr.Well, &wr.LogDate, &state,
		&archived, &unitT, &unitP, &wr.DepthBasis, &disturbance, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	wr.State = model.WellRunState(state)
	wr.UnitTemp = model.TemperatureUnit(unitT)
	wr.UnitPress = model.PressureUnit(unitP)
	wr.Archived = archived != 0
	wr.Disturbance = disturbance != 0
	wr.CreatedAt = created
	wr.UpdatedAt = updated
	return &wr, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}
