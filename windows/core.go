package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

type Kind string

const (
	Retirement Kind = "retirement"
	Winter     Kind = "winter"
	Summer     Kind = "summer"
	Labor      Kind = "labor"
	National   Kind = "national"
)

var Events = []Kind{Retirement, Winter, Summer, Labor, National}

func (k Kind) Title() string {
	switch k {
	case Retirement:
		return "退休"
	case Winter:
		return "寒假"
	case Summer:
		return "暑假"
	case Labor:
		return "五一假期"
	case National:
		return "十一假期"
	}
	return ""
}

type Dates struct {
	Start string `json:"start"`
	End   string `json:"end,omitempty"`
}
type State struct {
	Dates   map[Kind]Dates `json:"dates"`
	Pinned  bool           `json:"pinned"`
	path    string
	blocked bool
}

func defaultState() State {
	return State{Dates: map[Kind]Dates{Retirement: {Start: "2046-01-10"}, Winter: {Start: "2027-01-18", End: "2027-02-18"}, Summer: {Start: "2027-07-12", End: "2027-08-26"}}}
}
func storagePath() (string, error) {
	dir, err := storageDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "data.json"), nil
}

// RIZI_COUNTDOWN_TEST_DIRECTORY is deliberately a directory rather than a
// file. Tests and review builds can therefore isolate both stores without ever
// resolving the real user's LOCALAPPDATA directory.
func storageDirectory() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("RIZI_COUNTDOWN_TEST_DIRECTORY")); dir != "" {
		return dir, nil
	}
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(base, "DaysCountdown"), nil
}
func LoadState() (State, error) {
	p, err := storagePath()
	if err != nil {
		return State{blocked: true}, err
	}
	return loadPath(p)
}
func loadPath(p string) (State, error) {
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		s := defaultState()
		s.path = p
		return s, nil
	}
	if err != nil {
		return State{path: p, blocked: true}, err
	}
	var s State
	if err = json.Unmarshal(b, &s); err != nil {
		return State{path: p, blocked: true}, fmt.Errorf("读取保存的数据失败: %w", err)
	}
	if s.Dates == nil {
		return State{path: p, blocked: true}, errors.New("保存的数据缺少日期列表")
	}
	for k, d := range s.Dates {
		if err = validate(k, d); err != nil {
			return State{path: p, blocked: true}, fmt.Errorf("保存的数据无效: %w", err)
		}
	}
	s.path = p
	return s, nil
}
func validKind(k Kind) bool {
	return k == Retirement || k == Winter || k == Summer || k == Labor || k == National
}
func parseDate(s string) (time.Time, error) {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return time.Time{}, errors.New("请输入有效日期（YYYY-MM-DD）")
	}
	for i, c := range s {
		if i == 4 || i == 7 {
			continue
		}
		if c < '0' || c > '9' {
			return time.Time{}, errors.New("请输入有效日期（YYYY-MM-DD）")
		}
	}
	t, e := time.Parse("2006-01-02", s)
	if e != nil || t.Year() < 1 {
		return time.Time{}, errors.New("请输入有效日期（YYYY-MM-DD）")
	}
	return t, nil
}
func validate(k Kind, d Dates) error {
	if !validKind(k) {
		return errors.New("未知事项")
	}
	a, e := parseDate(d.Start)
	if e != nil {
		return e
	}
	if d.End != "" {
		b, e := parseDate(d.End)
		if e != nil {
			return e
		}
		if b.Before(a) {
			return errors.New("结束日期不能早于开始日期")
		}
		if k == Retirement {
			return errors.New("退休日期无需填写结束日期")
		}
	}
	return nil
}
func (s *State) save(next State) error {
	if s.blocked {
		return errors.New("原有数据读取失败，为避免覆盖，请先处理存储文件")
	}
	p := s.path
	if p == "" {
		var err error
		p, err = storagePath()
		if err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(p)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".data-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp, p); err != nil {
		return err
	}
	next.path = p
	*s = next
	return nil
}
func (s *State) SetDates(k Kind, d *Dates) error {
	if !validKind(k) {
		return errors.New("未知事项")
	}
	if d != nil {
		if err := validate(k, *d); err != nil {
			return err
		}
	}
	n := *s
	n.Dates = make(map[Kind]Dates, len(s.Dates))
	for k, v := range s.Dates {
		n.Dates[k] = v
	}
	if d == nil {
		delete(n.Dates, k)
	} else {
		n.Dates[k] = *d
	}
	return s.save(n)
}
func (s *State) SetPinned(p bool) error { n := *s; n.Pinned = p; return s.save(n) }
func civil(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
func annual(k Kind, now time.Time) string {
	month := time.May
	if k == National {
		month = time.October
	}
	today := civil(now)
	t := time.Date(today.Year(), month, 1, 0, 0, 0, 0, time.UTC)
	if t.Before(today) {
		t = t.AddDate(1, 0, 0)
	}
	return t.Format("2006-01-02")
}
func SuggestedStart(k Kind, s State, now time.Time) string {
	if d, ok := s.Dates[k]; ok {
		return d.Start
	}
	if k == Labor || k == National {
		return annual(k, now)
	}
	return civil(now).Format("2006-01-02")
}
func Describe(k Kind, s State, now time.Time) (string, string) {
	return DescribeNamed(k, s, now, k.Title(), false)
}

// DescribeNamed keeps the 2.0 wording for default names, while custom names
// use neutral status text so a renamed item never continues to say “退休” or
// “放假”.
func DescribeNamed(k Kind, s State, now time.Time, name string, customized bool) (string, string) {
	d, custom := s.Dates[k]
	if !custom {
		if k != Labor && k != National {
			if customized {
				return "待设置 · 设置" + name + "日期", ""
			}
			if k == Retirement {
				return "待设置 · 设置退休日期", ""
			}
			return "待设置 · 设置假期日期", ""
		}
		d.Start = annual(k, now)
	}
	if validate(k, d) != nil {
		return "待设置 · 日期无效，请重新设置", ""
	}
	start, _ := parseDate(d.Start)
	today := civil(now)
	delta := int((start.Unix() - today.Unix()) / 86400)
	line := d.Start
	if d.End != "" {
		line += " 至 " + d.End
	}
	if delta > 0 {
		if customized {
			return fmt.Sprintf("距%s %d 天", name, delta), line
		}
		caption := "距放假"
		if k == Retirement {
			caption = "距退休"
		} else if !custom {
			caption = "距节日"
		}
		return fmt.Sprintf("%s %d 天", caption, delta), line
	}
	if k == Retirement {
		if customized {
			if delta == 0 {
				return "今天 · " + name, line
			}
			return fmt.Sprintf("%s已结束 %d 天", name, -delta), line
		}
		if delta == 0 {
			return "今天 · 退休日", line
		}
		return fmt.Sprintf("已退休 %d 天", -delta), line
	}
	if d.End != "" {
		end, _ := parseDate(d.End)
		if !today.After(end) {
			if customized {
				return fmt.Sprintf("%s进行中 · 第 %d 天", name, -delta+1), line
			}
			return fmt.Sprintf("假期中 · 第 %d 天", -delta+1), line
		}
		if customized {
			return fmt.Sprintf("%s已结束 %d 天", name, int((today.Unix()-end.Unix())/86400)), line
		}
		return fmt.Sprintf("假期已结束 %d 天", int((today.Unix()-end.Unix())/86400)), line
	}
	if delta == 0 {
		if customized {
			return "今天 · " + name, line
		}
		if !custom {
			return "今天 · 节日当天", line
		}
		return "今天 · 假期开始日", line
	}
	if customized {
		return fmt.Sprintf("%s已结束 %d 天", name, -delta), line
	}
	return fmt.Sprintf("开始日期已过 %d 天", -delta), line
}

// Planner data intentionally lives beside, not inside, the compatible
// countdown data.json. This lets old builds continue to read dates and pinned
// state without seeing fields they do not understand.
type Repeat string

const (
	RepeatNone     Repeat = "none"
	RepeatDaily    Repeat = "daily"
	RepeatWeekdays Repeat = "weekdays"
	RepeatWeekly   Repeat = "weekly"
)

type Task struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Details        string   `json:"details,omitempty"`
	Date           string   `json:"date,omitempty"`
	CreatedDate    string   `json:"createdDate,omitempty"`
	ReminderTime   string   `json:"reminderTime,omitempty"`
	Repeat         Repeat   `json:"repeat"`
	Priority       bool     `json:"priority,omitempty"`
	CompletedDates []string `json:"completedDates,omitempty"`
	SourceNoteID   string   `json:"sourceNoteID,omitempty"`
}

type Note struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type Planner struct {
	Version       int             `json:"version"`
	Tasks         []Task          `json:"tasks"`
	Notes         []Note          `json:"notes"`
	ReminderFired map[string]bool `json:"reminderFired,omitempty"`
	path          string
	blocked       bool
}

type TaskOccurrence struct {
	TaskID string
	Date   string
}

type TaskListOccurrence struct {
	TaskID    string
	Date      string
	DayOffset int
	Category  string
}

type DueReminder struct {
	TaskID     string
	Occurrence string
	Title      string
	Time       string
	Key        string
}

type CalendarCell struct {
	Date      string
	Day       int
	InMonth   bool
	IsToday   bool
	WeekIndex int
}

var idCounter uint64

func newID(prefix string, now time.Time) string {
	return fmt.Sprintf("%s-%x-%x", prefix, now.UnixNano(), atomic.AddUint64(&idCounter, 1))
}

func plannerPath() (string, error) {
	dir, err := storageDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "planner.json"), nil
}

func defaultPlanner() Planner {
	return Planner{Version: 1, Tasks: []Task{}, Notes: []Note{}, ReminderFired: map[string]bool{}}
}

func LoadPlanner() (Planner, error) {
	p, err := plannerPath()
	if err != nil {
		return Planner{blocked: true}, err
	}
	return loadPlannerPath(p)
}

func loadPlannerPath(p string) (Planner, error) {
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		// A prior save may have been interrupted after moving the old file aside.
		// Never interpret that recoverable state as an empty new planner.
		backup := p + ".bak"
		if bb, backupErr := os.ReadFile(backup); backupErr == nil {
			if renameErr := os.Rename(backup, p); renameErr != nil {
				return Planner{path: p, blocked: true}, fmt.Errorf("恢复计划数据备份失败: %w", renameErr)
			}
			b = bb
			err = nil
		} else if !errors.Is(backupErr, os.ErrNotExist) {
			return Planner{path: p, blocked: true}, fmt.Errorf("读取计划数据备份失败: %w", backupErr)
		}
	}
	if errors.Is(err, os.ErrNotExist) {
		v := defaultPlanner()
		v.path = p
		return v, nil
	}
	if err != nil {
		return Planner{path: p, blocked: true}, err
	}
	var v Planner
	if err := json.Unmarshal(b, &v); err != nil {
		return Planner{path: p, blocked: true}, fmt.Errorf("读取计划数据失败: %w", err)
	}
	if v.Version == 0 {
		v.Version = 1
	}
	if v.Version != 1 {
		return Planner{path: p, blocked: true}, fmt.Errorf("不支持的计划数据版本: %d", v.Version)
	}
	if v.Tasks == nil {
		v.Tasks = []Task{}
	}
	if v.Notes == nil {
		v.Notes = []Note{}
	}
	if v.ReminderFired == nil {
		v.ReminderFired = map[string]bool{}
	}
	if err := validatePlanner(v); err != nil {
		return Planner{path: p, blocked: true}, fmt.Errorf("计划数据无效: %w", err)
	}
	v.path = p
	return v, nil
}

func validatePlanner(v Planner) error {
	tasks := map[string]bool{}
	for i := range v.Tasks {
		t := v.Tasks[i]
		if err := validateTask(t); err != nil {
			return fmt.Errorf("任务 %d: %w", i+1, err)
		}
		if tasks[t.ID] {
			return errors.New("任务 ID 重复")
		}
		tasks[t.ID] = true
	}
	notes := map[string]bool{}
	for i := range v.Notes {
		n := v.Notes[i]
		if strings.TrimSpace(n.ID) == "" || strings.TrimSpace(n.Title) == "" {
			return fmt.Errorf("笔记 %d 缺少 ID 或标题", i+1)
		}
		if notes[n.ID] {
			return errors.New("笔记 ID 重复")
		}
		notes[n.ID] = true
	}
	return nil
}

func validRepeat(r Repeat) bool {
	return r == RepeatNone || r == RepeatDaily || r == RepeatWeekdays || r == RepeatWeekly
}

func validateTime(s string) error {
	if s == "" {
		return nil
	}
	if len(s) != 5 || s[2] != ':' {
		return errors.New("提醒时间应为 HH:mm")
	}
	for i := 0; i < len(s); i++ {
		if i == 2 {
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return errors.New("提醒时间应为 HH:mm")
		}
	}
	_, err := time.Parse("15:04", s)
	if err != nil {
		return errors.New("提醒时间应为 HH:mm")
	}
	return nil
}

func validateTask(t Task) error {
	if strings.TrimSpace(t.ID) == "" {
		return errors.New("缺少任务 ID")
	}
	if strings.TrimSpace(t.Title) == "" {
		return errors.New("任务标题不能为空")
	}
	if t.Date != "" {
		if _, err := parseDate(t.Date); err != nil {
			return err
		}
	}
	if t.CreatedDate != "" {
		if _, err := parseDate(t.CreatedDate); err != nil {
			return errors.New("创建日期无效")
		}
	}
	if !validRepeat(t.Repeat) {
		return errors.New("重复规则无效")
	}
	if t.Repeat != RepeatNone && t.Date == "" {
		return errors.New("重复任务需要起始日期")
	}
	if err := validateTime(t.ReminderTime); err != nil {
		return err
	}
	if t.ReminderTime != "" && t.Date == "" {
		return errors.New("设置提醒前请先选择任务日期")
	}
	for _, d := range t.CompletedDates {
		if d != "unscheduled" {
			if _, err := parseDate(d); err != nil {
				return errors.New("完成日期无效")
			}
		}
	}
	return nil
}

func copyPlanner(v Planner) Planner {
	n := v
	n.Tasks = append([]Task(nil), v.Tasks...)
	for i := range n.Tasks {
		n.Tasks[i].CompletedDates = append([]string(nil), v.Tasks[i].CompletedDates...)
	}
	n.Notes = append([]Note(nil), v.Notes...)
	n.ReminderFired = make(map[string]bool, len(v.ReminderFired))
	for k, val := range v.ReminderFired {
		n.ReminderFired[k] = val
	}
	return n
}

func writeJSONFile(p string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".planner-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	// Windows does not replace an existing destination with os.Rename. Keep a
	// recoverable .bak during replacement instead of deleting the old file first.
	backup := p + ".bak"
	_ = os.Remove(backup)
	if _, err := os.Stat(p); err == nil {
		if err = os.Rename(p, backup); err != nil {
			return err
		}
	}
	if err = os.Rename(tmp, p); err != nil {
		_ = os.Rename(backup, p)
		return err
	}
	_ = os.Remove(backup)
	return nil
}

func (p *Planner) save(next Planner) error {
	if p.blocked {
		return errors.New("原有计划数据读取失败，为避免覆盖，请先处理存储文件")
	}
	if err := validatePlanner(next); err != nil {
		return err
	}
	path := p.path
	if path == "" {
		var err error
		path, err = plannerPath()
		if err != nil {
			return err
		}
	}
	next.Version = 1
	if err := writeJSONFile(path, next); err != nil {
		return err
	}
	next.path = path
	*p = next
	return nil
}

func (p *Planner) UpsertTask(t Task, now time.Time) (string, error) {
	t.Title = strings.TrimSpace(t.Title)
	t.Details = strings.TrimSpace(t.Details)
	if t.ID == "" {
		t.ID = newID("task", now)
	}
	if t.CreatedDate == "" {
		t.CreatedDate = civil(now).Format("2006-01-02")
	}
	if t.Repeat == "" {
		t.Repeat = RepeatNone
	}
	if err := validateTask(t); err != nil {
		return "", err
	}
	n := copyPlanner(*p)
	found := false
	for i := range n.Tasks {
		if n.Tasks[i].ID == t.ID {
			t.CompletedDates = append([]string(nil), n.Tasks[i].CompletedDates...)
			if t.SourceNoteID == "" {
				t.SourceNoteID = n.Tasks[i].SourceNoteID
			}
			n.Tasks[i] = t
			found = true
			break
		}
	}
	if !found {
		n.Tasks = append(n.Tasks, t)
	}
	return t.ID, p.save(n)
}

func (p *Planner) DeleteTask(id string) error {
	n := copyPlanner(*p)
	n.Tasks = n.Tasks[:0]
	for _, t := range p.Tasks {
		if t.ID != id {
			n.Tasks = append(n.Tasks, t)
		}
	}
	if len(n.Tasks) == len(p.Tasks) {
		return errors.New("找不到任务")
	}
	return p.save(n)
}

func completionKey(t Task, occurrence string) string {
	if occurrence != "" {
		return occurrence
	}
	if t.Date != "" {
		return t.Date
	}
	return "unscheduled"
}

func IsTaskCompleted(t Task, occurrence string) bool {
	key := completionKey(t, occurrence)
	for _, d := range t.CompletedDates {
		if d == key {
			return true
		}
	}
	return false
}

func (p *Planner) SetTaskCompleted(id, occurrence string, completed bool) error {
	n := copyPlanner(*p)
	for i := range n.Tasks {
		if n.Tasks[i].ID != id {
			continue
		}
		key := completionKey(n.Tasks[i], occurrence)
		filtered := n.Tasks[i].CompletedDates[:0]
		for _, d := range n.Tasks[i].CompletedDates {
			if d != key {
				filtered = append(filtered, d)
			}
		}
		n.Tasks[i].CompletedDates = filtered
		if completed {
			n.Tasks[i].CompletedDates = append(n.Tasks[i].CompletedDates, key)
			sort.Strings(n.Tasks[i].CompletedDates)
		}
		return p.save(n)
	}
	return errors.New("找不到任务")
}

func TaskOccursOn(t Task, date string) bool {
	day, err := parseDate(date)
	if err != nil || t.Date == "" {
		return false
	}
	start, _ := parseDate(t.Date)
	if day.Before(start) {
		return false
	}
	switch t.Repeat {
	case RepeatNone:
		return date == t.Date
	case RepeatDaily:
		return true
	case RepeatWeekdays:
		return day.Weekday() != time.Saturday && day.Weekday() != time.Sunday
	case RepeatWeekly:
		return day.Weekday() == start.Weekday()
	}
	return false
}

func TasksForDate(tasks []Task, date string) []TaskOccurrence {
	out := []TaskOccurrence{}
	for _, t := range tasks {
		if TaskOccursOn(t, date) {
			out = append(out, TaskOccurrence{TaskID: t.ID, Date: date})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		var a, b Task
		for _, t := range tasks {
			if t.ID == out[i].TaskID {
				a = t
			}
			if t.ID == out[j].TaskID {
				b = t
			}
		}
		if a.Priority != b.Priority {
			return a.Priority
		}
		return a.Title < b.Title
	})
	return out
}

// SelectTaskListOccurrences is the single selection source for the Today
// page's 今日 and 未来7天 modes. A one-day horizon may additionally include
// overdue one-off tasks and unscheduled tasks. Wider horizons deliberately do
// not include either, so the future list remains a calendar of real
// occurrences rather than another inbox.
func SelectTaskListOccurrences(tasks []Task, today string, horizon int, query string, includeOverdue, includeUnscheduled bool) []TaskListOccurrence {
	start, err := parseDate(today)
	if err != nil || horizon < 1 {
		return []TaskListOccurrence{}
	}
	q := strings.ToLower(strings.TrimSpace(query))
	matches := func(t Task) bool {
		return q == "" || strings.Contains(strings.ToLower(t.Title), q) || strings.Contains(strings.ToLower(t.Details), q)
	}
	out := []TaskListOccurrence{}
	if includeOverdue {
		for _, t := range tasks {
			if matches(t) && t.Repeat == RepeatNone && t.Date != "" && t.Date < today {
				out = append(out, TaskListOccurrence{TaskID: t.ID, Date: t.Date, DayOffset: -1, Category: "overdue"})
			}
		}
	}
	for offset := 0; offset < horizon; offset++ {
		date := start.AddDate(0, 0, offset).Format("2006-01-02")
		for _, t := range tasks {
			if matches(t) && TaskOccursOn(t, date) {
				out = append(out, TaskListOccurrence{TaskID: t.ID, Date: date, DayOffset: offset, Category: "scheduled"})
			}
		}
	}
	if includeUnscheduled {
		for _, t := range tasks {
			if matches(t) && t.Repeat == RepeatNone && t.Date == "" {
				out = append(out, TaskListOccurrence{TaskID: t.ID, DayOffset: horizon, Category: "unscheduled"})
			}
		}
	}
	return out
}

func (p *Planner) UpsertNote(n Note, now time.Time) (string, error) {
	n.Title = strings.TrimSpace(n.Title)
	n.Body = strings.TrimSpace(n.Body)
	stamp := now.Format(time.RFC3339)
	if n.ID == "" {
		n.ID = newID("note", now)
		n.CreatedAt = stamp
	}
	if n.CreatedAt == "" {
		n.CreatedAt = stamp
	}
	n.UpdatedAt = stamp
	if n.Title == "" {
		return "", errors.New("笔记标题不能为空")
	}
	next := copyPlanner(*p)
	found := false
	for i := range next.Notes {
		if next.Notes[i].ID == n.ID {
			next.Notes[i] = n
			found = true
			break
		}
	}
	if !found {
		next.Notes = append(next.Notes, n)
	}
	return n.ID, p.save(next)
}

func (p *Planner) DeleteNote(id string) error {
	n := copyPlanner(*p)
	n.Notes = n.Notes[:0]
	for _, note := range p.Notes {
		if note.ID != id {
			n.Notes = append(n.Notes, note)
		}
	}
	if len(n.Notes) == len(p.Notes) {
		return errors.New("找不到笔记")
	}
	return p.save(n)
}

func (p *Planner) CreateTaskFromNote(noteID, date string, now time.Time) (string, error) {
	for _, n := range p.Notes {
		if n.ID == noteID {
			return p.UpsertTask(Task{Title: n.Title, Details: n.Body, Date: date, Repeat: RepeatNone, SourceNoteID: n.ID}, now)
		}
	}
	return "", errors.New("找不到来源笔记")
}

func reminderKey(taskID, occurrence, reminderTime string) string {
	return taskID + "|" + occurrence + "|" + reminderTime
}

func (p Planner) DueReminders(now time.Time) []DueReminder {
	today := civil(now).Format("2006-01-02")
	clock := now.Format("15:04")
	out := []DueReminder{}
	for _, t := range p.Tasks {
		if t.ReminderTime == "" || t.ReminderTime > clock {
			continue
		}
		occurrence := today
		if t.Date == "" || !TaskOccursOn(t, today) {
			continue
		}
		if IsTaskCompleted(t, today) {
			continue
		}
		key := reminderKey(t.ID, occurrence, t.ReminderTime)
		if p.ReminderFired[key] {
			continue
		}
		out = append(out, DueReminder{TaskID: t.ID, Occurrence: occurrence, Title: t.Title, Time: t.ReminderTime, Key: key})
	}
	return out
}

func (p *Planner) MarkRemindersFired(reminders []DueReminder) error {
	if len(reminders) == 0 {
		return nil
	}
	n := copyPlanner(*p)
	for _, r := range reminders {
		n.ReminderFired[r.Key] = true
	}
	return p.save(n)
}

func MonthGrid(month, today time.Time) []CalendarCell {
	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	back := (int(first.Weekday()) + 6) % 7 // Monday is column zero.
	start := first.AddDate(0, 0, -back)
	todayKey := civil(today).Format("2006-01-02")
	out := make([]CalendarCell, 42)
	for i := range out {
		d := start.AddDate(0, 0, i)
		out[i] = CalendarCell{Date: d.Format("2006-01-02"), Day: d.Day(), InMonth: d.Month() == month.Month(), IsToday: d.Format("2006-01-02") == todayKey, WeekIndex: i / 7}
	}
	return out
}
