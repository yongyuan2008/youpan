package main

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

func TestNewlySavedNoteIsSelectedAndBodyIsPreserved(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	oldMain, oldPlanner, oldLayouts := mainWnd, planner, controlLayouts
	oldList, oldSearch, oldTitle, oldBody := noteList, noteSearch, noteTitle, noteBody
	oldID, oldSavedTitle, oldSavedBody, oldRefs := editingNoteID, noteSavedTitle, noteSavedBody, noteRefs
	defer func() {
		mainWnd, planner, controlLayouts = oldMain, oldPlanner, oldLayouts
		noteList, noteSearch, noteTitle, noteBody = oldList, oldSearch, oldTitle, oldBody
		editingNoteID, noteSavedTitle, noteSavedBody, noteRefs = oldID, oldSavedTitle, oldSavedBody, oldRefs
	}()
	hwnd, _, err := pCreateWindow.Call(0, uintptr(unsafe.Pointer(w("STATIC"))), 0, 0x80000000, 0, 0, 550, 450, 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatalf("create hidden note test window: %v", err)
	}
	defer pDestroyWindow.Call(hwnd)
	mainWnd = hwnd
	controlLayouts = map[uintptr]controlLayout{}
	noteList = listbox(0, 0, 200, 100, idNoteList)
	noteSearch = edit("", 0, 100, 200, 30, idNoteSearch, false)
	noteTitle = edit("", 220, 0, 150, 30, idNoteTitle, false)
	noteBody = edit("", 220, 40, 180, 180, idNoteBody, true)
	if noteList == 0 || noteSearch == 0 || noteTitle == 0 || noteBody == 0 {
		t.Fatal("create note test controls")
	}
	editingNoteID, noteSavedTitle, noteSavedBody, noteRefs = "", "", "", nil
	loaded, loadErr := loadPlannerPath(filepath.Join(t.TempDir(), "planner.json"))
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	planner = loaded
	first, saveErr := planner.UpsertNote(Note{Title: "旧笔记"}, time.Now())
	if saveErr != nil {
		t.Fatal(saveErr)
	}
	refreshNotes()
	pSendMessage.Call(noteList, lbSetCurSel, 0, 0)
	loadSelectedNote()
	if editingNoteID != first {
		t.Fatal("could not start editing the existing note")
	}
	newNote()
	if index := listSel(noteList); index != -1 || editingNoteID != "" {
		t.Fatal("new note kept the previous selection")
	}
	const body = "  记录\r\n\r\n"
	setText(noteTitle, "新笔记")
	setText(noteBody, body)
	saveNote()
	index := listSel(noteList)
	if index < 0 || index >= len(noteRefs) || noteRefs[index] != editingNoteID || editingNoteID == first {
		t.Fatal("selected note does not match the saved note")
	}
	for _, note := range planner.Notes {
		if note.ID == editingNoteID && note.Body == body {
			return
		}
	}
	t.Fatal("saved note did not retain its body")
}
