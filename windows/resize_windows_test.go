package main

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

type minMaxInfo struct {
	reserved, maxSize, maxPosition, minTrackSize, maxTrackSize point
}

func TestHiddenWindowShrinksAndKeepsPersistenceScale(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	oldMain, oldScale, oldWindowScale := mainWnd, scale, windowScale
	oldAppearance, oldReady, oldPending, oldResizing := appearance, windowReady, sizeSavePending, resizingWindow
	oldMinimumWidth, oldMinimumHeight := minOuterWidth, minOuterHeight
	oldLayouts, oldHeights := controlLayouts, fontPixelHeights
	oldFonts := [7]uintptr{normalFont, headingFont, smallFont, boldFont, heroFont, metricFont, symbolFont}
	mainWnd = 0
	scale, windowScale = 1.25, 1.25
	minOuterWidth, minOuterHeight = 700, 638
	controlLayouts = map[uintptr]controlLayout{}
	fontPixelHeights = [7]int{}
	normalFont, headingFont, smallFont, boldFont, heroFont, metricFont, symbolFont = 0, 0, 0, 0, 0, 0, 0
	updateFontsForScale()

	instance, _, _ := pGetModuleHandle.Call(0)
	className := w("YoupanResizeTest")
	class := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   syscall.NewCallback(wndProc),
		Instance:  instance,
		ClassName: className,
	}
	if result, _, err := pRegisterClass.Call(uintptr(unsafe.Pointer(&class))); result == 0 {
		t.Fatalf("register test window: %v", err)
	}
	unregisterClass := u32.NewProc("UnregisterClassW")
	peekMessage := u32.NewProc("PeekMessageW")
	defer func() {
		if mainWnd != 0 {
			pDestroyWindow.Call(mainWnd)
		}
		var queued msg
		peekMessage.Call(uintptr(unsafe.Pointer(&queued)), 0, 0x0012, 0x0012, 1)
		unregisterClass.Call(uintptr(unsafe.Pointer(className)), instance)
		for _, h := range [7]uintptr{normalFont, headingFont, smallFont, boldFont, heroFont, metricFont, symbolFont} {
			if h != 0 {
				pDeleteObject.Call(h)
			}
		}
		mainWnd, scale, windowScale = oldMain, oldScale, oldWindowScale
		appearance, windowReady, sizeSavePending, resizingWindow = oldAppearance, oldReady, oldPending, oldResizing
		minOuterWidth, minOuterHeight = oldMinimumWidth, oldMinimumHeight
		controlLayouts, fontPixelHeights = oldLayouts, oldHeights
		normalFont, headingFont, smallFont, boldFont = oldFonts[0], oldFonts[1], oldFonts[2], oldFonts[3]
		heroFont, metricFont, symbolFont = oldFonts[4], oldFonts[5], oldFonts[6]
	}()

	hwnd, _, err := pCreateWindow.Call(0, uintptr(unsafe.Pointer(className)), 0, mainWindowStyle(), 20, 20, 950, 875, 0, 0, instance, 0)
	if hwnd == 0 {
		t.Fatalf("create hidden test window: %v", err)
	}
	mainWnd = hwnd
	getWindowLong := u32.NewProc("GetWindowLongPtrW")
	style, _, _ := getWindowLong.Call(hwnd, ^uintptr(15)) // GWL_STYLE (-16)
	wantStyle := uintptr(wsMinimizeBox | wsMaximizeBox | wsThickFrame)
	if style&wantStyle != wantStyle {
		t.Fatalf("resizable/minimize/maximize style missing: %#x", style)
	}
	child := create("STATIC", "resize", 0, 0, 166, 100, 552, 34, 990)
	if child == 0 {
		t.Fatal("create child control")
	}
	font(child, smallFont)
	oldSmallFont := smallFont

	var limits minMaxInfo
	pSendMessage.Call(hwnd, wmGetMinMaxInfo, 0, uintptr(unsafe.Pointer(&limits)))
	if limits.minTrackSize.X != 700 || limits.minTrackSize.Y != 638 {
		t.Fatalf("minimum track size=%+v", limits.minTrackSize)
	}

	const swpNoMoveNoZOrderNoActivate = 0x2 | 0x4 | 0x10
	path := filepath.Join(t.TempDir(), "appearance.json")
	appearance, err = loadAppearancePath(path)
	if err != nil {
		t.Fatal(err)
	}
	windowReady = true
	pSendMessage.Call(hwnd, wmEnterSizeMove, 0, 0)
	if ok, _, err := pSetWindowPos.Call(hwnd, 0, 0, 0, 400, 300, swpNoMoveNoZOrderNoActivate); ok == 0 {
		t.Fatalf("shrink hidden window: %v", err)
	}
	var outer, client, childRect rect
	if ok, _, err := pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&outer))); ok == 0 {
		t.Fatalf("get outer rect: %v", err)
	}
	if ok, _, err := pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client))); ok == 0 {
		t.Fatalf("get client rect: %v", err)
	}
	if ok, _, err := pGetWindowRect.Call(child, uintptr(unsafe.Pointer(&childRect))); ok == 0 {
		t.Fatalf("get child rect: %v", err)
	}
	outerWidth, outerHeight := int(outer.Right-outer.Left), int(outer.Bottom-outer.Top)
	if outerWidth < 700 || outerHeight < 638 {
		t.Fatalf("window crossed minimum: outer=%dx%d", outerWidth, outerHeight)
	}
	clientWidth, clientHeight := int(client.Right-client.Left), int(client.Bottom-client.Top)
	wantScale := contentScale(clientWidth, clientHeight, windowScale)
	if math.Abs(scale-wantScale) > 1e-9 || scale >= windowScale {
		t.Fatalf("content scale=%v want=%v windowScale=%v", scale, wantScale, windowScale)
	}
	wantChildWidth := int(float64(552)*scale + .5)
	if got := int(childRect.Right - childRect.Left); got != wantChildWidth {
		t.Fatalf("child width=%d want=%d at scale %.3f", got, wantChildWidth, scale)
	}
	currentFont, _, _ := pSendMessage.Call(child, 0x0031, 0, 0) // WM_GETFONT
	if currentFont != smallFont || smallFont == oldSmallFont {
		t.Fatalf("font role was not rescaled: current=%#x small=%#x old=%#x", currentFont, smallFont, oldSmallFont)
	}
	getObject := g32.NewProc("GetObjectW")
	if size, _, _ := getObject.Call(oldSmallFont, 0, 0); size != 0 {
		t.Fatalf("old font handle was not released: %#x", oldSmallFont)
	}
	if got := windowLogical(int32(outerWidth)); got != int(float64(outerWidth)/1.25+.5) || got < minWindowWidth {
		t.Fatalf("saved logical width=%d outer=%d contentScale=%.3f", got, outerWidth, scale)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("saved before the drag ended: %v", err)
	}
	pSendMessage.Call(hwnd, wmExitSizeMove, 0, 0)
	reloaded, err := loadAppearancePath(path)
	if err != nil || reloaded.Width != windowLogical(int32(outerWidth)) || reloaded.Height != windowLogical(int32(outerHeight)) {
		t.Fatalf("drag size did not persist: %dx%d, %v", reloaded.Width, reloaded.Height, err)
	}
}
