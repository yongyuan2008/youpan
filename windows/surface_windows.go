//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"
)

var (
	pDrawText         = u32.NewProc("DrawTextW")
	pSaveDC           = g32.NewProc("SaveDC")
	pRestoreDC        = g32.NewProc("RestoreDC")
	pSelectObject     = g32.NewProc("SelectObject")
	pSetBkMode        = g32.NewProc("SetBkMode")
	pCreateRoundRgn   = g32.NewProc("CreateRoundRectRgn")
	pExtSelectClipRgn = g32.NewProc("ExtSelectClipRgn")
	pPolygon          = g32.NewProc("Polygon")
	pEllipse          = g32.NewProc("Ellipse")
	pGetStockObject   = g32.NewProc("GetStockObject")
	pGradientFill     = syscall.NewLazyDLL("msimg32.dll").NewProc("GradientFill")
)

const bsOwnerDraw = 0x000b

type drawItem struct {
	ControlType, ControlID, ItemID, Action, State uint32
	Hwnd, Hdc                                     uintptr
	Rect                                          rect
	Data                                          uintptr
}

type gradientVertex struct {
	X, Y                    int32
	Red, Green, Blue, Alpha uint16
}

type calendarDayPaint struct {
	number                                       string
	inMonth, today, selected, hasTasks, hasBadge bool
}

var (
	navNames  = [...]string{"倒计时", "今天", "日历", "笔记", "设置"}
	navIcons  = [...]string{"◷", "✓", "▦", "▤", "⚙"}
	cardIcons = [...]string{"◷", "❄", "☀", "♧", "⚑"}
	textInk   = rgb(31, 41, 55)
	textMuted = rgb(94, 105, 123)
	accentInk = rgb(30, 79, 161)
)

func inset(r rect, x, y int32) rect {
	return rect{r.Left + x, r.Top + y, r.Right - x, r.Bottom - y}
}

func area(r rect, left, top, right, bottom int) rect {
	return rect{r.Left + int32(px(left)), r.Top + int32(px(top)), r.Right - int32(px(right)), r.Top + int32(px(bottom))}
}

func surface(dc uintptr, r rect, radius int, color uintptr) {
	clip, _, _ := pCreateRoundRgn.Call(uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), px(radius*2), px(radius*2))
	if clip == 0 {
		return
	}
	defer pDeleteObject.Call(clip)
	saved, _, _ := pSaveDC.Call(dc)
	if saved == 0 {
		return
	}
	defer pRestoreDC.Call(dc, saved)
	pExtSelectClipRgn.Call(dc, clip, 1) // RGN_AND
	fillColor(dc, r, color)
}

func fillColor(dc uintptr, r rect, color uintptr) {
	brush, _, _ := pCreateSolidBrush.Call(color)
	if brush != 0 {
		pFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), brush)
		pDeleteObject.Call(brush)
	}
}

func surfaceText(dc uintptr, r rect, text string, face, color, align uintptr) {
	if text == "" || r.Left >= r.Right || r.Top >= r.Bottom {
		return
	}
	pSelectObject.Call(dc, face)
	pSetTextColor.Call(dc, color)
	pSetBkMode.Call(dc, 1) // TRANSPARENT
	pDrawText.Call(dc, uintptr(unsafe.Pointer(w(text))), ^uintptr(0), uintptr(unsafe.Pointer(&r)), align|0x4|0x20|0x800|0x8000)
}

func solidShape(dc uintptr, color uintptr, draw func()) {
	brush, _, _ := pCreateSolidBrush.Call(color)
	if brush == 0 {
		return
	}
	previousBrush, _, _ := pSelectObject.Call(dc, brush)
	nullPen, _, _ := pGetStockObject.Call(8)
	previousPen, _, _ := pSelectObject.Call(dc, nullPen)
	draw()
	pSelectObject.Call(dc, previousPen)
	pSelectObject.Call(dc, previousBrush)
	pDeleteObject.Call(brush)
}

func heroArtwork(dc uintptr, r rect) {
	origin := point{r.Right - int32(px(154)), r.Top + int32(px(18))}
	p := func(x, y int) point { return point{origin.X + int32(px(x)), origin.Y + int32(px(y))} }
	solidShape(dc, rgb(174, 204, 250), func() {
		a, b := p(87, 0), p(118, 31)
		pEllipse.Call(dc, uintptr(a.X), uintptr(a.Y), uintptr(b.X), uintptr(b.Y))
	})
	solidShape(dc, rgb(120, 159, 227), func() {
		mountains := [...]point{p(0, 61), p(44, 23), p(87, 55), p(136, 35), p(136, 77), p(0, 77)}
		pPolygon.Call(dc, uintptr(unsafe.Pointer(&mountains[0])), uintptr(len(mountains)))
	})
	solidShape(dc, rgb(87, 131, 210), func() {
		mountains := [...]point{p(0, 72), p(41, 42), p(76, 64), p(136, 44), p(136, 78), p(0, 78)}
		pPolygon.Call(dc, uintptr(unsafe.Pointer(&mountains[0])), uintptr(len(mountains)))
	})
}

func heroSurface(dc uintptr, r rect, pressed bool) {
	surface(dc, r, 14, rgb(35, 74, 163))
	clip, _, _ := pCreateRoundRgn.Call(uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), px(28), px(28))
	if clip == 0 {
		return
	}
	defer pDeleteObject.Call(clip)
	saved, _, _ := pSaveDC.Call(dc)
	if saved == 0 {
		return
	}
	defer pRestoreDC.Call(dc, saved)
	pExtSelectClipRgn.Call(dc, clip, 1)
	start := uint16(36)
	if pressed {
		start = 25
	}
	vertices := [...]gradientVertex{
		{X: r.Left, Y: r.Top, Red: start << 8, Green: 66 << 8, Blue: 156 << 8},
		{X: r.Right, Y: r.Bottom, Red: 68 << 8, Green: 119 << 8, Blue: 216 << 8},
	}
	mesh := [...]uint32{0, 1}
	pGradientFill.Call(dc, uintptr(unsafe.Pointer(&vertices[0])), 2, uintptr(unsafe.Pointer(&mesh[0])), 1, 0)
	if r.Right-r.Left >= int32(px(430)) {
		heroArtwork(dc, r)
	}
}

func drawSurface(lp uintptr) bool {
	if lp == 0 {
		return false
	}
	var item drawItem
	pCopyMemory.Call(uintptr(unsafe.Pointer(&item)), lp, unsafe.Sizeof(item))
	id := int(item.ControlID)
	countdown := id >= idCountdownBase && id < idCountdownBase+len(countdownButtons)
	calendarDay := id >= idCalDayBase && id < idCalDayBase+len(calendarDays)
	navigation := id >= idNavCountdown && id <= idNavNotes || id == idSettings
	if item.ControlType != 4 || !countdown && !navigation && !calendarDay { // ODT_BUTTON
		return false
	}
	r, dc := item.Rect, item.Hdc
	saved, _, _ := pSaveDC.Call(dc)
	if saved == 0 {
		return false
	}
	defer pRestoreDC.Call(dc, saved)
	pressed := item.State&0x1 != 0
	focused := item.State&0x10 != 0 && item.State&0x200 == 0
	if calendarDay {
		paintCalendarDay(dc, r, calendarDaysPaint[id-idCalDayBase], pressed, focused)
		return true
	}
	if navigation {
		i := id - idNavCountdown
		if id == idSettings {
			i = len(navNames) - 1
		}
		pFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), sideBrush)
		active := settingsOpen && id == idSettings || !settingsOpen && i == view
		color := textInk
		if active || pressed {
			bg := rgb(214, 228, 249)
			if pressed {
				bg = rgb(198, 217, 246)
			}
			surface(dc, r, 8, bg)
			color = accentInk
		}
		surfaceText(dc, area(r, 9, 0, 0, 38), navIcons[i], symbolFont, color, 0)
		surfaceText(dc, area(r, 32, 0, 6, 38), navNames[i], normalFont, color, 0)
		if focused {
			fillColor(dc, area(r, 3, 10, 106, 28), accentInk)
		}
		return true
	}

	i := id - idCountdownBase
	c := cardStatus[i]
	name, _ := customName(Events[i])
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), bgBrush)
	if i == 0 {
		heroSurface(dc, r, pressed)
		value, caption := c.Metric, c.Caption
		if c.Date == "" || c.Caption == "今天" {
			value, caption = c.Caption, name
		}
		surfaceText(dc, area(r, 19, 12, 160, 34), caption, boldFont, rgb(222, 234, 255), 0)
		surfaceText(dc, area(r, 19, 34, 160, 85), value, heroFont, rgb(255, 255, 255), 0)
		surfaceText(dc, area(r, 20, 88, 160, 111), c.Date, smallFont, rgb(222, 234, 255), 0)
		if focused {
			fillColor(dc, area(r, 20, 80, 20, 82), rgb(202, 221, 253))
		}
		return true
	}

	surface(dc, r, 10, rgb(224, 231, 241))
	fill := rgb(255, 255, 255)
	if pressed {
		fill = rgb(230, 240, 255)
	}
	surface(dc, inset(r, 1, 1), 10, fill)
	badge := area(r, 14, 13, 0, 49)
	badge.Right = badge.Left + int32(px(36))
	surface(dc, badge, 9, rgb(230, 239, 253))
	surfaceText(dc, badge, cardIcons[i], symbolFont, accentInk, 1)
	surfaceText(dc, area(r, 62, 9, 150, 33), name, boldFont, textInk, 0)
	surfaceText(dc, area(r, 62, 34, 150, 54), c.Date, smallFont, textMuted, 0)
	value, caption := c.Metric, c.Caption
	if c.Date == "" || c.Caption == "今天" {
		value, caption = c.Caption, ""
	}
	metric := area(r, 0, 6, 16, 35)
	metric.Left = metric.Right - int32(px(126))
	surfaceText(dc, metric, value, metricFont, textInk, 2)
	metric.Top, metric.Bottom = r.Top+int32(px(35)), r.Top+int32(px(54))
	surfaceText(dc, metric, strings.TrimSpace(caption), smallFont, textMuted, 2)
	if focused {
		fillColor(dc, area(r, 4, 15, int(float64(r.Right-r.Left)/scale)-6, 47), accentInk)
	}
	return true
}

func paintCalendarDay(dc uintptr, r rect, day calendarDayPaint, pressed, focused bool) {
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), bgBrush)
	fill, color, face := rgb(250, 252, 255), textInk, normalFont
	if !day.inMonth {
		fill, color = rgb(244, 247, 251), textMuted
	}
	if day.selected || pressed {
		fill, color, face = rgb(214, 228, 249), accentInk, boldFont
	}
	if day.today || focused {
		surface(dc, r, 8, rgb(116, 154, 215))
		r = inset(r, 1, 1)
		color = accentInk
	}
	surface(dc, r, 8, fill)
	surfaceText(dc, area(r, 0, 3, 0, 30), day.number, face, color, 1)
	center := (r.Left + r.Right) / 2
	top := r.Top + int32(px(31))
	dot := func(x int32, color uintptr) {
		d := int32(max(1, int(px(4))))
		surface(dc, rect{x, top, x + d, top + d}, 2, color)
	}
	if day.hasTasks && day.hasBadge {
		dot(center-int32(px(5)), accentInk)
		dot(center+int32(px(2)), rgb(190, 104, 22))
	} else if day.hasTasks {
		dot(center-int32(px(2)), accentInk)
	} else if day.hasBadge {
		dot(center-int32(px(2)), rgb(190, 104, 22))
	}
}
