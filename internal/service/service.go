// Package service is the orchestration layer that wires the business modules
// (ingest, datum, layer, compare, snapshot) to the store and enforces the
// state machines and concurrency boundaries declared in the requirement.
package service

import (
	"fmt"
	"sync"

	"task227-geowell/internal/calibrate"
	"task227-geowell/internal/compare"
	"task227-geowell/internal/datum"
	"task227-geowell/internal/ingest"
	"task227-geowell/internal/layer"
	"task227-geowell/internal/model"
	"task227-geowell/internal/snapshot"
	"task227-geowell/internal/store"
	"task227-geowell/internal/validate"
)

// Service orchestrates well-run lifecycle operations.
type Service struct {
	store   *store.Store
	layerP  layer.Params
	cmpTol  compare.Tolerance
	mu      sync.Mutex // serializes layering per run
	layering map[int64]bool
}

// New constructs a Service with default algorithm parameters.
func New(s *store.Store) *Service {
	return &Service{
		store:    s,
		layerP:   layer.DefaultParams(),
		cmpTol:   compare.DefaultTolerance(),
		layering: make(map[int64]bool),
	}
}

// ---- Well run lifecycle ----

// CreateWellRun validates units and creates a collecting well run.
func (svc *Service) CreateWellRun(code, name, well, logDate, unitTemp, unitPress string) (*model.WellRun, error) {
	tu, pu, err := ingest.EncodeUnit(unitTemp, unitPress)
	if err != nil {
		return nil, err
	}
	r := &model.WellRun{
		Code:     code,
		Name:     name,
		Well:     well,
		LogDate:  logDate,
		State:    model.WellRunCollecting,
		UnitTemp: tu,
		UnitPress: pu,
	}
	if err := svc.store.CreateWellRun(r); err != nil {
		return nil, err
	}
	return r, nil
}

// IngestPoints validates, converts and persists a batch of raw points. It also
// transitions a collecting run toward pending_layering once data exists.
func (svc *Service) IngestPoints(runID int64, raw []ingest.RawPoint) error {
	r, err := svc.store.GetWellRun(runID)
	if err != nil {
		return err
	}
	if r.Archived {
		return model.ErrArchivedMutation
	}
	if len(raw) == 0 {
		return fmt.Errorf("empty point batch")
	}
	if err := ingest.Validate(raw); err != nil {
		return err
	}
	pts := ingest.ToMeasurePoints(runID, raw)
	if err := svc.store.BulkUpsertPoints(runID, pts); err != nil {
		return err
	}
	// Once points exist, the run may move to pending_layering.
	if r.State == model.WellRunCollecting {
		if err := svc.transitionRun(runID, model.WellRunPendingLayering); err != nil {
			return err
		}
	}
	return nil
}

// CorrectDepth computes and applies the datum correction, then persists the
// corrected depths and disturbance flag on the run.
func (svc *Service) CorrectDepth(runID int64, referenceDepth float64) (*datum.Correction, error) {
	r, err := svc.store.GetWellRun(runID)
	if err != nil {
		return nil, err
	}
	if r.Archived {
		return nil, model.ErrArchivedMutation
	}
	pts, err := svc.store.ListPoints(runID)
	if err != nil {
		return nil, err
	}
	if len(pts) == 0 {
		return nil, fmt.Errorf("no points to correct")
	}
	c := datum.Compute(pts, referenceDepth)
	cal := datum.Apply(pts, c)
	if err := datum.VerifyMonotonic(cal); err != nil {
		return nil, err
	}
	if err := svc.store.BulkUpsertPoints(runID, cal); err != nil {
		return nil, err
	}
	if err := svc.store.SetWellRunMeta(runID, c.DepthBasis, c.Disturbance); err != nil {
		return nil, err
	}
	return &c, nil
}

// Layer runs the segmentation pipeline (serial per run) and persists segments.
func (svc *Service) Layer(runID int64) ([]model.Segment, error) {
	svc.mu.Lock()
	if svc.layering[runID] {
		svc.mu.Unlock()
		return nil, model.ErrLayeringLocked
	}
	svc.layering[runID] = true
	svc.mu.Unlock()
	defer func() {
		svc.mu.Lock()
		delete(svc.layering, runID)
		svc.mu.Unlock()
	}()

	r, err := svc.store.GetWellRun(runID)
	if err != nil {
		return nil, err
	}
	if r.Archived {
		return nil, model.ErrArchivedMutation
	}
	pts, err := svc.store.ListPoints(runID)
	if err != nil {
		return nil, err
	}
	if err := validate.RunReadiness(r, len(pts), 0); err != nil {
		return nil, err
	}
	grid := layer.Resample(pts, svc.layerP.GridStep)
	if len(grid) < 2 {
		return nil, fmt.Errorf("insufficient valid points after resampling")
	}
	segs := layer.Detect(grid, svc.layerP)
	states := layer.Classify(segs)
	modelSegs := layer.ToModel(runID, segs, states)
	if err := svc.store.ReplaceSegments(runID, modelSegs); err != nil {
		return nil, err
	}
	if err := svc.transitionRun(runID, model.WellRunLayered); err != nil {
		return nil, err
	}
	return modelSegs, nil
}

// CorrectDepthPlan applies a calibration strategy (offset/tie_point/stretch)
// using the supplied markers, then persists the corrected depths. It is an
// alternative to CorrectDepth for runs with known calibration markers.
func (svc *Service) CorrectDepthPlan(runID int64, strat calibrate.Strategy, markers []calibrate.Marker) (*calibrate.Plan, error) {
	r, err := svc.store.GetWellRun(runID)
	if err != nil {
		return nil, err
	}
	if r.Archived {
		return nil, model.ErrArchivedMutation
	}
	pts, err := svc.store.ListPoints(runID)
	if err != nil {
		return nil, err
	}
	if len(pts) == 0 {
		return nil, fmt.Errorf("no points to correct")
	}
	plan, err := calibrate.Build(strat, markers)
	if err != nil {
		return nil, err
	}
	cal := calibrate.ApplyPoints(pts, plan)
	if err := datum.VerifyMonotonic(cal); err != nil {
		return nil, err
	}
	if err := svc.store.BulkUpsertPoints(runID, cal); err != nil {
		return nil, err
	}
	if err := svc.store.SetWellRunMeta(runID, plan.Offset, r.Disturbance); err != nil {
		return nil, err
	}
	return &plan, nil
}

// ConfirmSegment transitions a candidate/stable/anomaly segment to confirmed.
func (svc *Service) ConfirmSegment(segID int64, st model.SegmentState) error {
	// segment lookup is done in httpapi via store; here we just persist.
	return svc.store.UpdateSegmentState(segID, st, st == model.SegConfirmed)
}

// CompareRuns builds a cross-run boundary comparison and returns the movements.
func (svc *Service) CompareRuns(baselineID, targetID int64) ([]compare.Movement, error) {
	bRun, err := svc.store.GetWellRun(baselineID)
	if err != nil {
		return nil, err
	}
	tRun, err := svc.store.GetWellRun(targetID)
	if err != nil {
		return nil, err
	}
	if bRun.Well != tRun.Well {
		return nil, fmt.Errorf("runs belong to different wells")
	}
	if err := validate.UnitConsistent(bRun, tRun); err != nil {
		return nil, err
	}
	bSegs, err := svc.store.ListSegments(baselineID)
	if err != nil {
		return nil, err
	}
	tSegs, err := svc.store.ListSegments(targetID)
	if err != nil {
		return nil, err
	}
	bBounds := compare.BoundariesFromSegments(bSegs)
	tBounds := compare.BoundariesFromSegments(tSegs)
	return compare.Compare(bBounds, tBounds, svc.cmpTol), nil
}

// PublishComparison builds, reviews and publishes a versioned snapshot. The
// wellhead disturbance on either run blocks publication until excluded.
func (svc *Service) PublishComparison(baselineID, targetID int64) (*model.ComparisonSnapshot, error) {
	bRun, err := svc.store.GetWellRun(baselineID)
	if err != nil {
		return nil, err
	}
	tRun, err := svc.store.GetWellRun(targetID)
	if err != nil {
		return nil, err
	}
	moves, err := svc.CompareRuns(baselineID, targetID)
	if err != nil {
		return nil, err
	}
	detail, err := compare.BuildDetail(moves)
	if err != nil {
		return nil, err
	}
	latest, err := svc.store.LatestVersion(bRun.Well)
	if err != nil {
		return nil, err
	}
	pub := snapshot.NewPublisher(bRun.Well, latest)
	code, ver := pub.NextCode()
	snap := pub.Build(code, ver, baselineID, targetID, detail)
	if err := svc.store.CreateSnapshot(snap); err != nil {
		return nil, err
	}
	// under_review -> published with disturbance guard.
	if err := svc.store.UpdateSnapshotState(snap.ID, model.SnapUnderReview, ""); err != nil {
		return nil, err
	}
	snap.State = model.SnapUnderReview
	if err := validate.DisturbanceResolved(bRun, tRun); err != nil {
		return snap, err // left in under_review for engineer to resolve
	}
	if err := snapshot.CanPublish(snap, bRun.Disturbance, tRun.Disturbance); err != nil {
		return snap, err // left in under_review for engineer to resolve
	}
	if err := svc.store.SupersedePublished(bRun.Well, snap.ID); err != nil {
		return nil, err
	}
	if err := svc.store.UpdateSnapshotState(snap.ID, model.SnapPublished, ""); err != nil {
		return nil, err
	}
	return svc.store.GetSnapshot(snap.ID)
}

// ArchiveRun seals a layered well run so it can no longer be mutated.
func (svc *Service) ArchiveRun(runID int64) error {
	r, err := svc.store.GetWellRun(runID)
	if err != nil {
		return err
	}
	if r.State != model.WellRunLayered {
		return fmt.Errorf("%w: only layered runs can be archived (state=%s)", model.ErrBadTransition, r.State)
	}
	return svc.transitionRun(runID, model.WellRunArchived)
}

// transitionRun applies a well-run state transition, persisting archived flag.
func (svc *Service) transitionRun(runID int64, to model.WellRunState) error {
	r, err := svc.store.GetWellRun(runID)
	if err != nil {
		return err
	}
	if !r.State.CanTransition(to) {
		return &model.StateError{From: string(r.State), To: string(to)}
	}
	return svc.store.UpdateWellRunState(runID, to, to == model.WellRunArchived)
}
