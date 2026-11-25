package main

import "math"

const (
	baseClientWidth  = 744
	baseClientHeight = 661
)

type fittedGeometry struct {
	Scale                         float64
	ClientWidth, ClientHeight     int
	MinOuterWidth, MinOuterHeight int
}

// fitGeometry chooses the largest UI scale that preserves the complete base
// client area inside the current work area. Non-client dimensions are supplied
// by the platform because a system-DPI-aware caption and frame do not shrink
// when the UI scale is reduced.
func fitGeometry(workWidth, workHeight int, systemScale float64, nonClientWidth, nonClientHeight int) fittedGeometry {
	if workWidth < 1 {
		workWidth = 1
	}
	if workHeight < 1 {
		workHeight = 1
	}
	if math.IsNaN(systemScale) || math.IsInf(systemScale, 0) || systemScale <= 0 {
		systemScale = 1
	}
	nonClientWidth = max(0, min(nonClientWidth, workWidth-1))
	nonClientHeight = max(0, min(nonClientHeight, workHeight-1))
	availableWidth := max(1, workWidth-nonClientWidth)
	availableHeight := max(1, workHeight-nonClientHeight)
	scale := math.Min(systemScale, math.Min(float64(availableWidth)/baseClientWidth, float64(availableHeight)/baseClientHeight))
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = math.Min(systemScale, 1/float64(max(baseClientWidth, baseClientHeight)))
	}
	clientWidth := min(availableWidth, max(1, int(float64(baseClientWidth)*scale+.5)))
	clientHeight := min(availableHeight, max(1, int(float64(baseClientHeight)*scale+.5)))
	return fittedGeometry{
		Scale:          scale,
		ClientWidth:    clientWidth,
		ClientHeight:   clientHeight,
		MinOuterWidth:  min(workWidth, clientWidth+nonClientWidth),
		MinOuterHeight: min(workHeight, clientHeight+nonClientHeight),
	}
}
