package main

import (
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"
)

func TestDesktopWidgetVisibilityOpacityAndPosition(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	oldAppearance, oldOpacity := appearance, previewOpacity
	defer func() {
		hideDesktopWidget()
		appearance, previewOpacity = oldAppearance, oldOpacity
	}()
	path := filepath.Join(t.TempDir(), "appearance.json")
	var err error
	appearance, err = loadAppearancePath(path)
	if err != nil {
		t.Fatal(err)
	}
	previewOpacity = .68
	if err = changeDesktopPin(true); err != nil {
		t.Fatal(err)
	}
	isVisible := u32.NewProc("IsWindowVisible")
	if visible, _, _ := isVisible.Call(desktopWidget); visible == 0 {
		t.Fatal("pinned desktop widget is not visible")
	}
	getAlpha := u32.NewProc("GetLayeredWindowAttributes")
	var alpha byte
	if ok, _, err := getAlpha.Call(desktopWidget, 0, uintptr(unsafe.Pointer(&alpha)), 0); ok == 0 || alpha != 173 {
		t.Fatalf("widget transparency = %d, %v", alpha, err)
	}

	hwnd := desktopWidget
	var bounds rect
	if ok, _, err := pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&bounds))); ok == 0 {
		t.Fatalf("read widget bounds: %v", err)
	}
	pSendMessage.Call(hwnd, 0x0232, 0, 0) // WM_EXITSIZEMOVE
	reloaded, err := loadAppearancePath(path)
	if err != nil || !reloaded.DesktopPinned || reloaded.DesktopPosition == nil ||
		*reloaded.DesktopPosition != (DesktopPosition{X: bounds.Left, Y: bounds.Top}) {
		t.Fatalf("persisted widget position = %+v, %v", reloaded, err)
	}
	if err = changeDesktopPin(false); err != nil {
		t.Fatal(err)
	}
	if exists, _, _ := u32.NewProc("IsWindow").Call(hwnd); exists != 0 {
		t.Fatal("disabled widget was left open")
	}
	reloaded, err = loadAppearancePath(path)
	if err != nil || reloaded.DesktopPinned {
		t.Fatalf("desktop pin stayed enabled: %+v, %v", reloaded, err)
	}
}
