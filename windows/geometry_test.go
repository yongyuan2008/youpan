package main

import (
	"math"
	"testing"
)

func TestFitGeometryCommonDPIWorkAreas(t *testing.T) {
	tests := []struct {
		name                            string
		workWidth, workHeight           int
		systemScale                     float64
		nonClientWidth, nonClientHeight int
		wantSystemScale                 bool
	}{
		{"1366x768 at 125 percent", 1366, 768, 1.25, 20, 49, false},
		{"1366x768 at 150 percent", 1366, 768, 1.50, 24, 59, false},
		{"1920x1080 at 125 percent", 1920, 1080, 1.25, 20, 49, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fitGeometry(tc.workWidth, tc.workHeight, tc.systemScale, tc.nonClientWidth, tc.nonClientHeight)
			if got.Scale <= 0 || got.Scale > tc.systemScale {
				t.Fatalf("scale=%v system=%v", got.Scale, tc.systemScale)
			}
			if tc.wantSystemScale && math.Abs(got.Scale-tc.systemScale) > 1e-9 {
				t.Fatalf("ordinary screen scale=%v want=%v", got.Scale, tc.systemScale)
			}
			if !tc.wantSystemScale && got.Scale >= tc.systemScale {
				t.Fatalf("constrained screen was not fitted: %+v", got)
			}
			if got.MinOuterWidth > tc.workWidth || got.MinOuterHeight > tc.workHeight {
				t.Fatalf("minimum exceeds work area: %+v", got)
			}
			if got.ClientWidth < int(float64(baseClientWidth)*got.Scale+.5) || got.ClientHeight < int(float64(baseClientHeight)*got.Scale+.5) {
				t.Fatalf("base content clipped: %+v", got)
			}
			if math.Round(float64(got.MinOuterWidth)/got.Scale) < minWindowWidth || math.Round(float64(got.MinOuterHeight)/got.Scale) < minWindowHeight {
				t.Fatalf("stored logical minimum would fail validation: %+v", got)
			}
			if got.MinOuterWidth >= got.ClientWidth+tc.nonClientWidth || got.MinOuterHeight >= got.ClientHeight+tc.nonClientHeight {
				t.Fatalf("default size has no room to shrink: %+v", got)
			}
			w := got.MinOuterWidth - tc.nonClientWidth
			h := got.MinOuterHeight - tc.nonClientHeight
			compact := contentScale(w, h, got.Scale)
			if compact >= got.Scale || math.Round(baseClientWidth*compact) > float64(w) || math.Round(baseClientHeight*compact) > float64(h) {
				t.Fatalf("compact window would clip content: %dx%d at %.3f", w, h, compact)
			}
		})
	}
}

func TestFitGeometryVerySmallWorkAreaIsBounded(t *testing.T) {
	got := fitGeometry(300, 180, 1.5, 24, 59)
	if got.Scale <= 0 || got.MinOuterWidth > 300 || got.MinOuterHeight > 180 || got.ClientWidth < 1 || got.ClientHeight < 1 {
		t.Fatalf("invalid small-work-area fit: %+v", got)
	}
}
