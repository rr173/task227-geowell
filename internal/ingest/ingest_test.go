package ingest

import (
	"errors"
	"testing"

	"task227-geowell/internal/model"
)

func TestValidateRejectsDuplicateAndNonMonotonicPoints(t *testing.T) {
	tests := []struct {
		name string
		pts  []RawPoint
		want error
	}{
		{
			name: "duplicate sequence",
			pts:  []RawPoint{{Seq: 0, Depth: 1}, {Seq: 0, Depth: 2}},
			want: model.ErrDuplicatePoint,
		},
		{
			name: "depth reverses",
			pts:  []RawPoint{{Seq: 0, Depth: 2}, {Seq: 1, Depth: 1}},
			want: model.ErrDepthNotMonotonic,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := Validate(tc.pts); !errors.Is(err, tc.want) {
				t.Fatalf("Validate() error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestFingerprintIsIndependentOfUploadOrder(t *testing.T) {
	a := []RawPoint{{Seq: 2, Depth: 20, Temp: 30, Pressure: 2}, {Seq: 1, Depth: 10, Temp: 29, Pressure: 1}}
	b := []RawPoint{a[1], a[0]}
	if gotA, gotB := Fingerprint(a), Fingerprint(b); gotA != gotB {
		t.Fatalf("Fingerprint differs by input order: %q vs %q", gotA, gotB)
	}
}
