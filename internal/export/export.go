// Package export serializes comparison snapshots and run profiles to portable
// formats (JSON / CSV) so engineers can archive or share the results.
package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"task227-geowell/internal/model"
)

// SnapshotJSON marshals a snapshot with its parsed movements.
func SnapshotJSON(snap *model.ComparisonSnapshot) (string, error) {
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SegmentsJSON marshals a run's segments.
func SegmentsJSON(segs []model.Segment) (string, error) {
	b, err := json.MarshalIndent(segs, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SegmentsCSV renders segments as comma-separated values with a header row.
func SegmentsCSV(segs []model.Segment) string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"index", "top_depth", "bottom_depth", "temp_grad", "press_grad", "gradient_jump", "state", "confirmed"})
	for _, s := range segs {
		_ = w.Write([]string{
			strconv.Itoa(s.Index),
			fmt.Sprintf("%.3f", s.TopDepth),
			fmt.Sprintf("%.3f", s.BottomDepth),
			fmt.Sprintf("%.6f", s.TempGrad),
			fmt.Sprintf("%.6f", s.PressGrad),
			fmt.Sprintf("%.6f", s.GradientJump),
			string(s.State),
			strconv.FormatBool(s.Confirmed),
		})
	}
	w.Flush()
	return b.String()
}

// PointsCSV renders measurement points as CSV with a header row.
func PointsCSV(pts []model.MeasurePoint) string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"seq", "depth_raw", "depth_cal", "temp", "pressure", "state", "gap"})
	for _, p := range pts {
		_ = w.Write([]string{
			strconv.Itoa(p.Seq),
			fmt.Sprintf("%.3f", p.DepthRaw),
			fmt.Sprintf("%.3f", p.DepthCal),
			fmt.Sprintf("%.3f", p.Temp),
			fmt.Sprintf("%.3f", p.Pressure),
			string(p.State),
			strconv.FormatBool(p.Gap),
		})
	}
	w.Flush()
	return b.String()
}
