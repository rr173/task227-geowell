// Package model defines the core domain entities, state machines and errors
// for the geothermal well temperature/pressure profile layering service.
package model

import (
	"errors"
	"fmt"
)

// Domain errors returned by the service layer.
var (
	ErrNotFound          = errors.New("resource not found")
	ErrDepthNotMonotonic = errors.New("depth values are not strictly increasing")
	ErrUnitNotDeclared   = errors.New("measurement unit not declared")
	ErrDuplicatePoint    = errors.New("duplicate measurement point for well run")
	ErrArchivedMutation  = errors.New("well run is archived and cannot be modified")
	ErrBadTransition     = errors.New("illegal state transition")
	ErrLayeringLocked    = errors.New("layering already in progress for well run")
	ErrInvalidSnapshot   = errors.New("snapshot is not in a publishable state")
)

// WellRunState is the lifecycle of a well logging run.
type WellRunState string

const (
	WellRunCollecting     WellRunState = "collecting"      // 采集中
	WellRunPendingLayering WellRunState = "pending_layering" // 待分层
	WellRunLayered        WellRunState = "layered"        // 已分层
	WellRunArchived       WellRunState = "archived"       // 已封存
)

// wellRunTransitions enforces the allowed transitions. Depth correction on an
// already-layered run invalidates its profile (the corrected depths change the
// segment boundaries), so layered may fall back to pending_layering to force a
// fresh layering pass before the run is compared or archived again.
var wellRunTransitions = map[WellRunState][]WellRunState{
	WellRunCollecting:      {WellRunPendingLayering, WellRunArchived},
	WellRunPendingLayering: {WellRunLayered, WellRunArchived},
	WellRunLayered:         {WellRunPendingLayering, WellRunArchived},
	WellRunArchived:        {},
}

// CanTransition reports whether moving from -> to is permitted.
func (s WellRunState) CanTransition(to WellRunState) bool {
	for _, n := range wellRunTransitions[s] {
		if n == to {
			return true
		}
	}
	return false
}

// PointState is the calibration state of a single measurement point.
type PointState string

const (
	PointPendingCal    PointState = "pending_cal"     // 待校准
	PointValid         PointState = "valid"          // 有效
	PointDepthConflict PointState = "depth_conflict" // 深度冲突
	PointMissing       PointState = "missing"        // 缺失
)

// SegmentState is the lifecycle of a detected well segment.
type SegmentState string

const (
	SegCandidate SegmentState = "candidate" // 候选
	SegStable    SegmentState = "stable"    // 稳定
	SegAnomaly   SegmentState = "anomaly"   // 异常
	SegConfirmed SegmentState = "confirmed" // 确认
)

// SegmentStateTransitions enforces candidate / stable / anomaly -> confirmed.
var segmentTransitions = map[SegmentState][]SegmentState{
	SegCandidate: {SegStable, SegAnomaly, SegConfirmed},
	SegStable:    {SegConfirmed},
	SegAnomaly:   {SegConfirmed},
	SegConfirmed: {},
}

// CanTransition reports whether moving from -> to is permitted for segments.
func (s SegmentState) CanTransition(to SegmentState) bool {
	for _, n := range segmentTransitions[s] {
		if n == to {
			return true
		}
	}
	return false
}

// SnapshotState is the lifecycle of a comparison snapshot.
type SnapshotState string

const (
	SnapDraft       SnapshotState = "draft"        // 草稿
	SnapUnderReview SnapshotState = "under_review" // 待复核
	SnapPublished   SnapshotState = "published"    // 发布
	SnapSuperseded  SnapshotState = "superseded"   // 替代
)

// SnapshotStateTransitions enforces draft -> under_review -> published -> superseded.
var snapshotTransitions = map[SnapshotState][]SnapshotState{
	SnapDraft:       {SnapUnderReview, SnapSuperseded},
	SnapUnderReview: {SnapPublished, SnapSuperseded},
	SnapPublished:   {SnapSuperseded},
	SnapSuperseded:  {},
}

// CanTransition reports whether moving from -> to is permitted for snapshots.
func (s SnapshotState) CanTransition(to SnapshotState) bool {
	for _, n := range snapshotTransitions[s] {
		if n == to {
			return true
		}
	}
	return false
}

// TemperatureUnit and PressureUnit declare the declared measurement unit.
type TemperatureUnit string

const (
	TempCelsius TemperatureUnit = "C"
	TempKelvin  TemperatureUnit = "K"
)

type PressureUnit string

const (
	PressMPa  PressureUnit = "MPa"
	PressBar  PressureUnit = "bar"
	PressKPa  PressureUnit = "kPa"
)

// WellRun is a single logging campaign in one well.
type WellRun struct {
	ID          int64        `json:"id"`
	Code        string       `json:"code"`        // unique business code, idempotent key
	Name        string       `json:"name"`
	Well        string       `json:"well"`        // well identifier
	LogDate     string       `json:"log_date"`    // ISO date of logging
	State       WellRunState `json:"state"`
	Archived    bool         `json:"archived"`
	UnitTemp    TemperatureUnit `json:"unit_temp"`
	UnitPress   PressureUnit    `json:"unit_press"`
	DepthBasis  float64      `json:"depth_basis"` // corrected depth datum (m)
	Disturbance bool         `json:"disturbance"` // wellhead disturbance flagged
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
}

// MeasurePoint is one sampled depth reading within a well run.
type MeasurePoint struct {
	ID        int64      `json:"id"`
	WellRunID int64      `json:"well_run_id"`
	Seq       int        `json:"seq"`        // measurement sequence, idempotent within run
	DepthRaw  float64    `json:"depth_raw"`  // raw measured depth (m)
	DepthCal  float64    `json:"depth_cal"`  // depth after datum correction (m)
	Temp      float64    `json:"temp"`       // temperature in declared unit
	Pressure  float64    `json:"pressure"`   // pressure in declared unit
	State     PointState `json:"state"`
	Gap       bool       `json:"gap"`        // missing/undeterminable segment
	CreatedAt string     `json:"created_at"`
}

// Segment is a detected well interval with a gradient signature.
type Segment struct {
	ID           int64       `json:"id"`
	WellRunID    int64       `json:"well_run_id"`
	Index        int         `json:"index"`
	TopDepth     float64     `json:"top_depth"`
	BottomDepth  float64     `json:"bottom_depth"`
	TempGrad     float64     `json:"temp_grad"`    // dT/dz (unit/m)
	PressGrad    float64     `json:"press_grad"`   // dP/dz (unit/m)
	GradientJump float64     `json:"gradient_jump"` // |Δgradient| at boundary
	State        SegmentState `json:"state"`
	Confirmed    bool        `json:"confirmed"`
	CreatedAt    string      `json:"created_at"`
}

// ComparisonSnapshot captures a published cross-run boundary comparison.
type ComparisonSnapshot struct {
	ID        int64         `json:"id"`
	Code      string        `json:"code"` // unique per well, versioned
	Well      string        `json:"well"`
	Version   int           `json:"version"`
	State     SnapshotState `json:"state"`
	BaselineRunID int64     `json:"baseline_run_id"`
	TargetRunID   int64     `json:"target_run_id"`
	Detail    string        `json:"detail"` // JSON: boundary movements
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

// BoundaryMovement describes how a gradient boundary moved between two runs.
type BoundaryMovement struct {
	DepthBaseline float64 `json:"depth_baseline"`
	DepthTarget   float64 `json:"depth_target"`
	DeltaDepth    float64 `json:"delta_depth"`
	Anomaly       bool    `json:"anomaly"`
	Note          string  `json:"note"`
}

// StateError is a typed error carrying the illegal transition.
type StateError struct {
	From string
	To   string
}

func (e *StateError) Error() string {
	return fmt.Sprintf("%v: %s -> %s", ErrBadTransition, e.From, e.To)
}
