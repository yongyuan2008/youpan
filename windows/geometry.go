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

// fitGeometry sets the window size units from the work area and system DPI.
// The client UI can shrink further inside an individual window.
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
	scale := contentScale(availableWidth, availableHeight, systemScale)
	clientWidth := min(availableWidth, max(1, int(float64(baseClientWidth)*scale+.5)))
	clientHeight := min(availableHeight, max(1, int(float64(baseClientHeight)*scale+.5)))
	return fittedGeometry{
		Scale:          scale,
		ClientWidth:    clientWidth,
		ClientHeight:   clientHeight,
		MinOuterWidth:  min(workWidth, max(nonClientWidth+1, int(float64(minWindowWidth)*scale+.5))),
		MinOuterHeight: min(workHeight, max(nonClientHeight+1, int(float64(minWindowHeight)*scale+.5))),
	}
}

func contentScale(clientWidth, clientHeight int, ceiling float64) float64 {
	widthFit := float64(max(1, clientWidth)) / baseClientWidth
	heightFit := float64(max(1, clientHeight)) / baseClientHeight
	return math.Min(ceiling, math.Min(widthFit, heightFit))
}
