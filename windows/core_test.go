package main

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStorageAndValidation(t *testing.T) {
	p := filepath.Join(t.TempDir(), "app", "data.json")
	s, e := loadPath(p)
	if e != nil || len(s.Dates) != 3 {
		t.Fatalf("defaults: %+v %v", s, e)
	}
	if e = s.SetDates(Winter, nil); e != nil {
		t.Fatal(e)
	}
	if e = s.SetPinned(true); e != nil {
		t.Fatal(e)
	}
	s, e = loadPath(p)
	if e != nil || !s.Pinned {
		t.Fatalf("reload: %+v %v", s, e)
	}
	if _, ok := s.Dates[Winter]; ok {
		t.Fatal("cleared date returned")
	}
	for _, d := range []Dates{{Start: "2027-02-29"}, {Start: "2027-02-02", End: "2027-02-01"}, {Start: "2027-2-02"}} {
		if s.SetDates(Winter, &d) == nil {
			t.Fatalf("accepted %+v", d)
		}
	}
	d := Dates{Start: "2027-01-01", End: "2027-01-02"}
	if s.SetDates(Retirement, &d) == nil {
		t.Fatal("retirement end accepted")
	}
	if e = os.WriteFile(p, []byte("broken"), 0600); e != nil {
		t.Fatal(e)
	}
	bad, e := loadPath(p)
	if e == nil {
		t.Fatal("corruption accepted")
	}
	if bad.SetPinned(false) == nil {
		t.Fatal("overwrote unreadable data")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "broken" {
		t.Fatal("corrupt file overwritten")
	}
}
func TestDescribeCivilDates(t *testing.T) {
	loc, e := time.LoadLocation("America/New_York")
	if e != nil {
		t.Fatal(e)
	}
	s := State{Dates: map[Kind]Dates{Winter: {Start: "2027-03-13", End: "2027-03-15"}, Retirement: {Start: "2027-03-13"}}}
	cases := []struct {
		k    Kind
		day  int
		want string
	}{{Winter, 12, "距放假 1 天"}, {Winter, 13, "假期中 · 第 1 天"}, {Winter, 14, "假期中 · 第 2 天"}, {Winter, 15, "假期中 · 第 3 天"}, {Winter, 16, "假期已结束 1 天"}, {Retirement, 13, "今天 · 退休日"}, {Retirement, 14, "已退休 1 天"}}
	for _, c := range cases {
		got, _ := Describe(c.k, s, time.Date(2027, 3, c.day, 23, 0, 0, 0, loc))
		if got != c.want {
			t.Errorf("%s day %d got %q want %q", c.k, c.day, got, c.want)
		}
	}
	s.Dates[Summer] = Dates{Start: "2027-03-13"}
	got, _ := Describe(Summer, s, time.Date(2027, 3, 14, 0, 0, 0, 0, loc))
	if !strings.Contains(got, "开始日期已过 1 天") {
		t.Fatal(got)
	}
}
func TestAnnual(t *testing.T) {
	s := State{Dates: map[Kind]Dates{}}
	today := time.Date(2027, 5, 1, 12, 0, 0, 0, time.UTC)
	got, line := Describe(Labor, s, today)
	if got != "今天 · 节日当天" || line != "2027-05-01" {
		t.Fatal(got, line)
	}
	_, line = Describe(Labor, s, today.AddDate(0, 0, 1))
	if line != "2028-05-01" {
		t.Fatal(line)
	}
	d := Dates{Start: "2027-05-03", End: "2027-05-05"}
	s.Dates[Labor] = d
	_, line = Describe(Labor, s, today)
	if line != "2027-05-03 至 2027-05-05" {
		t.Fatal(line)
	}
}

func TestStorageOverrideKeepsTestsIsolated(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIZI_COUNTDOWN_TEST_DIRECTORY", dir)
	p, err := storagePath()
	if err != nil || p != filepath.Join(dir, "data.json") {
		t.Fatalf("countdown path=%q err=%v", p, err)
	}
	pp, err := plannerPath()
	if err != nil || pp != filepath.Join(dir, "planner.json") {
		t.Fatalf("planner path=%q err=%v", pp, err)
	}
	ap, err := appearancePath()
	if err != nil || ap != filepath.Join(dir, "appearance.json") {
		t.Fatalf("appearance path=%q err=%v", ap, err)
	}
}

func TestAppearanceReloadMultipleNamesAndLegacyStores(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "appearance.json")
	a, err := loadAppearancePath(p)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.SetWindow(1100, 900); err != nil {
		t.Fatal(err)
	}
	if err = a.SetOpacity(.73); err != nil {
		t.Fatal(err)
	}
	if err = a.SetNames(map[Kind]string{Retirement: "自由日", Winter: "冰雪季", Summer: "海边季"}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := loadAppearancePath(p)
	if err != nil || reloaded.Width != 1100 || reloaded.Height != 900 || reloaded.Opacity != .73 || reloaded.Names[Winter] != "冰雪季" {
		t.Fatalf("reload=%+v err=%v", reloaded, err)
	}
	// 2.0 data files remain valid and unchanged by appearance writes.
	dataPath := filepath.Join(dir, "data.json")
	legacy := []byte(`{"dates":{"retirement":{"start":"2046-01-10"}},"pinned":true}`)
	if err = os.WriteFile(dataPath, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := loadPath(dataPath)
	if err != nil || !s.Pinned || s.Dates[Retirement].Start != "2046-01-10" {
		t.Fatalf("legacy state=%+v err=%v", s, err)
	}
	after, _ := os.ReadFile(dataPath)
	if string(after) != string(legacy) {
		t.Fatal("appearance write modified legacy data schema")
	}
}

func TestAppearanceNameUnicodeWhitespaceAndLength(t *testing.T) {
	a, _ := loadAppearancePath(filepath.Join(t.TempDir(), "appearance.json"))
	if err := a.SetNames(map[Kind]string{Retirement: "  自由日  "}); err != nil {
		t.Fatal(err)
	}
	if got, custom := a.Name(Retirement); got != "自由日" || !custom {
		t.Fatalf("name=%q custom=%v", got, custom)
	}
	for _, name := range []string{"", "   ", strings.Repeat("界", 25)} {
		if err := a.SetNames(map[Kind]string{Winter: name}); err == nil {
			t.Fatalf("accepted invalid name %q", name)
		}
	}
	if err := a.SetNames(map[Kind]string{Winter: strings.Repeat("界", 24)}); err != nil {
		t.Fatal(err)
	}
}

func TestAppearanceBoundsAndMalformedProtection(t *testing.T) {
	p := filepath.Join(t.TempDir(), "appearance.json")
	a, _ := loadAppearancePath(p)
	for _, size := range [][2]int{{759, 700}, {760, 699}, {3841, 700}, {760, 2161}} {
		if err := a.SetWindow(size[0], size[1]); err == nil {
			t.Fatalf("accepted size %v", size)
		}
	}
	for _, opacity := range []float64{.49, 1.01, 0.0, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if err := a.SetOpacity(opacity); err == nil {
			t.Fatalf("accepted opacity %v", opacity)
		}
	}
	if err := a.SetOpacity(.5); err != nil {
		t.Fatal(err)
	}
	broken := []byte(`{"width":760,"height":700,"opacity":NaN,"names":{}}`)
	if err := os.WriteFile(p, broken, 0600); err != nil {
		t.Fatal(err)
	}
	blocked, err := loadAppearancePath(p)
	if err == nil || !blocked.blocked {
		t.Fatal("malformed appearance accepted")
	}
	if err = blocked.SetOpacity(.8); err == nil {
		t.Fatal("blocked appearance allowed overwrite")
	}
	after, _ := os.ReadFile(p)
	if string(after) != string(broken) {
		t.Fatal("malformed appearance was overwritten")
	}
}

func TestAppearanceSaveFailureDoesNotMutateMemory(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(parent, []byte("occupied"), 0600); err != nil {
		t.Fatal(err)
	}
	a := defaultAppearance()
	a.path = filepath.Join(parent, "appearance.json")
	if err := a.SetOpacity(.75); err == nil {
		t.Fatal("save unexpectedly succeeded")
	}
	if a.Opacity != 1 {
		t.Fatalf("failed save claimed in-memory success: %v", a.Opacity)
	}
}

func TestCustomNameDescriptionDoesNotChangeDates(t *testing.T) {
	s := State{Dates: map[Kind]Dates{Retirement: {Start: "2027-03-13"}, Winter: {Start: "2027-03-13", End: "2027-03-15"}}}
	before := s.Dates[Retirement]
	got, line := DescribeNamed(Retirement, s, time.Date(2027, 3, 14, 12, 0, 0, 0, time.UTC), "自由日", true)
	if got != "自由日已结束 1 天" || line != "2027-03-13" {
		t.Fatalf("got=%q line=%q", got, line)
	}
	got, _ = DescribeNamed(Winter, s, time.Date(2027, 3, 14, 12, 0, 0, 0, time.UTC), "冰雪季", true)
	if got != "冰雪季进行中 · 第 2 天" || strings.Contains(got, "假期") {
		t.Fatal(got)
	}
	if s.Dates[Retirement] != before {
		t.Fatal("renaming description changed stored dates")
	}
}

func TestPlannerTaskCRUDRecurrenceAndCompletion(t *testing.T) {
	p, err := loadPlannerPath(filepath.Join(t.TempDir(), "planner.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2027, 3, 1, 9, 0, 0, 0, time.Local) // Monday
	daily, err := p.UpsertTask(Task{Title: "每日复盘", Date: "2027-03-01", Repeat: RepeatDaily, Priority: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	weekdays, err := p.UpsertTask(Task{Title: "工作日计划", Date: "2027-03-01", Repeat: RepeatWeekdays}, now)
	if err != nil {
		t.Fatal(err)
	}
	weekly, err := p.UpsertTask(Task{Title: "周会", Date: "2027-03-01", Repeat: RepeatWeekly}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(TasksForDate(p.Tasks, "2027-03-06")) != 1 { // Saturday: daily only
		t.Fatalf("Saturday occurrences: %+v", TasksForDate(p.Tasks, "2027-03-06"))
	}
	if got := len(TasksForDate(p.Tasks, "2027-03-08")); got != 3 {
		t.Fatalf("next Monday occurrences=%d", got)
	}
	if err := p.SetTaskCompleted(daily, "2027-03-08", true); err != nil {
		t.Fatal(err)
	}
	var dailyTask Task
	for _, task := range p.Tasks {
		if task.ID == daily {
			dailyTask = task
		}
	}
	if !IsTaskCompleted(dailyTask, "2027-03-08") || IsTaskCompleted(dailyTask, "2027-03-09") {
		t.Fatal("recurring completion leaked across occurrences")
	}
	if err := p.DeleteTask(weekdays); err != nil {
		t.Fatal(err)
	}
	if err := p.DeleteTask(weekly); err != nil {
		t.Fatal(err)
	}
	reloaded, err := loadPlannerPath(p.path)
	if err != nil || len(reloaded.Tasks) != 1 || reloaded.Tasks[0].ID != daily {
		t.Fatalf("reload=%+v err=%v", reloaded.Tasks, err)
	}
}

func TestUnscheduledAndRecurringValidation(t *testing.T) {
	p, _ := loadPlannerPath(filepath.Join(t.TempDir(), "planner.json"))
	now := time.Date(2027, 3, 2, 12, 0, 0, 0, time.Local)
	id, err := p.UpsertTask(Task{Title: "以后再安排", Repeat: RepeatNone}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(TasksForDate(p.Tasks, "2027-03-02")) != 0 {
		t.Fatal("unscheduled task appeared on calendar")
	}
	if err := p.SetTaskCompleted(id, "", true); err != nil {
		t.Fatal(err)
	}
	if !IsTaskCompleted(p.Tasks[0], "") {
		t.Fatal("unscheduled completion not persisted")
	}
	for _, repeat := range []Repeat{RepeatDaily, RepeatWeekdays, RepeatWeekly} {
		if _, err := p.UpsertTask(Task{Title: "无锚点重复任务", Repeat: repeat}, now); err == nil {
			t.Fatalf("%s task without date accepted", repeat)
		}
	}
	if _, err := p.UpsertTask(Task{Title: "无日期提醒", ReminderTime: "09:00", Repeat: RepeatNone}, now); err == nil {
		t.Fatal("unscheduled reminder accepted")
	}
	if _, err := p.UpsertTask(Task{Title: "非 ASCII 时间", Date: "2027-03-02", ReminderTime: "０9:00", Repeat: RepeatNone}, now); err == nil {
		t.Fatal("non-ASCII reminder time accepted")
	}
}

func TestReminderCatchUpDedupAndCompletionSuppression(t *testing.T) {
	p, _ := loadPlannerPath(filepath.Join(t.TempDir(), "planner.json"))
	now := time.Date(2027, 3, 2, 10, 30, 0, 0, time.Local)
	id, err := p.UpsertTask(Task{Title: "准时服药", Date: "2027-03-02", ReminderTime: "09:00", Repeat: RepeatDaily}, now)
	if err != nil {
		t.Fatal(err)
	}
	due := p.DueReminders(now)
	if len(due) != 1 || due[0].TaskID != id {
		t.Fatalf("due=%+v", due)
	}
	if err := p.MarkRemindersFired(due); err != nil {
		t.Fatal(err)
	}
	if got := p.DueReminders(now.Add(15 * time.Minute)); len(got) != 0 {
		t.Fatalf("dedup failed: %+v", got)
	}
	if got := p.DueReminders(now.AddDate(0, 0, 1)); len(got) != 1 {
		t.Fatalf("next occurrence should remind once: %+v", got)
	}
	if err := p.SetTaskCompleted(id, "2027-03-03", true); err != nil {
		t.Fatal(err)
	}
	if got := p.DueReminders(now.AddDate(0, 0, 1)); len(got) != 0 {
		t.Fatalf("completed reminder not suppressed: %+v", got)
	}
	if got := p.DueReminders(now.AddDate(0, 0, -1)); len(got) != 0 {
		t.Fatalf("older recurrence should not be caught up: %+v", got)
	}
}

func TestNoteToTaskLinkPreservesNote(t *testing.T) {
	p, _ := loadPlannerPath(filepath.Join(t.TempDir(), "planner.json"))
	now := time.Date(2027, 3, 2, 10, 30, 0, 0, time.Local)
	noteID, err := p.UpsertNote(Note{Title: "准备家长会", Body: "列出三个讨论要点"}, now)
	if err != nil {
		t.Fatal(err)
	}
	taskID, err := p.CreateTaskFromNote(noteID, "2027-03-05", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Notes) != 1 || p.Notes[0].ID != noteID {
		t.Fatal("source note was removed or changed")
	}
	if len(p.Tasks) != 1 || p.Tasks[0].ID != taskID || p.Tasks[0].SourceNoteID != noteID || p.Tasks[0].Details != p.Notes[0].Body {
		t.Fatalf("linked task=%+v note=%+v", p.Tasks, p.Notes)
	}
}

func TestMonthGridMondayFirst(t *testing.T) {
	grid := MonthGrid(time.Date(2027, 8, 15, 0, 0, 0, 0, time.UTC), time.Date(2027, 8, 3, 12, 0, 0, 0, time.Local))
	if len(grid) != 42 || grid[0].Date != "2027-07-26" || grid[6].Date != "2027-08-01" {
		t.Fatalf("unexpected grid: first=%+v sunday=%+v", grid[0], grid[6])
	}
	if !grid[8].IsToday || grid[8].Date != "2027-08-03" {
		t.Fatalf("today cell=%+v", grid[8])
	}
}

func TestTaskListSearchAndSevenDayBoundary(t *testing.T) {
	tasks := []Task{
		{ID: "daily", Title: "晨间复盘", Details: "记录睡眠与状态", Date: "2027-12-30", Repeat: RepeatDaily},
		{ID: "weekly", Title: "项目周会", Details: "产品路线", Date: "2027-12-29", Repeat: RepeatWeekly},
		{ID: "once", Title: "提交报告", Details: "年度总结", Date: "2028-01-04", Repeat: RepeatNone},
		{ID: "outside", Title: "第八天事项", Date: "2028-01-06", Repeat: RepeatNone},
		{ID: "unscheduled", Title: "以后安排", Details: "睡眠资料", Repeat: RepeatNone},
	}
	got := SelectTaskListOccurrences(tasks, "2027-12-30", 7, "睡眠", false, false)
	if len(got) != 7 || got[0].Date != "2027-12-30" || got[6].Date != "2028-01-05" {
		t.Fatalf("detail search/year boundary: %+v", got)
	}
	for _, item := range got {
		if item.TaskID != "daily" {
			t.Fatalf("search admitted unrelated task: %+v", item)
		}
	}
	all := SelectTaskListOccurrences(tasks, "2027-12-30", 7, "", false, false)
	for _, item := range all {
		if item.TaskID == "unscheduled" || item.TaskID == "outside" {
			t.Fatalf("future view admitted unscheduled/out-of-range item: %+v", item)
		}
	}
	foundLastDay := false
	for _, item := range all {
		if item.TaskID == "once" && item.Date == "2028-01-04" && item.DayOffset == 5 {
			foundLastDay = true
		}
	}
	if !foundLastDay {
		t.Fatalf("cross-year occurrence missing: %+v", all)
	}
}

func TestTaskListTodayAndOccurrenceIndependence(t *testing.T) {
	tasks := []Task{
		{ID: "overdue", Title: "补交材料", Date: "2027-01-31", Repeat: RepeatNone},
		{ID: "weekdays", Title: "工作记录", Date: "2027-01-30", Repeat: RepeatWeekdays, CompletedDates: []string{"2027-02-01"}},
		{ID: "weekly", Title: "每周整理", Date: "2027-01-25", Repeat: RepeatWeekly},
		{ID: "unscheduled", Title: "待安排", Repeat: RepeatNone},
	}
	got := SelectTaskListOccurrences(tasks, "2027-02-01", 1, "", true, true)
	if len(got) != 4 {
		t.Fatalf("today selection=%+v", got)
	}
	categories := map[string]string{}
	for _, item := range got {
		categories[item.TaskID] = item.Category
	}
	if categories["overdue"] != "overdue" || categories["unscheduled"] != "unscheduled" || categories["weekdays"] != "scheduled" {
		t.Fatalf("today categories=%+v", categories)
	}
	future := SelectTaskListOccurrences(tasks, "2027-02-01", 7, "工作", false, false)
	if len(future) != 5 { // Monday through Friday only.
		t.Fatalf("weekday occurrences=%+v", future)
	}
	var weekday Task
	for _, task := range tasks {
		if task.ID == "weekdays" {
			weekday = task
		}
	}
	if !IsTaskCompleted(weekday, "2027-02-01") || IsTaskCompleted(weekday, "2027-02-02") {
		t.Fatal("completion leaked between selected future occurrences")
	}
}

func TestPlannerOldCompatibilityAndMalformedProtection(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "planner.json")
	legacy := map[string]any{"tasks": []any{}, "notes": []any{}}
	b, _ := json.Marshal(legacy)
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadPlannerPath(p)
	if err != nil || loaded.Version != 1 {
		t.Fatalf("old compatible file: %+v %v", loaded, err)
	}
	broken := []byte(`{"version":1,"tasks":[{"id":"x","title":"","repeat":"none"}],"notes":[]}`)
	if err := os.WriteFile(p, broken, 0600); err != nil {
		t.Fatal(err)
	}
	blocked, err := loadPlannerPath(p)
	if err == nil || !blocked.blocked {
		t.Fatal("malformed planner accepted")
	}
	if _, err := blocked.UpsertTask(Task{Title: "不得覆盖", Repeat: RepeatNone}, time.Now()); err == nil {
		t.Fatal("blocked store allowed write")
	}
	after, _ := os.ReadFile(p)
	if string(after) != string(broken) {
		t.Fatal("malformed source was overwritten")
	}
}

func TestPlannerRecoversInterruptedBackup(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "planner.json")
	original := defaultPlanner()
	original.Tasks = []Task{{ID: "keep", Title: "不能丢失", Date: "2027-03-02", Repeat: RepeatNone}}
	b, _ := json.Marshal(original)
	if err := os.WriteFile(p+".bak", b, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadPlannerPath(p)
	if err != nil || len(loaded.Tasks) != 1 || loaded.Tasks[0].ID != "keep" {
		t.Fatalf("backup recovery=%+v err=%v", loaded.Tasks, err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("main planner not restored: %v", err)
	}
	if _, err := os.Stat(p + ".bak"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("backup still present: %v", err)
	}
}
