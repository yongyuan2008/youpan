package main

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

func TestExitPostsQuit(t *testing.T) {
	const (
		hwndMessage = ^uintptr(2)
		wmQuit      = 0x0012
		pmRemove    = 1
	)
	isWindow := u32.NewProc("IsWindow")
	peekMessage := u32.NewProc("PeekMessageW")
	unregisterClass := u32.NewProc("UnregisterClassW")
	tests := []struct {
		name    string
		message uint32
		command uintptr
	}{
		{"Close", wmClose, 0},
		{"TrayQuit", wmCommand, idTrayQuit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			instance, _, _ := pGetModuleHandle.Call(0)
			className := w("YoupanExitTest" + tt.name)
			class := wndClassEx{
				Size:      uint32(unsafe.Sizeof(wndClassEx{})),
				WndProc:   syscall.NewCallback(wndProc),
				Instance:  instance,
				ClassName: className,
			}
			if result, _, err := pRegisterClass.Call(uintptr(unsafe.Pointer(&class))); result == 0 {
				t.Fatalf("register test window: %v", err)
			}
			defer func() {
				if ok, _, err := unregisterClass.Call(uintptr(unsafe.Pointer(className)), instance); ok == 0 {
					t.Errorf("unregister test window: %v", err)
				}
			}()
			hwnd, _, err := pCreateWindow.Call(0, uintptr(unsafe.Pointer(className)), 0, 0, 0, 0, 0, 0, hwndMessage, 0, instance, 0)
			if hwnd == 0 {
				t.Fatalf("create message-only window: %v", err)
			}
			mainWnd = hwnd
			defer func() {
				if exists, _, _ := isWindow.Call(hwnd); exists != 0 {
					pDestroyWindow.Call(hwnd)
				}
				var queued msg
				peekMessage.Call(uintptr(unsafe.Pointer(&queued)), 0, wmQuit, wmQuit, pmRemove)
				mainWnd = 0
				exiting = false
			}()

			pSendMessage.Call(hwnd, uintptr(tt.message), tt.command, 0)
			if exists, _, _ := isWindow.Call(hwnd); exists != 0 {
				t.Error("exit left the main window alive")
			}
			var queued msg
			if ok, _, _ := peekMessage.Call(uintptr(unsafe.Pointer(&queued)), 0, wmQuit, wmQuit, pmRemove); ok == 0 || queued.Message != wmQuit {
				t.Error("exit did not post WM_QUIT to end the message loop")
			}
		})
	}
}
