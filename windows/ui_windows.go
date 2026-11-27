//go:build windows

package main

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	u32                   = syscall.NewLazyDLL("user32.dll")
	k32                   = syscall.NewLazyDLL("kernel32.dll")
	g32                   = syscall.NewLazyDLL("gdi32.dll")
	sh32                  = syscall.NewLazyDLL("shell32.dll")
	cc32                  = syscall.NewLazyDLL("comctl32.dll")
	ntdll                 = syscall.NewLazyDLL("ntdll.dll")
	pRegisterClass        = u32.NewProc("RegisterClassExW")
	pCreateWindow         = u32.NewProc("CreateWindowExW")
	pDefWindowProc        = u32.NewProc("DefWindowProcW")
	pShowWindow           = u32.NewProc("ShowWindow")
	pUpdateWindow         = u32.NewProc("UpdateWindow")
	pGetMessage           = u32.NewProc("GetMessageW")
	pTranslateMessage     = u32.NewProc("TranslateMessage")
	pDispatchMessage      = u32.NewProc("DispatchMessageW")
	pIsDialogMessage      = u32.NewProc("IsDialogMessageW")
	pPostQuit             = u32.NewProc("PostQuitMessage")
	pSendMessage          = u32.NewProc("SendMessageW")
	pSetWindowText        = u32.NewProc("SetWindowTextW")
	pGetWindowText        = u32.NewProc("GetWindowTextW")
	pGetWindowTextLength  = u32.NewProc("GetWindowTextLengthW")
	pSetWindowPos         = u32.NewProc("SetWindowPos")
	pSetTimer             = u32.NewProc("SetTimer")
	pKillTimer            = u32.NewProc("KillTimer")
	pMessageBox           = u32.NewProc("MessageBoxW")
	pLoadCursor           = u32.NewProc("LoadCursorW")
	pLoadIcon             = u32.NewProc("LoadIconW")
	pGetSystemMetrics     = u32.NewProc("GetSystemMetrics")
	pSystemParametersInfo = u32.NewProc("SystemParametersInfoW")
	pGetClientRect        = u32.NewProc("GetClientRect")
	pGetWindowRect        = u32.NewProc("GetWindowRect")
	pAdjustWindowRectEx   = u32.NewProc("AdjustWindowRectEx")
	pGetDlgItem           = u32.NewProc("GetDlgItem")
	pMoveWindow           = u32.NewProc("MoveWindow")
	pIsZoomed             = u32.NewProc("IsZoomed")
	pIsIconic             = u32.NewProc("IsIconic")
	pSetForegroundWindow  = u32.NewProc("SetForegroundWindow")
	pFindWindow           = u32.NewProc("FindWindowW")
	pSetFocus             = u32.NewProc("SetFocus")
	pDestroyWindow        = u32.NewProc("DestroyWindow")
	pSetProcessDPIAware   = u32.NewProc("SetProcessDPIAware")
	pGetDpiForSystem      = u32.NewProc("GetDpiForSystem")
	pBeginPaint           = u32.NewProc("BeginPaint")
	pEndPaint             = u32.NewProc("EndPaint")
	pFillRect             = u32.NewProc("FillRect")
	pInvalidateRect       = u32.NewProc("InvalidateRect")
	pSetLayeredAlpha      = u32.NewProc("SetLayeredWindowAttributes")
	pCreatePopupMenu      = u32.NewProc("CreatePopupMenu")
	pAppendMenu           = u32.NewProc("AppendMenuW")
	pTrackPopupMenu       = u32.NewProc("TrackPopupMenu")
	pDestroyMenu          = u32.NewProc("DestroyMenu")
	pGetCursorPos         = u32.NewProc("GetCursorPos")
	pCreateSolidBrush     = g32.NewProc("CreateSolidBrush")
	pDeleteObject         = g32.NewProc("DeleteObject")
	pSetTextColor         = g32.NewProc("SetTextColor")
	pSetBkColor           = g32.NewProc("SetBkColor")
	pCreateFont           = g32.NewProc("CreateFontW")
	pGetModuleHandle      = k32.NewProc("GetModuleHandleW")
	pCreateMutex          = k32.NewProc("CreateMutexW")
	pCloseHandle          = k32.NewProc("CloseHandle")
	pCopyMemory           = ntdll.NewProc("RtlMoveMemory")
	pShellNotifyIcon      = sh32.NewProc("Shell_NotifyIconW")
	pInitCommonControls   = cc32.NewProc("InitCommonControlsEx")
)

const (
	wsChild            = 0x40000000
	wsVisible          = 0x10000000
	wsTabstop          = 0x00010000
	wsBorder           = 0x00800000
	wsVScroll          = 0x00200000
	wsCaption          = 0x00C00000
	wsSysmenu          = 0x00080000
	wsMinimizeBox      = 0x00020000
	wsMaximizeBox      = 0x00010000
	wsThickFrame       = 0x00040000
	wsExLayered        = 0x00080000
	bnDoubleClicked    = 5
	wmDestroy          = 0x0002
	wmSize             = 0x0005
	wmClose            = 0x0010
	wmGetMinMaxInfo    = 0x0024
	wmDrawItem         = 0x002b
	wmCommand          = 0x0111
	wmHScroll          = 0x0114
	wmTimer            = 0x0113
	wmSetFont          = 0x0030
	wmTray             = 0x8001
	wmLButtonUp        = 0x0202
	wmRButtonUp        = 0x0205
	bmGetCheck         = 0x00f0
	bmSetCheck         = 0x00f1
	lbAddString        = 0x0180
	lbReset            = 0x0184
	lbGetCurSel        = 0x0188
	lbSetCurSel        = 0x0186
	cbAddString        = 0x0143
	cbSetCurSel        = 0x014e
	cbGetCurSel        = 0x0147
	swHide             = 0
	swShow             = 5
	swRestore          = 9
	idNavCountdown     = 100
	idNavToday         = 101
	idNavCalendar      = 102
	idNavNotes         = 103
	idPin              = 110
	idSettings         = 111
	idCountdownBase    = 200
	idDateStart        = 220
	idDateEnd          = 221
	idDateSave         = 222
	idDateCancel       = 223
	idDateReset        = 224
	idQuickTitle       = 300
	idQuickAdd         = 301
	idTodayList        = 302
	idTodayDone        = 303
	idTodayEdit        = 304
	idTodayDelete      = 305
	idShowCompleted    = 306
	idTestReminder     = 307
	idTodaySearch      = 308
	idModeToday        = 309
	idModeFuture       = 310
	idTaskTitle        = 330
	idTaskDetails      = 331
	idTaskDate         = 332
	idTaskReminder     = 333
	idTaskRepeat       = 334
	idTaskPriority     = 335
	idTaskSave         = 336
	idTaskCancel       = 337
	idCalPrev          = 400
	idCalNext          = 401
	idCalToday         = 402
	idCalDayBase       = 420
	idCalList          = 470
	idCalAdd           = 471
	idCalDone          = 472
	idCalEdit          = 473
	idNoteSearch       = 500
	idNoteList         = 501
	idNoteTitle        = 502
	idNoteBody         = 503
	idNoteNew          = 504
	idNoteSave         = 505
	idNoteDelete       = 506
	idNoteMakeTask     = 507
	idTrayOpen         = 600
	idTrayQuit         = 601
	idOpacitySlider    = 703
	idNameBase         = 710
	idNamesSave        = 720
	idNamesReset       = 721
	idVisualReset      = 722
	idSettingsBack     = 723
	idTimerWindowSave  = 2
	idTimerOpacitySave = 3
)

type rect struct{ Left, Top, Right, Bottom int32 }
type paintStruct struct {
	Hdc                uintptr
	Erase              int32
	Paint              rect
	Restore, IncUpdate int32
	Reserved           [32]byte
}
type point struct{ X, Y int32 }
type initCommonControls struct {
	Size, Classes uint32
}
type msg struct {
	Hwnd           uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             point
	LPrivate       uint32
}
type wndClassEx struct {
	Size, Style                        uint32
	WndProc                            uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	IconSmall                          uintptr
}
type notifyIconData struct {
	CbSize                        uint32
	HWnd                          uintptr
	UID, UFlags, UCallbackMessage uint32
	HIcon                         uintptr
	SzTip                         [128]uint16
	DwState, DwStateMask          uint32
	SzInfo                        [256]uint16
	UTimeoutOrVersion             uint32
	SzInfoTitle                   [64]uint16
	DwInfoFlags                   uint32
	GuidItem                      [16]byte
	HBalloonIcon                  uintptr
}

var (
	state                                                                                State
	planner                                                                              Planner
	appearance                                                                           Appearance
	mainWnd, appIcon                                                                     uintptr
	scale, systemScale                                                                   = 1.0, 1.0
	workArea                                                                             rect
	minOuterWidth, minOuterHeight, nonClientWidth, nonClientHeight                       int
	normalFont, headingFont, smallFont, boldFont, bgBrush, sideBrush                     uintptr
	heroFont, metricFont, symbolFont, pinButton, settingsButton, countdownDate           uintptr
	view                                                                                 int
	viewControls                                                                         [4][]uintptr
	navControls                                                                          []uintptr
	countdownButtons                                                                     [5]uintptr
	cardStatus                                                                           [5]countdownStatus
	dateEditor                                                                           []uintptr
	dateStart, dateEnd, dateEndLabel, dateContext, dateReset                             uintptr
	dateEditing                                                                          Kind
	dateEditorOpen                                                                       bool
	quickTitle, todayList, showCompleted, todaySearch, modeTodayButton, modeFutureButton uintptr
	todayRefs                                                                            []TaskOccurrence
	todayMode                                                                            int
	taskEditor                                                                           []uintptr
	taskTitle, taskDetails, taskDate, taskReminder, taskRepeat, taskPriority             uintptr
	editingTaskID                                                                        string
	taskEditorOpen                                                                       bool
	calendarMonth, calendarSelected                                                      time.Time
	calendarLabel, calendarSelectedLabel, calendarList                                   uintptr
	calendarDays                                                                         [42]uintptr
	calendarWeekLabels                                                                   [7]uintptr
	calendarDaysPaint                                                                    [42]calendarDayPaint
	lastCalendarClick                                                                    int
	lastCalendarClickDate                                                                string
	calendarRefs                                                                         []TaskOccurrence
	noteSearch, noteList, noteTitle, noteBody                                            uintptr
	noteRefs                                                                             []string
	editingNoteID, noteSavedTitle, noteSavedBody                                         string
	settingsControls                                                                     []uintptr
	nameEdits                                                                            [5]uintptr
	opacitySlider, opacityLabel, todayHint, noteHint                                     uintptr
	settingsStatus                                                                       uintptr
	settingsOpen, windowReady, sizeSavePending, opacitySavePending                       bool
	previewOpacity                                                                       float64
	trayAdded, trayHintShown, exiting                                                    bool
)

func w(s string) *uint16          { p, _ := syscall.UTF16PtrFromString(s); return p }
func px(v int) uintptr            { return uintptr(int(float64(v)*scale + .5)) }
func setText(h uintptr, s string) { pSetWindowText.Call(h, uintptr(unsafe.Pointer(w(s)))) }
func getText(h uintptr) string {
	n, _, _ := pGetWindowTextLength.Call(h)
	b := make([]uint16, int(n)+1)
	if len(b) > 0 {
		pGetWindowText.Call(h, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	}
	return syscall.UTF16ToString(b)
}
func copy16(dst []uint16, s string) {
	src, _ := syscall.UTF16FromString(s)
	if len(src) > len(dst) {
		src = src[:len(dst)]
		src[len(src)-1] = 0
	}
	copy(dst, src)
}
func message(h uintptr, s string, flags uintptr) uintptr {
	r, _, _ := pMessageBox.Call(h, uintptr(unsafe.Pointer(w(s))), uintptr(unsafe.Pointer(w("有盼"))), flags)
	return r
}
func create(class, text string, style, ex uintptr, x, y, wid, hei, id int) uintptr {
	h, _, _ := pCreateWindow.Call(ex, uintptr(unsafe.Pointer(w(class))), uintptr(unsafe.Pointer(w(text))), style|wsChild|wsVisible, px(x), px(y), px(wid), px(hei), mainWnd, uintptr(id), 0, 0)
	pSendMessage.Call(h, wmSetFont, normalFont, 1)
	return h
}
func font(h, f uintptr) { pSendMessage.Call(h, wmSetFont, f, 1) }
func setVisible(a []uintptr, on bool) {
	c := uintptr(swHide)
	if on {
		c = swShow
	}
	for _, h := range a {
		pShowWindow.Call(h, c)
	}
}
func static(s string, x, y, wid, hei int, head bool) uintptr {
	h := create("STATIC", s, 0, 0, x, y, wid, hei, 0)
	if head {
		font(h, headingFont)
	}
	return h
}
func button(s string, x, y, wid, hei, id int) uintptr {
	return create("BUTTON", s, wsTabstop|0x8000, 0, x, y, wid, hei, id)
}
func edit(s string, x, y, wid, hei, id int, multi bool) uintptr {
	st := uintptr(wsTabstop | wsBorder | 0x0080)
	if multi {
		st = wsTabstop | wsBorder | wsVScroll | 0x0004 | 0x0040 | 0x1000
	}
	return create("EDIT", s, st, 0x200, x, y, wid, hei, id)
}
func listbox(x, y, wid, hei, id int) uintptr {
	return create("LISTBOX", "", wsTabstop|wsBorder|wsVScroll|0x0001|0x0040, 0x200, x, y, wid, hei, id)
}
func listReset(h uintptr) { pSendMessage.Call(h, lbReset, 0, 0) }
func listAdd(h uintptr, s string) {
	pSendMessage.Call(h, lbAddString, 0, uintptr(unsafe.Pointer(w(s))))
}
func listSel(h uintptr) int {
	r, _, _ := pSendMessage.Call(h, lbGetCurSel, 0, 0)
	if int32(r) < 0 {
		return -1
	}
	return int(r)
}
func selectedRef(h uintptr, refs []TaskOccurrence) (TaskOccurrence, bool) {
	i := listSel(h)
	if i < 0 || i >= len(refs) {
		return TaskOccurrence{}, false
	}
	return refs[i], true
}
func taskByID(id string) (Task, bool) {
	for _, t := range planner.Tasks {
		if t.ID == id {
			return t, true
		}
	}
	return Task{}, false
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func logical(v int32) int { return int(float64(v)/scale + .5) }
func moveControl(h uintptr, x, y, wid, hei int) {
	if h != 0 {
		pMoveWindow.Call(h, px(x), px(y), px(wid), px(hei), 1)
	}
}
func moveID(id, x, y, wid, hei int) {
	h, _, _ := pGetDlgItem.Call(mainWnd, uintptr(id))
	moveControl(h, x, y, wid, hei)
}
func customName(k Kind) (string, bool) { return appearance.Name(k) }
func mainWindowStyle() uintptr {
	return wsCaption | wsSysmenu | wsMinimizeBox | wsMaximizeBox | wsThickFrame
}
func currentWorkArea() rect {
	work := rect{}
	if ok, _, _ := pSystemParametersInfo.Call(0x30, 0, uintptr(unsafe.Pointer(&work)), 0); ok != 0 && work.Right > work.Left && work.Bottom > work.Top {
		return work
	}
	sw, _, _ := pGetSystemMetrics.Call(0)
	sh, _, _ := pGetSystemMetrics.Call(1)
	return rect{0, 0, int32(max(1, int(sw))), int32(max(1, int(sh)))}
}
func configureWindowGeometry() {
	workArea = currentWorkArea()
	// The 96-DPI frame/caption delta of the existing 760x700 outer window
	// and 744x661 client layout is 16x39. Keep that as a conservative fallback
	// and lower bound if AdjustWindowRectEx is unavailable or rounds down.
	nonClientWidth = int(float64(defaultWindowWidth-baseClientWidth)*systemScale + .5)
	nonClientHeight = int(float64(defaultWindowHeight-baseClientHeight)*systemScale + .5)
	probe := rect{0, 0, 100, 100}
	if ok, _, _ := pAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&probe)), mainWindowStyle(), 0, wsExLayered); ok != 0 {
		nonClientWidth = max(nonClientWidth, int(probe.Right-probe.Left)-100)
		nonClientHeight = max(nonClientHeight, int(probe.Bottom-probe.Top)-100)
	}
	fit := fitGeometry(int(workArea.Right-workArea.Left), int(workArea.Bottom-workArea.Top), systemScale, nonClientWidth, nonClientHeight)
	scale = fit.Scale
	minOuterWidth, minOuterHeight = fit.MinOuterWidth, fit.MinOuterHeight
}
func windowOuterForLogical(width, height int) (int, int) {
	workWidth, workHeight := int(workArea.Right-workArea.Left), int(workArea.Bottom-workArea.Top)
	physicalWidth := max(minOuterWidth, int(float64(width)*scale+.5))
	physicalHeight := max(minOuterHeight, int(float64(height)*scale+.5))
	return min(workWidth, physicalWidth), min(workHeight, physicalHeight)
}
func applyOpacity(value float64) {
	if value < minOpacity {
		value = minOpacity
	}
	if value > maxOpacity {
		value = maxOpacity
	}
	previewOpacity = value
	alpha := uintptr(value*255 + .5)
	pSetLayeredAlpha.Call(mainWnd, 0, alpha, 0x2)
	if opacityLabel != 0 {
		setText(opacityLabel, fmt.Sprintf("透明度：%d%%", int(value*100+.5)))
	}
}
func saveCurrentWindowSize() {
	if !windowReady || mainWnd == 0 {
		return
	}
	zoomed, _, _ := pIsZoomed.Call(mainWnd)
	iconic, _, _ := pIsIconic.Call(mainWnd)
	if zoomed != 0 || iconic != 0 {
		return
	}
	var r rect
	if ok, _, _ := pGetWindowRect.Call(mainWnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return
	}
	wid, hei := logical(r.Right-r.Left), logical(r.Bottom-r.Top)
	if wid == appearance.Width && hei == appearance.Height {
		return
	}
	if err := appearance.SetWindow(wid, hei); err != nil {
		message(mainWnd, "无法保存窗口大小："+err.Error(), 0x10)
	}
}
func savePreviewOpacity() {
	if !opacitySavePending {
		return
	}
	opacitySavePending = false
	if err := appearance.SetOpacity(previewOpacity); err != nil {
		message(mainWnd, "无法保存透明度："+err.Error(), 0x10)
	}
}

func layoutControls(clientWidth, clientHeight int) {
	// These are the client dimensions of the original 760x700 window at 96
	// DPI. The minimum outer size keeps every original control visible.
	dw := max(0, clientWidth-744)
	dh := max(0, clientHeight-661)
	moveControl(settingsButton, 12, clientHeight-52, 118, 38)
	moveControl(pinButton, 16, clientHeight-88, 112, 25)
	moveControl(countdownButtons[0], 166, 94, 552+dw, 124)
	for i := 1; i < len(countdownButtons); i++ {
		moveControl(countdownButtons[i], 166, 262+(i-1)*71, 552+dw, 62)
	}
	moveControl(quickTitle, 166, 100, 438+dw, 34)
	moveID(idQuickAdd, 614+dw, 100, 104, 34)
	// Today view: stretch the text fields and list; keep actions attached to
	// the bottom. No control is recreated, so selection and unsaved text stay.
	moveControl(todaySearch, 216, 145, 266+dw, 32)
	moveControl(modeTodayButton, 494+dw, 145, 94, 32)
	moveControl(modeFutureButton, 598+dw, 145, 120, 32)
	moveControl(todayList, 166, 216, 552+dw, 290+dh)
	moveID(idTodayDone, 166, 520+dh, 120, 36)
	moveID(idTodayEdit, 296, 520+dh, 86, 36)
	moveID(idTodayDelete, 392, 520+dh, 86, 36)
	moveID(idTestReminder, 488, 520+dh, 104, 36)
	moveControl(todayHint, 166, 571+dh, 540+dw, 25)

	// Calendar columns and the selected-day task list grow with the window.
	colExtra := dw / 7
	for i, h := range calendarWeekLabels {
		moveControl(h, 166+i*(79+colExtra), 68, 70+colExtra, 24)
	}
	for i, h := range calendarDays {
		moveControl(h, 166+(i%7)*(79+colExtra), 94+(i/7)*47, 70+colExtra, 40)
	}
	moveControl(calendarList, 166, 416, 552+dw, 132+dh)
	moveID(idCalAdd, 166, 562+dh, 145, 36)
	moveID(idCalDone, 321, 562+dh, 120, 36)
	moveID(idCalEdit, 451, 562+dh, 86, 36)

	// Notes benefit most from extra space. The left index remains stable while
	// the editor expands both horizontally and vertically.
	moveControl(noteSearch, 222, 69, 496+dw, 32)
	moveControl(noteList, 166, 112, 205, 398+dh)
	moveControl(noteTitle, 388, 112, 330+dw, 34)
	moveControl(noteBody, 388, 158, 330+dw, 352+dh)
	moveID(idNoteNew, 166, 524+dh, 76, 36)
	moveID(idNoteSave, 388, 524+dh, 100, 36)
	moveID(idNoteDelete, 498, 524+dh, 76, 36)
	moveID(idNoteMakeTask, 584+dw, 524+dh, 134, 36)
	moveControl(noteHint, 388, 574+dh, 330+dw, 25)
	pInvalidateRect.Call(mainWnd, 0, 1)
}
func noteDirty() bool {
	return editingNoteID != "" && (getText(noteTitle) != noteSavedTitle || getText(noteBody) != noteSavedBody) || editingNoteID == "" && (strings.TrimSpace(getText(noteTitle)) != "" || strings.TrimSpace(getText(noteBody)) != "")
}
func confirmDiscardNote() bool {
	if !noteDirty() {
		return true
	}
	if message(mainWnd, "笔记有未保存的修改，放弃这些修改？", 0x04|0x30) != 6 {
		return false
	}
	setText(noteTitle, noteSavedTitle)
	setText(noteBody, noteSavedBody)
	return true
}

func setTopmost(on bool) {
	ins := ^uintptr(1)
	if on {
		ins = ^uintptr(0)
	}
	pSetWindowPos.Call(mainWnd, ins, 0, 0, 0, 0, 0x1|0x2|0x10)
}
func showView(next int) {
	if dateEditorOpen || taskEditorOpen || next < 0 || next > 3 {
		return
	}
	if view == 3 && next != 3 && !confirmDiscardNote() {
		return
	}
	if settingsOpen {
		if !finishNamesEditing() {
			return
		}
		settingsOpen = false
		setVisible(settingsControls, false)
	}
	view = next
	for i := range viewControls {
		setVisible(viewControls[i], i == view)
	}
	refreshView()
	invalidateNavigation()
	pInvalidateRect.Call(mainWnd, 0, 1)
}
func openSettings() {
	if dateEditorOpen || taskEditorOpen || settingsOpen {
		return
	}
	if view == 3 && !confirmDiscardNote() {
		return
	}
	settingsOpen = true
	setText(settingsStatus, "")
	setVisible(viewControls[view], false)
	setVisible(settingsControls, true)
	for i, k := range Events {
		name, _ := customName(k)
		setText(nameEdits[i], name)
	}
	position := int(previewOpacity*100 + .5)
	pSendMessage.Call(opacitySlider, 0x405, 1, uintptr(position))
	applyOpacity(previewOpacity)
	invalidateNavigation()
	pInvalidateRect.Call(mainWnd, 0, 1)
}
func closeSettings() {
	if !settingsOpen || !finishNamesEditing() {
		return
	}
	savePreviewOpacity()
	settingsOpen = false
	setVisible(settingsControls, false)
	setVisible(viewControls[view], true)
	refreshView()
	invalidateNavigation()
	pInvalidateRect.Call(mainWnd, 0, 1)
}
func finishNamesEditing() bool {
	for i, k := range Events {
		name, _ := customName(k)
		if getText(nameEdits[i]) == name {
			continue
		}
		switch message(mainWnd, "名称有未保存的修改。要保存后离开吗？\n\n选择“否”将放弃这些修改。", 0x03|0x20) {
		case 6: // Yes
			return saveNames()
		case 7: // No
			return true
		default:
			return false
		}
	}
	return true
}
func saveNames() bool {
	names := make(map[Kind]string, len(Events))
	for i, k := range Events {
		name := strings.TrimSpace(getText(nameEdits[i]))
		if err := validateCustomName(name); err != nil {
			message(mainWnd, k.Title()+"名称无效："+err.Error(), 0x10)
			pSetFocus.Call(nameEdits[i])
			return false
		}
		names[k] = name
	}
	if err := appearance.SetNames(names); err != nil {
		message(mainWnd, "保存名称失败："+err.Error(), 0x10)
		return false
	}
	for i, k := range Events {
		setText(nameEdits[i], names[k])
	}
	setText(settingsStatus, "名称已保存")
	refreshCountdown()
	refreshCalendar()
	return true
}
func resetNames() {
	if err := appearance.ResetNames(); err != nil {
		message(mainWnd, "恢复默认名称失败："+err.Error(), 0x10)
		return
	}
	for i, k := range Events {
		setText(nameEdits[i], k.Title())
	}
	setText(settingsStatus, "已恢复默认名称")
	refreshCountdown()
	refreshCalendar()
}
func resetVisuals() {
	if err := appearance.ResetVisuals(); err != nil {
		message(mainWnd, "恢复默认外观失败："+err.Error(), 0x10)
		return
	}
	pKillTimer.Call(mainWnd, idTimerOpacitySave)
	opacitySavePending = false
	previewOpacity = appearance.Opacity
	applyOpacity(previewOpacity)
	pSendMessage.Call(opacitySlider, 0x405, 1, 100)
	width, height := windowOuterForLogical(appearance.Width, appearance.Height)
	pSetWindowPos.Call(mainWnd, 0, 0, 0, uintptr(width), uintptr(height), 0x2|0x4)
}
func refreshView() {
	switch view {
	case 0:
		refreshCountdown()
	case 1:
		refreshToday()
	case 2:
		refreshCalendar()
	case 3:
		refreshNotes()
	}
}

func refreshCountdown() {
	now := time.Now()
	weekdays := [...]string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}
	setText(countdownDate, "今天 · "+now.Format("2006年1月2日")+" "+weekdays[now.Weekday()])
	for i, k := range Events {
		name, customized := customName(k)
		c := describeCountdown(k, state, now, name, customized)
		cardStatus[i] = c
		s := name + "，" + c.description()
		if c.Date != "" {
			s += "，" + c.Date
		}
		setText(countdownButtons[i], s)
		pInvalidateRect.Call(countdownButtons[i], 0, 0)
	}
}

func invalidateNavigation() {
	for _, h := range navControls {
		pInvalidateRect.Call(h, 0, 0)
	}
}
func refreshToday() {
	selected, hasSelection := selectedRef(todayList, todayRefs)
	listReset(todayList)
	todayRefs = todayRefs[:0]
	today := civil(time.Now()).Format("2006-01-02")
	show, _, _ := pSendMessage.Call(showCompleted, bmGetCheck, 0, 0)
	horizon, includeOverdue, includeUnscheduled := 1, true, true
	setText(modeTodayButton, "● 今日")
	setText(modeFutureButton, "○ 未来7天")
	if todayMode == 1 {
		horizon, includeOverdue, includeUnscheduled = 7, false, false
		setText(modeTodayButton, "○ 今日")
		setText(modeFutureButton, "● 未来7天")
	}
	type row struct {
		ref      TaskOccurrence
		group    int
		text     string
		priority bool
	}
	rows := []row{}
	items := SelectTaskListOccurrences(planner.Tasks, today, horizon, getText(todaySearch), includeOverdue, includeUnscheduled)
	for _, item := range items {
		t, ok := taskByID(item.TaskID)
		if !ok {
			continue
		}
		group, label := item.DayOffset+1, "今天"
		switch item.Category {
		case "overdue":
			group, label = 0, "逾期 "+item.Date
		case "unscheduled":
			group, label = horizon+1, "待安排"
		case "scheduled":
			if item.DayOffset > 0 {
				d, _ := parseDate(item.Date)
				weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
				label = d.Format("01月02日") + " " + weekdays[d.Weekday()]
			}
		}
		done := IsTaskCompleted(t, item.Date)
		if done && show != 1 {
			continue
		}
		if done && todayMode == 0 {
			group = 3
			label = "已完成 · " + label
		}
		mark := "○ "
		if done {
			mark = "✓ "
		}
		star := ""
		if t.Priority {
			star = "★ "
		}
		rows = append(rows, row{TaskOccurrence{t.ID, item.Date}, group, "[" + label + "] " + mark + star + t.Title, t.Priority})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].group != rows[j].group {
			return rows[i].group < rows[j].group
		}
		if rows[i].priority != rows[j].priority {
			return rows[i].priority
		}
		return rows[i].text < rows[j].text
	})
	for _, r := range rows {
		listAdd(todayList, r.text)
		todayRefs = append(todayRefs, r.ref)
		if hasSelection && r.ref == selected {
			pSendMessage.Call(todayList, lbSetCurSel, uintptr(len(todayRefs)-1), 0)
		}
	}
}
func countdownBadge(date string) string {
	names := []string{}
	for _, k := range Events {
		d := state.Dates[k]
		if date == d.Start || d.End != "" && date == d.End {
			name, _ := customName(k)
			names = append(names, name)
		}
	}
	return strings.Join(names, "/")
}
func refreshCalendar() {
	selected, hasSelection := selectedRef(calendarList, calendarRefs)
	setText(calendarLabel, fmt.Sprintf("%d 年 %d 月", calendarMonth.Year(), calendarMonth.Month()))
	selectedDate := calendarSelected.Format("2006-01-02")
	calendarRefs = TasksForDate(planner.Tasks, selectedDate)
	setText(calendarSelectedLabel, fmt.Sprintf("%s · %d 项待办", calendarSelected.Format("01月02日"), len(calendarRefs)))
	for i, day := range MonthGrid(calendarMonth, time.Now()) {
		count := len(TasksForDate(planner.Tasks, day.Date))
		badge := countdownBadge(day.Date)
		calendarDaysPaint[i] = calendarDayPaint{
			number: fmt.Sprint(day.Day), inMonth: day.InMonth, today: day.IsToday,
			selected: day.Date == selectedDate, hasTasks: count != 0, hasBadge: badge != "",
		}
		label := fmt.Sprintf("%s，%d 项待办", day.Date, count)
		if day.IsToday {
			label += "，今天"
		}
		if day.Date == selectedDate {
			label += "，已选"
		}
		if badge != "" {
			label += "，" + badge
		}
		setText(calendarDays[i], label)
		pInvalidateRect.Call(calendarDays[i], 0, 0)
	}
	listReset(calendarList)
	for i, r := range calendarRefs {
		t, _ := taskByID(r.TaskID)
		m := "○ "
		if IsTaskCompleted(t, r.Date) {
			m = "✓ "
		}
		if t.Priority {
			m += "★ "
		}
		listAdd(calendarList, m+t.Title)
		if hasSelection && r == selected {
			pSendMessage.Call(calendarList, lbSetCurSel, uintptr(i), 0)
		}
	}
}
func refreshNotes() {
	selectedID := ""
	if i := listSel(noteList); i >= 0 && i < len(noteRefs) {
		selectedID = noteRefs[i]
	}
	listReset(noteList)
	noteRefs = noteRefs[:0]
	q := strings.ToLower(strings.TrimSpace(getText(noteSearch)))
	ii := make([]int, len(planner.Notes))
	for i := range ii {
		ii[i] = i
	}
	sort.SliceStable(ii, func(i, j int) bool { return planner.Notes[ii[i]].UpdatedAt > planner.Notes[ii[j]].UpdatedAt })
	for _, i := range ii {
		n := planner.Notes[i]
		if q == "" || strings.Contains(strings.ToLower(n.Title), q) || strings.Contains(strings.ToLower(n.Body), q) {
			listAdd(noteList, n.Title)
			noteRefs = append(noteRefs, n.ID)
			if n.ID == selectedID {
				pSendMessage.Call(noteList, lbSetCurSel, uintptr(len(noteRefs)-1), 0)
			}
		}
	}
}

func openDateEditor(k Kind) {
	dateEditorOpen = true
	dateEditing = k
	setVisible(navControls, false)
	setVisible(viewControls[view], false)
	setVisible(dateEditor, true)
	name, customized := customName(k)
	setText(dateContext, name+"日期")
	d := state.Dates[k]
	if d.Start == "" {
		d.Start = SuggestedStart(k, state, time.Now())
	}
	setText(dateStart, d.Start)
	setText(dateEnd, d.End)
	on := k != Retirement
	pShowWindow.Call(dateEnd, map[bool]uintptr{true: swShow, false: swHide}[on])
	pShowWindow.Call(dateEndLabel, map[bool]uintptr{true: swShow, false: swHide}[on])
	if k == Labor || k == National {
		if customized {
			setText(dateReset, "恢复默认日期")
		} else {
			setText(dateReset, "恢复节日默认")
		}
	} else {
		setText(dateReset, "清空日期")
	}
	pSetFocus.Call(dateStart)
}
func closeDateEditor() {
	dateEditorOpen = false
	setVisible(dateEditor, false)
	setVisible(navControls, true)
	setVisible(viewControls[view], true)
	refreshView()
}
func saveDateEditor() {
	d := Dates{Start: strings.TrimSpace(getText(dateStart))}
	if dateEditing != Retirement {
		d.End = strings.TrimSpace(getText(dateEnd))
	}
	if err := state.SetDates(dateEditing, &d); err != nil {
		message(mainWnd, "保存失败："+err.Error(), 0x10)
		return
	}
	closeDateEditor()
}

func repeatIndex(r Repeat) int {
	switch r {
	case RepeatDaily:
		return 1
	case RepeatWeekdays:
		return 2
	case RepeatWeekly:
		return 3
	}
	return 0
}
func repeatAt(i int) Repeat {
	return []Repeat{RepeatNone, RepeatDaily, RepeatWeekdays, RepeatWeekly}[max(0, min(i, 3))]
}
func openTaskEditor(id, prefill string) {
	taskEditorOpen = true
	editingTaskID = id
	setVisible(navControls, false)
	setVisible(viewControls[view], false)
	setVisible(taskEditor, true)
	setText(taskTitle, "")
	setText(taskDetails, "")
	setText(taskDate, prefill)
	setText(taskReminder, "")
	pSendMessage.Call(taskRepeat, cbSetCurSel, 0, 0)
	pSendMessage.Call(taskPriority, bmSetCheck, 0, 0)
	if t, ok := taskByID(id); ok {
		setText(taskTitle, t.Title)
		setText(taskDetails, t.Details)
		setText(taskDate, t.Date)
		setText(taskReminder, t.ReminderTime)
		pSendMessage.Call(taskRepeat, cbSetCurSel, uintptr(repeatIndex(t.Repeat)), 0)
		if t.Priority {
			pSendMessage.Call(taskPriority, bmSetCheck, 1, 0)
		}
	}
	pSetFocus.Call(taskTitle)
}
func closeTaskEditor() {
	taskEditorOpen = false
	setVisible(taskEditor, false)
	setVisible(navControls, true)
	setVisible(viewControls[view], true)
	refreshView()
}
func saveTaskEditor() {
	r, _, _ := pSendMessage.Call(taskRepeat, cbGetCurSel, 0, 0)
	check, _, _ := pSendMessage.Call(taskPriority, bmGetCheck, 0, 0)
	t := Task{ID: editingTaskID, Title: getText(taskTitle), Details: getText(taskDetails), Date: strings.TrimSpace(getText(taskDate)), ReminderTime: strings.TrimSpace(getText(taskReminder)), Repeat: repeatAt(int(r)), Priority: check == 1}
	if t.Repeat != RepeatNone && t.Date == "" {
		message(mainWnd, "重复任务需要填写起始日期。", 0x10)
		pSetFocus.Call(taskDate)
		return
	}
	if old, ok := taskByID(editingTaskID); ok {
		t.CreatedDate = old.CreatedDate
		t.SourceNoteID = old.SourceNoteID
	}
	if _, err := planner.UpsertTask(t, time.Now()); err != nil {
		message(mainWnd, "保存失败："+err.Error(), 0x10)
		return
	}
	closeTaskEditor()
}
func quickAdd() {
	title := strings.TrimSpace(getText(quickTitle))
	if title == "" {
		return
	}
	now := time.Now()
	id, err := planner.UpsertTask(Task{Title: title, Date: civil(now).Format("2006-01-02"), Repeat: RepeatNone}, now)
	if err != nil {
		message(mainWnd, "添加失败："+err.Error(), 0x10)
		return
	}
	setText(quickTitle, "")
	setText(todaySearch, "")
	refreshToday()
	for i, ref := range todayRefs {
		if ref.TaskID == id {
			pSendMessage.Call(todayList, lbSetCurSel, uintptr(i), 0)
			break
		}
	}
}
func toggleTask(r TaskOccurrence) {
	t, ok := taskByID(r.TaskID)
	if !ok {
		return
	}
	if err := planner.SetTaskCompleted(t.ID, r.Date, !IsTaskCompleted(t, r.Date)); err != nil {
		message(mainWnd, "无法更新任务："+err.Error(), 0x10)
		return
	}
	refreshView()
}
func deleteTask(r TaskOccurrence) {
	t, ok := taskByID(r.TaskID)
	if !ok || message(mainWnd, "删除任务“"+t.Title+"”？", 0x04|0x20) != 6 {
		return
	}
	if err := planner.DeleteTask(t.ID); err != nil {
		message(mainWnd, "删除失败："+err.Error(), 0x10)
		return
	}
	refreshView()
}
func selectCalendarDay(i int) string {
	grid := MonthGrid(calendarMonth, time.Now())
	if i < 0 || i >= len(grid) {
		return ""
	}
	date, err := parseDate(grid[i].Date)
	if err != nil {
		return ""
	}
	calendarSelected = date
	if date.Month() != calendarMonth.Month() {
		calendarMonth = time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.Local)
	}
	refreshCalendar()
	return grid[i].Date
}

func loadSelectedNote() {
	i := listSel(noteList)
	if i < 0 || i >= len(noteRefs) {
		return
	}
	if !confirmDiscardNote() {
		oldIndex := -1
		for j, id := range noteRefs {
			if id == editingNoteID {
				oldIndex = j
				break
			}
		}
		pSendMessage.Call(noteList, lbSetCurSel, uintptr(oldIndex), 0)
		return
	}
	for _, n := range planner.Notes {
		if n.ID == noteRefs[i] {
			editingNoteID = n.ID
			noteSavedTitle = n.Title
			noteSavedBody = n.Body
			setText(noteTitle, n.Title)
			setText(noteBody, n.Body)
			return
		}
	}
}
func newNote() {
	if !confirmDiscardNote() {
		return
	}
	editingNoteID = ""
	noteSavedTitle = ""
	noteSavedBody = ""
	setText(noteTitle, "")
	setText(noteBody, "")
	pSetFocus.Call(noteTitle)
}
func saveNote() {
	n := Note{ID: editingNoteID, Title: getText(noteTitle), Body: getText(noteBody)}
	if oldID := editingNoteID; oldID != "" {
		for _, o := range planner.Notes {
			if o.ID == oldID {
				n.CreatedAt = o.CreatedAt
			}
		}
	}
	id, err := planner.UpsertNote(n, time.Now())
	if err != nil {
		message(mainWnd, "保存失败："+err.Error(), 0x10)
		return
	}
	editingNoteID = id
	noteSavedTitle = strings.TrimSpace(n.Title)
	noteSavedBody = strings.TrimSpace(n.Body)
	setText(noteTitle, noteSavedTitle)
	setText(noteBody, noteSavedBody)
	refreshNotes()
}
func deleteNote() {
	if editingNoteID == "" || message(mainWnd, "删除这条笔记？由它生成的任务会保留。", 0x04|0x20) != 6 {
		return
	}
	if err := planner.DeleteNote(editingNoteID); err != nil {
		message(mainWnd, "删除失败："+err.Error(), 0x10)
		return
	}
	editingNoteID = ""
	noteSavedTitle = ""
	noteSavedBody = ""
	setText(noteTitle, "")
	setText(noteBody, "")
	refreshNotes()
}
func makeTaskFromNote() {
	if editingNoteID == "" || noteDirty() {
		message(mainWnd, "请先保存笔记，再生成任务。", 0x40)
		return
	}
	if _, err := planner.CreateTaskFromNote(editingNoteID, civil(time.Now()).Format("2006-01-02"), time.Now()); err != nil {
		message(mainWnd, "生成任务失败："+err.Error(), 0x10)
		return
	}
	message(mainWnd, "已生成今天的任务；原笔记仍然保留。", 0x40)
}

func addTray() {
	if trayAdded {
		return
	}
	n := notifyIconData{CbSize: uint32(unsafe.Sizeof(notifyIconData{})), HWnd: mainWnd, UID: 1, UFlags: 7, UCallbackMessage: wmTray, HIcon: appIcon}
	copy16(n.SzTip[:], "有盼 · 关闭窗口后仍在托盘运行")
	r, _, _ := pShellNotifyIcon.Call(0, uintptr(unsafe.Pointer(&n)))
	trayAdded = r != 0
}
func removeTray() {
	if !trayAdded {
		return
	}
	n := notifyIconData{CbSize: uint32(unsafe.Sizeof(notifyIconData{})), HWnd: mainWnd, UID: 1}
	pShellNotifyIcon.Call(2, uintptr(unsafe.Pointer(&n)))
	trayAdded = false
}
func balloon(title, body string) {
	if !trayAdded {
		return
	}
	n := notifyIconData{CbSize: uint32(unsafe.Sizeof(notifyIconData{})), HWnd: mainWnd, UID: 1, UFlags: 0x10, DwInfoFlags: 1}
	copy16(n.SzInfoTitle[:], title)
	copy16(n.SzInfo[:], body)
	pShellNotifyIcon.Call(1, uintptr(unsafe.Pointer(&n)))
}
func showMain() { pShowWindow.Call(mainWnd, swRestore); pSetForegroundWindow.Call(mainWnd) }
func trayMenu() {
	m, _, _ := pCreatePopupMenu.Call()
	if m == 0 {
		return
	}
	defer pDestroyMenu.Call(m)
	pAppendMenu.Call(m, 0, idTrayOpen, uintptr(unsafe.Pointer(w("打开有盼"))))
	pAppendMenu.Call(m, 0x800, 0, 0)
	pAppendMenu.Call(m, 0, idTrayQuit, uintptr(unsafe.Pointer(w("退出有盼"))))
	var pt point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	pSetForegroundWindow.Call(mainWnd)
	pTrackPopupMenu.Call(m, 0x102, uintptr(pt.X), uintptr(pt.Y), 0, mainWnd, 0)
}
func checkReminders() {
	due := planner.DueReminders(time.Now())
	if len(due) == 0 {
		return
	}
	if err := planner.MarkRemindersFired(due); err != nil {
		return
	}
	lines := []string{}
	for _, r := range due {
		lines = append(lines, r.Time+"  "+r.Title)
	}
	body := strings.Join(lines, "\n")
	presentReminder("现在要做：\n\n" + body)
}
func presentReminder(text string) {
	balloon("有盼提醒", text)
	message(mainWnd, text, 0x40)
}

func handleCommand(id int, n uint16) {
	if id >= idCalDayBase && id < idCalDayBase+len(calendarDays) {
		if view != 2 || settingsOpen || dateEditorOpen || taskEditorOpen {
			return
		}
		index := id - idCalDayBase
		switch n {
		case 0:
			lastCalendarClick = index
			lastCalendarClickDate = selectCalendarDay(index)
		case bnDoubleClicked:
			// A spillover date can change the month after the first click.
			day := lastCalendarClickDate
			if day == "" || lastCalendarClick != index {
				day = selectCalendarDay(index)
			}
			lastCalendarClickDate = ""
			if day != "" {
				openTaskEditor("", day)
			}
		}
		return
	}
	if n == 0 {
		lastCalendarClickDate = ""
	}
	if id == idTodaySearch && n == 0x300 {
		refreshToday()
		return
	}
	if id == idNoteSearch && n == 0x300 {
		refreshNotes()
		return
	}
	if id == idNoteList && n == 1 {
		loadSelectedNote()
		return
	}
	if n != 0 {
		return
	}
	switch id {
	case idNavCountdown:
		showView(0)
	case idNavToday:
		showView(1)
	case idNavCalendar:
		showView(2)
	case idNavNotes:
		showView(3)
	case idPin:
		c, _, _ := pSendMessage.Call(pinButton, bmGetCheck, 0, 0)
		if err := state.SetPinned(c == 1); err != nil {
			message(mainWnd, "无法保存置顶设置："+err.Error(), 0x10)
			if state.Pinned {
				pSendMessage.Call(pinButton, bmSetCheck, 1, 0)
			} else {
				pSendMessage.Call(pinButton, bmSetCheck, 0, 0)
			}
		}
		setTopmost(state.Pinned)
	case idSettings:
		openSettings()
	case idNamesSave:
		saveNames()
	case idNamesReset:
		resetNames()
	case idVisualReset:
		resetVisuals()
	case idSettingsBack:
		closeSettings()
	case idDateSave:
		saveDateEditor()
	case idDateCancel:
		closeDateEditor()
	case idDateReset:
		if message(mainWnd, "确认清空或恢复此倒计时的默认日期？", 0x04|0x20) != 6 {
			return
		}
		if err := state.SetDates(dateEditing, nil); err != nil {
			message(mainWnd, "无法保存更改："+err.Error(), 0x10)
		} else {
			closeDateEditor()
		}
	case idQuickAdd:
		quickAdd()
	case idShowCompleted:
		refreshToday()
	case idModeToday:
		todayMode = 0
		refreshToday()
	case idModeFuture:
		todayMode = 1
		refreshToday()
	case idTodayDone:
		if r, ok := selectedRef(todayList, todayRefs); ok {
			toggleTask(r)
		}
	case idTodayEdit:
		if r, ok := selectedRef(todayList, todayRefs); ok {
			openTaskEditor(r.TaskID, "")
		}
	case idTodayDelete:
		if r, ok := selectedRef(todayList, todayRefs); ok {
			deleteTask(r)
		}
	case idTestReminder:
		presentReminder("这是测试提醒。\n\n测试不会创建任务，也不会写入提醒记录。")
	case idTaskSave:
		saveTaskEditor()
	case idTaskCancel:
		closeTaskEditor()
	case idCalPrev:
		calendarMonth = calendarMonth.AddDate(0, -1, 0)
		calendarSelected = civil(calendarMonth)
		refreshCalendar()
	case idCalNext:
		calendarMonth = calendarMonth.AddDate(0, 1, 0)
		calendarSelected = civil(calendarMonth)
		refreshCalendar()
	case idCalToday:
		now := time.Now()
		calendarMonth = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		calendarSelected = civil(now)
		refreshCalendar()
	case idCalAdd:
		openTaskEditor("", calendarSelected.Format("2006-01-02"))
	case idCalDone:
		if r, ok := selectedRef(calendarList, calendarRefs); ok {
			toggleTask(r)
		}
	case idCalEdit:
		if r, ok := selectedRef(calendarList, calendarRefs); ok {
			openTaskEditor(r.TaskID, r.Date)
		}
	case idNoteNew:
		newNote()
	case idNoteSave:
		saveNote()
	case idNoteDelete:
		deleteNote()
	case idNoteMakeTask:
		makeTaskFromNote()
	case idTrayOpen:
		showMain()
	case idTrayQuit:
		if settingsOpen && !finishNamesEditing() {
			return
		}
		if view == 3 && !confirmDiscardNote() {
			return
		}
		savePreviewOpacity()
		saveCurrentWindowSize()
		exiting = true
		pDestroyWindow.Call(mainWnd)
	default:
		if id >= idCountdownBase && id < idCountdownBase+len(Events) {
			openDateEditor(Events[id-idCountdownBase])
		}
	}
}

func wndProc(hwnd uintptr, m uint32, wp, lp uintptr) uintptr {
	switch m {
	case wmGetMinMaxInfo:
		if lp != 0 {
			minimum := point{int32(minOuterWidth), int32(minOuterHeight)}
			maximum := point{int32(px(maxWindowWidth)), int32(px(maxWindowHeight))}
			// MINMAXINFO.MinTrackSize follows three POINT fields. Copying via
			// Win32 avoids retaining callback-owned memory as a Go pointer.
			pCopyMemory.Call(lp+3*unsafe.Sizeof(point{}), uintptr(unsafe.Pointer(&minimum)), unsafe.Sizeof(minimum))
			pCopyMemory.Call(lp+4*unsafe.Sizeof(point{}), uintptr(unsafe.Pointer(&maximum)), unsafe.Sizeof(maximum))
		}
		return 0
	case wmSize:
		if wp != 1 && mainWnd != 0 { // SIZE_MINIMIZED
			layoutControls(logical(int32(lp&0xffff)), logical(int32((lp>>16)&0xffff)))
			if windowReady {
				sizeSavePending = true
				pSetTimer.Call(mainWnd, idTimerWindowSave, 400, 0)
			}
		}
		return 0
	case wmDrawItem:
		if drawSurface(lp) {
			return 1
		}
	case 0x000f:
		var ps paintStruct
		dc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if dc != 0 {
			pFillRect.Call(dc, uintptr(unsafe.Pointer(&ps.Paint)), bgBrush)
			client := rect{}
			pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
			r := rect{0, 0, int32(px(142)), client.Bottom}
			pFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), sideBrush)
			line := rect{int32(px(142)), 0, int32(px(142)) + 1, client.Bottom}
			fillColor(dc, line, rgb(217, 225, 235))
		}
		pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case 0x0138:
		pSetTextColor.Call(wp, rgb(31, 41, 55))
		pSetBkColor.Call(wp, rgb(244, 247, 251))
		return bgBrush
	case 0x0135: // WM_CTLCOLORBTN
		if lp == pinButton {
			pSetTextColor.Call(wp, textInk)
			pSetBkColor.Call(wp, rgb(234, 239, 245))
			return sideBrush
		}
	case wmCommand:
		handleCommand(int(wp&0xffff), uint16(wp>>16))
		return 0
	case wmHScroll:
		if lp == opacitySlider {
			pos, _, _ := pSendMessage.Call(opacitySlider, 0x400, 0, 0)
			applyOpacity(float64(pos) / 100)
			opacitySavePending = true
			pSetTimer.Call(mainWnd, idTimerOpacitySave, 250, 0)
			return 0
		}
	case wmTimer:
		if wp == idTimerWindowSave {
			pKillTimer.Call(mainWnd, idTimerWindowSave)
			if sizeSavePending {
				sizeSavePending = false
				saveCurrentWindowSize()
			}
			return 0
		}
		if wp == idTimerOpacitySave {
			pKillTimer.Call(mainWnd, idTimerOpacitySave)
			savePreviewOpacity()
			return 0
		}
		refreshView()
		checkReminders()
		return 0
	case wmTray:
		if uint32(lp&0xffff) == wmLButtonUp {
			showMain()
		} else if uint32(lp&0xffff) == wmRButtonUp {
			trayMenu()
		}
		return 0
	case wmClose:
		if exiting {
			pDestroyWindow.Call(hwnd)
			return 0
		}
		savePreviewOpacity()
		saveCurrentWindowSize()
		pShowWindow.Call(hwnd, swHide)
		if !trayHintShown {
			trayHintShown = true
			balloon("有盼仍在运行", "关闭窗口后仍在托盘运行，退出有盼后停止提醒。")
		}
		return 0
	case wmDestroy:
		removeTray()
		pPostQuit.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, uintptr(m), wp, lp)
	return r
}
func rgb(r, g, b uintptr) uintptr { return r | g<<8 | b<<16 }

func buildControls() {
	for i, s := range navNames[:4] {
		navControls = append(navControls, create("BUTTON", s, wsTabstop|bsOwnerDraw, 0, 12, 24+i*47, 118, 38, idNavCountdown+i))
	}
	pinButton = create("BUTTON", "窗口置顶", wsTabstop|3, 0, 16, baseClientHeight-88, 112, 25, idPin)
	font(pinButton, smallFont)
	if state.Pinned {
		pSendMessage.Call(pinButton, bmSetCheck, 1, 0)
	}
	settingsButton = create("BUTTON", "设置", wsTabstop|bsOwnerDraw, 0, 12, baseClientHeight-52, 118, 38, idSettings)
	navControls = append(navControls, pinButton, settingsButton)
	countdownDate = static("", 166, 62, 330, 24, false)
	font(countdownDate, smallFont)
	otherHeading := static("其他倒计时", 166, 230, 180, 26, false)
	font(otherHeading, boldFont)
	v0 := []uintptr{static("倒计时", 166, 24, 220, 38, true), countdownDate, otherHeading}
	for i, k := range Events {
		y, height := 262+(i-1)*71, 62
		if i == 0 {
			y, height = 94, 124
		}
		h := create("BUTTON", k.Title(), wsTabstop|bsOwnerDraw, 0, 166, y, 552, height, idCountdownBase+i)
		countdownButtons[i] = h
		v0 = append(v0, h)
	}
	viewControls[0] = v0
	v1 := []uintptr{static("今天", 166, 24, 160, 40, true), static(time.Now().Format("2006年01月02日"), 166, 64, 220, 25, false)}
	quickTitle = edit("", 166, 100, 438, 34, idQuickTitle, false)
	v1 = append(v1, quickTitle, button("添加到今天", 614, 100, 104, 34, idQuickAdd))
	v1 = append(v1, static("搜索", 166, 150, 48, 25, false))
	todaySearch = edit("", 216, 145, 266, 32, idTodaySearch, false)
	modeTodayButton = button("● 今日", 494, 145, 94, 32, idModeToday)
	modeFutureButton = button("○ 未来7天", 598, 145, 120, 32, idModeFuture)
	v1 = append(v1, todaySearch, modeTodayButton, modeFutureButton)
	showCompleted = create("BUTTON", "显示已完成", wsTabstop|3, 0, 166, 184, 130, 28, idShowCompleted)
	font(showCompleted, smallFont)
	todayList = listbox(166, 216, 552, 290, idTodayList)
	v1 = append(v1, showCompleted, todayList, button("完成 / 恢复", 166, 520, 120, 36, idTodayDone), button("编辑", 296, 520, 86, 36, idTodayEdit), button("删除", 392, 520, 86, 36, idTodayDelete), button("测试提醒", 488, 520, 104, 36, idTestReminder))
	todayHint = static("未来7天不含未安排事项；重复任务按每个发生日独立完成。", 166, 571, 540, 25, false)
	font(todayHint, smallFont)
	v1 = append(v1, todayHint)
	viewControls[1] = v1
	v2 := []uintptr{button("‹", 166, 22, 42, 34, idCalPrev), button("今天", 216, 22, 68, 34, idCalToday), button("›", 292, 22, 42, 34, idCalNext)}
	calendarLabel = static("", 350, 22, 190, 38, true)
	v2 = append(v2, calendarLabel)
	for i, s := range []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"} {
		calendarWeekLabels[i] = create("STATIC", s, 1, 0, 166+i*79, 68, 70, 24, 0)
		font(calendarWeekLabels[i], smallFont)
		v2 = append(v2, calendarWeekLabels[i])
	}
	for i := 0; i < 42; i++ {
		calendarDays[i] = create("BUTTON", "", wsTabstop|bsOwnerDraw, 0, 166+(i%7)*79, 94+(i/7)*47, 70, 40, idCalDayBase+i)
		v2 = append(v2, calendarDays[i])
	}
	calendarSelectedLabel = static("", 166, 386, 330, 26, false)
	calendarList = listbox(166, 416, 552, 132, idCalList)
	calHint := static("双击日期添加待办", 541, 387, 177, 25, false)
	font(calHint, smallFont)
	v2 = append(v2, calendarSelectedLabel, calHint, calendarList, button("新建待办", 166, 562, 145, 36, idCalAdd), button("完成 / 恢复", 321, 562, 120, 36, idCalDone), button("编辑", 451, 562, 86, 36, idCalEdit))
	viewControls[2] = v2
	v3 := []uintptr{static("笔记", 166, 24, 160, 40, true), static("搜索", 166, 73, 60, 26, false)}
	noteSearch = edit("", 222, 69, 496, 32, idNoteSearch, false)
	noteList = listbox(166, 112, 205, 398, idNoteList)
	noteTitle = edit("", 388, 112, 330, 34, idNoteTitle, false)
	noteBody = edit("", 388, 158, 330, 352, idNoteBody, true)
	v3 = append(v3, noteSearch, noteList, noteTitle, noteBody, button("新建", 166, 524, 76, 36, idNoteNew), button("保存笔记", 388, 524, 100, 36, idNoteSave), button("删除", 498, 524, 76, 36, idNoteDelete), button("生成今天任务", 584, 524, 134, 36, idNoteMakeTask))
	noteHint = static("生成任务会保留笔记，并在任务中记录来源链接。", 388, 574, 330, 25, false)
	font(noteHint, smallFont)
	v3 = append(v3, noteHint)
	viewControls[3] = v3
	dateEditor = append(dateEditor, static("设置日期", 180, 48, 260, 44, true))
	dateContext = static("", 180, 105, 300, 30, false)
	dateEditor = append(dateEditor, dateContext, static("开始日期（YYYY-MM-DD）", 180, 165, 360, 28, false))
	dateStart = edit("", 180, 198, 390, 36, idDateStart, false)
	dateEndLabel = static("结束日期（可选）", 180, 265, 360, 28, false)
	dateEnd = edit("", 180, 298, 390, 36, idDateEnd, false)
	dateReset = button("清空日期", 180, 382, 132, 38, idDateReset)
	dateEditor = append(dateEditor, dateStart, dateEndLabel, dateEnd, dateReset, button("保存", 322, 382, 102, 38, idDateSave), button("取消", 434, 382, 102, 38, idDateCancel))
	setVisible(dateEditor, false)
	taskEditor = append(taskEditor, static("任务", 180, 30, 200, 44, true), static("标题", 180, 88, 90, 26, false))
	taskTitle = edit("", 180, 116, 520, 34, idTaskTitle, false)
	taskEditor = append(taskEditor, taskTitle, static("详情", 180, 164, 90, 26, false))
	taskDetails = edit("", 180, 192, 520, 130, idTaskDetails, true)
	taskEditor = append(taskEditor, taskDetails, static("日期（重复/提醒时必填）", 180, 338, 220, 25, false))
	taskDate = edit("", 180, 366, 220, 34, idTaskDate, false)
	taskEditor = append(taskEditor, taskDate, static("提醒（HH:mm，需有日期）", 428, 338, 240, 25, false))
	taskReminder = edit("", 428, 366, 220, 34, idTaskReminder, false)
	taskEditor = append(taskEditor, taskReminder, static("重复", 180, 418, 90, 25, false))
	taskRepeat = create("COMBOBOX", "", wsTabstop|0x0003|wsVScroll, 0, 180, 446, 220, 150, idTaskRepeat)
	for _, s := range []string{"不重复", "每天", "周一至周五", "每周"} {
		pSendMessage.Call(taskRepeat, cbAddString, 0, uintptr(unsafe.Pointer(w(s))))
	}
	pSendMessage.Call(taskRepeat, cbSetCurSel, 0, 0)
	taskPriority = create("BUTTON", "星标优先", wsTabstop|3, 0, 428, 446, 130, 30, idTaskPriority)
	taskEditor = append(taskEditor, taskRepeat, taskPriority, button("保存任务", 180, 522, 112, 40, idTaskSave), button("取消", 302, 522, 92, 40, idTaskCancel))
	hh := static("提醒只在有盼运行时生效；关闭窗口仍运行，退出后停止。", 180, 582, 500, 26, false)
	font(hh, smallFont)
	taskEditor = append(taskEditor, hh)
	setVisible(taskEditor, false)
	settingsControls = append(settingsControls, static("设置", 166, 24, 160, 40, true), static("窗口大小", 166, 76, 120, 28, false))
	resizeHint := static("直接拖动窗口边角调整大小，松手后自动保存。", 166, 108, 440, 28, false)
	font(resizeHint, smallFont)
	settingsControls = append(settingsControls, resizeHint)
	opacityLabel = static("透明度：100%", 166, 160, 180, 28, false)
	opacitySlider = create("msctls_trackbar32", "", wsTabstop|0x1, 0, 166, 190, 410, 38, idOpacitySlider)
	pSendMessage.Call(opacitySlider, 0x406, 1, uintptr(100<<16|50)) // TBM_SETRANGE
	pSendMessage.Call(opacitySlider, 0x405, 1, uintptr(int(appearance.Opacity*100+.5)))
	settingsControls = append(settingsControls, opacityLabel, opacitySlider, static("50%", 166, 228, 48, 22, false), static("100%", 536, 228, 52, 22, false))
	settingsControls = append(settingsControls, static("倒计时名称（1–24 个字符）", 166, 265, 300, 28, false))
	for i, k := range Events {
		y := 302 + i*46
		settingsControls = append(settingsControls, static(k.Title(), 166, y+5, 90, 26, false))
		name, _ := customName(k)
		nameEdits[i] = edit(name, 260, y, 316, 32, idNameBase+i, false)
		settingsControls = append(settingsControls, nameEdits[i])
	}
	settingsControls = append(settingsControls, button("保存名称", 166, 544, 110, 36, idNamesSave), button("恢复默认名称", 286, 544, 130, 36, idNamesReset), button("恢复默认外观", 426, 544, 130, 36, idVisualReset), button("返回", 566, 544, 90, 36, idSettingsBack))
	settingsStatus = static("", 166, 590, 410, 25, false)
	font(settingsStatus, smallFont)
	settingsControls = append(settingsControls, settingsStatus)
	setVisible(settingsControls, false)
	for i := 1; i < 4; i++ {
		setVisible(viewControls[i], false)
	}
	refreshCountdown()
}

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pSetProcessDPIAware.Call()
	icc := initCommonControls{Size: uint32(unsafe.Sizeof(initCommonControls{})), Classes: 0x4}
	pInitCommonControls.Call(uintptr(unsafe.Pointer(&icc)))
	if pGetDpiForSystem.Find() == nil {
		if dpi, _, _ := pGetDpiForSystem.Call(); dpi > 0 {
			systemScale = float64(dpi) / 96
		}
	}
	scale = systemScale
	configureWindowGeometry()
	var mutex uintptr
	for i := 0; i < 20; i++ {
		h, _, e := pCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(w("Local\\DaysCountdownSingleInstance"))))
		if h == 0 {
			message(0, "无法启动应用："+e.Error(), 0x10)
			return
		}
		if e != syscall.Errno(183) {
			mutex = h
			break
		}
		pCloseHandle.Call(h)
		if old, _, _ := pFindWindow.Call(uintptr(unsafe.Pointer(w("DaysCountdownMainWindow"))), 0); old != 0 {
			pShowWindow.Call(old, swRestore)
			pSetForegroundWindow.Call(old)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	if mutex == 0 {
		return
	}
	defer pCloseHandle.Call(mutex)
	var err error
	state, err = LoadState()
	if err != nil {
		message(0, "读取倒计时数据失败，未进行任何覆盖。\n\n"+err.Error(), 0x10)
		return
	}
	planner, err = LoadPlanner()
	if err != nil {
		message(0, "读取计划数据失败，未进行任何覆盖。\n\n"+err.Error(), 0x10)
		return
	}
	appearance, err = LoadAppearance()
	if err != nil {
		message(0, "读取外观设置失败；倒计时和计划数据仍可使用。为避免覆盖，当前不会保存新的外观设置。\n\n"+err.Error(), 0x30)
	}
	previewOpacity = appearance.Opacity
	inst, _, _ := pGetModuleHandle.Call(0)
	cursor, _, _ := pLoadCursor.Call(0, 32512)
	appIcon, _, _ = pLoadIcon.Call(inst, 1)
	if appIcon == 0 {
		appIcon, _, _ = pLoadIcon.Call(0, 32512)
	}
	cls := w("DaysCountdownMainWindow")
	wc := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: syscall.NewCallback(wndProc), Instance: inst, Icon: appIcon, Cursor: cursor, Background: 6, ClassName: cls, IconSmall: appIcon}
	if r, _, _ := pRegisterClass.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		message(0, "无法注册应用窗口。", 0x10)
		return
	}
	normalFont, _, _ = pCreateFont.Call(^uintptr(px(17)-1), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(w("Microsoft YaHei UI"))))
	headingFont, _, _ = pCreateFont.Call(^uintptr(px(27)-1), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(w("Microsoft YaHei UI"))))
	smallFont, _, _ = pCreateFont.Call(^uintptr(px(13)-1), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(w("Microsoft YaHei UI"))))
	boldFont, _, _ = pCreateFont.Call(^uintptr(px(18)-1), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(w("Microsoft YaHei UI"))))
	bgBrush, _, _ = pCreateSolidBrush.Call(rgb(244, 247, 251))
	sideBrush, _, _ = pCreateSolidBrush.Call(rgb(234, 239, 245))
	heroFont, _, _ = pCreateFont.Call(^uintptr(px(41)-1), 0, 0, 0, 700, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(w("Microsoft YaHei UI"))))
	metricFont, _, _ = pCreateFont.Call(^uintptr(px(21)-1), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(w("Microsoft YaHei UI"))))
	symbolFont, _, _ = pCreateFont.Call(^uintptr(px(18)-1), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(w("Segoe UI Symbol"))))
	defer func() {
		for _, h := range []uintptr{normalFont, headingFont, smallFont, boldFont, heroFont, metricFont, symbolFont, bgBrush, sideBrush} {
			if h != 0 {
				pDeleteObject.Call(h)
			}
		}
	}()
	width, height := windowOuterForLogical(appearance.Width, appearance.Height)
	work := workArea
	workWidth, workHeight := int(work.Right-work.Left), int(work.Bottom-work.Top)
	x := int(work.Left) + max(0, (workWidth-width)/2)
	y := int(work.Top) + max(0, (workHeight-height)/2)
	mainWnd, _, _ = pCreateWindow.Call(wsExLayered, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(w("有盼"))), mainWindowStyle(), uintptr(x), uintptr(y), uintptr(width), uintptr(height), 0, 0, inst, 0)
	if mainWnd == 0 {
		message(0, "无法创建应用窗口。", 0x10)
		return
	}
	now := time.Now()
	calendarMonth = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	calendarSelected = civil(now)
	buildControls()
	var client rect
	if ok, _, _ := pGetClientRect.Call(mainWnd, uintptr(unsafe.Pointer(&client))); ok != 0 {
		layoutControls(logical(client.Right-client.Left), logical(client.Bottom-client.Top))
	}
	applyOpacity(appearance.Opacity)
	windowReady = true
	setTopmost(state.Pinned)
	addTray()
	pSetTimer.Call(mainWnd, 1, 30000, 0)
	pShowWindow.Call(mainWnd, swShow)
	pUpdateWindow.Call(mainWnd)
	checkReminders()
	var m msg
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		handled, _, _ := pIsDialogMessage.Call(mainWnd, uintptr(unsafe.Pointer(&m)))
		if handled == 0 {
			pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
		}
	}
}
