package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppearanceRecoversBackupAndKeepsDesktopSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "appearance.json")
	legacy := []byte(`{"width":760,"height":700,"opacity":0.75,"names":{"retirement":"旅行"}}`)
	if err := os.WriteFile(path+".bak", legacy, 0600); err != nil {
		t.Fatal(err)
	}
	a, err := loadAppearancePath(path)
	if err != nil || a.Opacity != .75 || a.Names[Retirement] != "旅行" || a.DesktopPinned {
		t.Fatalf("legacy backup = %+v, %v", a, err)
	}
	if err := a.SetDesktopPinned(true); err != nil {
		t.Fatal(err)
	}
	if err := a.SetDesktopPosition(-440, 88); err != nil {
		t.Fatal(err)
	}
	if err := a.SetWindow(minWindowWidth, minWindowHeight); err != nil {
		t.Fatal(err)
	}
	reloaded, err := loadAppearancePath(path)
	if err != nil || !reloaded.DesktopPinned || reloaded.DesktopPosition == nil ||
		*reloaded.DesktopPosition != (DesktopPosition{X: -440, Y: 88}) ||
		reloaded.Opacity != .75 || reloaded.Width != minWindowWidth || reloaded.Height != minWindowHeight {
		t.Fatalf("reloaded appearance = %+v, %v", reloaded, err)
	}
	if err := reloaded.ResetVisuals(); err != nil {
		t.Fatal(err)
	}
	if reloaded.DesktopPinned || reloaded.Opacity != 1 || reloaded.Names[Retirement] != "旅行" {
		t.Fatalf("reset changed the wrong appearance settings: %+v", reloaded)
	}
}

func TestPlannerRejectsNonPlannerDocumentsAndPreservesNoteBody(t *testing.T) {
	for _, damaged := range []string{
		`null`,
		`{}`,
		`{"version":1,"tasks":[]}`,
		`{"schemaVersion":1,"tasks":[],"notes":[{"id":"mac1","title":"Mac 笔记","body":"内容"}],"deliveredReminderKeys":[]}`,
	} {
		path := filepath.Join(t.TempDir(), "planner.json")
		if err := os.WriteFile(path, []byte(damaged), 0600); err != nil {
			t.Fatal(err)
		}
		p, err := loadPlannerPath(path)
		if err == nil {
			t.Fatalf("accepted planner data %s", damaged)
		}
		if _, err = p.UpsertNote(Note{Title: "不能覆盖"}, time.Now()); err == nil {
			t.Fatal("saved over rejected planner")
		}
		b, err := os.ReadFile(path)
		if err != nil || string(b) != damaged {
			t.Fatalf("rejected file changed: %s, %v", b, err)
		}
	}
	path := filepath.Join(t.TempDir(), "planner.json")
	p, err := loadPlannerPath(path)
	if err != nil {
		t.Fatal(err)
	}
	const body = "  准备\r\n\r\n"
	if _, err = p.UpsertNote(Note{Title: "  工作记录  ", Body: body}, time.Now()); err != nil {
		t.Fatal(err)
	}
	reloaded, err := loadPlannerPath(path)
	if err != nil || len(reloaded.Notes) != 1 || reloaded.Notes[0].Title != "工作记录" || reloaded.Notes[0].Body != body {
		t.Fatalf("reloaded notes = %+v, %v", reloaded.Notes, err)
	}
}

func TestCalendarCountMatchesSortedOccurrences(t *testing.T) {
	const date = "2027-03-08"
	tasks := []Task{
		{ID: "later", Title: "b", Date: date, Repeat: RepeatNone},
		{ID: "star", Title: "z", Date: "2027-03-01", Repeat: RepeatWeekly, Priority: true},
		{ID: "first", Title: "a", Date: "2027-03-05", Repeat: RepeatWeekdays},
		{ID: "future", Title: "c", Date: "2027-03-09", Repeat: RepeatDaily},
	}
	matching := TasksForDate(tasks, date)
	if countTasksForDate(tasks, date) != 3 || len(matching) != 3 ||
		matching[0].TaskID != "star" || matching[1].TaskID != "first" || matching[2].TaskID != "later" {
		t.Fatalf("unexpected calendar occurrences: %+v", matching)
	}
}
