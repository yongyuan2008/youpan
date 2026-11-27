package main

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

const (
	desktopWindowWidth  = 320
	desktopWindowHeight = 244
	desktopWidgetClass  = "YoupanDesktopCountdown"
)

type monitorInfo struct {
	Size          uint32
	Monitor, Work rect
	Flags         uint32
}

var (
	pForegroundWindow  = u32.NewProc("GetForegroundWindow")
	pShellWindow       = u32.NewProc("GetShellWindow")
	pWindowProcess     = u32.NewProc("GetWindowThreadProcessId")
	pWindowClass       = u32.NewProc("GetClassNameW")
	pMonitorFromRect   = u32.NewProc("MonitorFromRect")
	pMonitorInfo       = u32.NewProc("GetMonitorInfoW")
	desktopWidget      uintptr
	desktopClassReady  bool
	desktopLayerActive bool
	desktopPaintedDay  time.Time
	desktopFonts       [3]uintptr
)

func desktopPx(value int) int32 {
	return int32(float64(value)*systemScale + .5)
}

func showDesktopWidget() error {
	if desktopWidget != 0 {
		return nil
	}
	instance, _, _ := pGetModuleHandle.Call(0)
	className := w(desktopWidgetClass)
	if !desktopClassReady {
		cursor, _, _ := pLoadCursor.Call(0, 32646) // IDC_SIZEALL
		class := wndClassEx{
			Size: uint32(unsafe.Sizeof(wndClassEx{})), Instance: instance,
			ClassName: className, Cursor: cursor, WndProc: syscall.NewCallback(desktopProc),
		}
		if registered, _, err := pRegisterClass.Call(uintptr(unsafe.Pointer(&class))); registered == 0 {
			return fmt.Errorf("无法注册桌面组件：%w", err)
		}
		desktopClassReady = true
	}

	width, height := desktopPx(desktopWindowWidth), desktopPx(desktopWindowHeight)
	position := widgetOrigin(appearance.DesktopPosition, width, height)
	const exStyle = wsExLayered | 0x00000080 | 0x08000000 // WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE
	hwnd, _, err := pCreateWindow.Call(exStyle, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(w("有盼 · 桌面倒计时"))),
		0x80000000, uintptr(position.X), uintptr(position.Y), uintptr(width), uintptr(height), 0, 0, instance, 0)
	if hwnd == 0 {
		return fmt.Errorf("无法打开桌面组件：%w", err)
	}
	desktopWidget = hwnd
	for i, spec := range [3]struct{ size, weight int }{{15, 400}, {22, 600}, {18, 600}} {
		desktopFonts[i] = createFontHandle(int(desktopPx(spec.size)), spec.weight, "Microsoft YaHei UI")
	}
	if !applyDesktopOpacity(previewOpacity) {
		hideDesktopWidget()
		return fmt.Errorf("无法设置桌面组件透明度")
	}
	pShowWindow.Call(hwnd, 4) // SW_SHOWNOACTIVATE
	orderDesktopWidget(true)
	if timer, _, err := pSetTimer.Call(hwnd, 1, 1000, 0); timer == 0 {
		hideDesktopWidget()
		return fmt.Errorf("无法更新桌面组件：%w", err)
	}
	return nil
}

func hideDesktopWidget() {
	if desktopWidget != 0 {
		pDestroyWindow.Call(desktopWidget)
	}
}

func applyDesktopOpacity(value float64) bool {
	if desktopWidget == 0 {
		return true
	}
	ok, _, _ := pSetLayeredAlpha.Call(desktopWidget, 0, uintptr(value*255+.5), 0x2)
	return ok != 0
}

func changeDesktopPin(on bool) error {
	if on {
		if err := showDesktopWidget(); err != nil {
			return err
		}
	}
	if err := appearance.SetDesktopPinned(on); err != nil {
		if on && !appearance.DesktopPinned {
			hideDesktopWidget()
		}
		return err
	}
	if !on {
		hideDesktopWidget()
	}
	return nil
}

func widgetOrigin(saved *DesktopPosition, width, height int32) point {
	work := currentWorkArea()
	x, y := work.Right-width-desktopPx(24), work.Top+desktopPx(24)
	if saved != nil {
		x, y = saved.X, saved.Y
	}
	probe := rect{x, y, x + width, y + height}
	monitor, _, _ := pMonitorFromRect.Call(uintptr(unsafe.Pointer(&probe)), 2) // MONITOR_DEFAULTTONEAREST
	info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if monitor != 0 {
		if ok, _, _ := pMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info))); ok != 0 {
			work = info.Work
		}
	}
	return point{
		int32(max(int(work.Left), min(int(x), int(work.Right-width)))),
		int32(max(int(work.Top), min(int(y), int(work.Bottom-height)))),
	}
}

func desktopHasFocus() bool {
	shell, _, _ := pShellWindow.Call()
	front, _, _ := pForegroundWindow.Call()
	if shell == 0 || front == 0 {
		return false
	}
	if front == shell {
		return true
	}
	var class [32]uint16
	pWindowClass.Call(front, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
	name := syscall.UTF16ToString(class[:])
	if name != "WorkerW" && name != "Progman" {
		return false
	}
	var shellPID, frontPID uint32
	pWindowProcess.Call(shell, uintptr(unsafe.Pointer(&shellPID)))
	pWindowProcess.Call(front, uintptr(unsafe.Pointer(&frontPID)))
	return shellPID != 0 && frontPID == shellPID
}

func orderDesktopWidget(force bool) {
	active := desktopHasFocus()
	if force || desktopLayerActive != active {
		order := uintptr(1) // HWND_BOTTOM, behind ordinary app windows
		if active {
			order = 0 // HWND_TOP, above the active desktop without making it topmost
		}
		if ok, _, _ := pSetWindowPos.Call(desktopWidget, order, 0, 0, 0, 0, 0x1|0x2|0x10); ok != 0 {
			desktopLayerActive = active
		}
	}
}

func refreshDesktopWidget() {
	if desktopWidget != 0 {
		pInvalidateRect.Call(desktopWidget, 0, 0)
	}
}

func desktopProc(hwnd uintptr, messageID uint32, wp, lp uintptr) uintptr {
	switch messageID {
	case 0x0021: // WM_MOUSEACTIVATE
		return 3 // MA_NOACTIVATE
	case 0x0084: // WM_NCHITTEST
		result, _, _ := pDefWindowProc.Call(hwnd, uintptr(messageID), wp, lp)
		if result == 1 { // HTCLIENT
			return 2 // HTCAPTION, keep native dragging
		}
		return result
	case wmExitSizeMove:
		var bounds rect
		if ok, _, _ := pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&bounds))); ok != 0 {
			if err := appearance.SetDesktopPosition(bounds.Left, bounds.Top); err != nil {
				message(mainWnd, "无法保存桌面组件位置："+err.Error(), 0x10)
			}
		}
		orderDesktopWidget(true)
		return 0
	case 0x007e: // WM_DISPLAYCHANGE
		origin := widgetOrigin(appearance.DesktopPosition, desktopPx(desktopWindowWidth), desktopPx(desktopWindowHeight))
		pSetWindowPos.Call(hwnd, 0, uintptr(origin.X), uintptr(origin.Y), 0, 0, 0x1|0x4|0x10)
		return 0
	case wmTimer:
		orderDesktopWidget(false)
		if desktopPaintedDay != civil(time.Now()) {
			refreshDesktopWidget()
		}
		return 0
	case 0x000f: // WM_PAINT
		paintDesktopWidget(hwnd)
		return 0
	case wmDestroy:
		if hwnd == desktopWidget {
			desktopWidget = 0
		}
		for i, handle := range desktopFonts {
			if handle != 0 {
				pDeleteObject.Call(handle)
			}
			desktopFonts[i] = 0
		}
		return 0
	}
	result, _, _ := pDefWindowProc.Call(hwnd, uintptr(messageID), wp, lp)
	return result
}

func paintDesktopWidget(hwnd uintptr) {
	var paint paintStruct
	dc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))
	defer pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))
	if dc == 0 {
		return
	}
	saved, _, _ := pSaveDC.Call(dc)
	if saved == 0 {
		return
	}
	defer pRestoreDC.Call(dc, saved)
	var bounds rect
	pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&bounds)))
	fillColor(dc, bounds, rgb(247, 249, 253))
	row := func(left, top, right, bottom int) rect {
		return rect{desktopPx(left), desktopPx(top), desktopPx(right), desktopPx(bottom)}
	}
	surfaceText(dc, row(18, 12, 302, 42), "倒计时", desktopFonts[1], textInk, 0)
	now := time.Now()
	for i, kind := range Events {
		y := 50 + i*36
		name, customized := customName(kind)
		value := describeCountdown(kind, state, now, name, customized)
		metric := value.Metric
		if value.Date == "" || value.Caption == "今天" {
			metric = value.Caption
		}
		if i == 0 {
			fillColor(dc, row(12, y-1, 308, y+31), rgb(226, 236, 251))
		}
		surfaceText(dc, row(19, y, 207, y+30), name, desktopFonts[0], textInk, 0)
		surfaceText(dc, row(209, y, 298, y+30), metric, desktopFonts[2], accentInk, 2)
	}
	desktopPaintedDay = civil(now)
}
