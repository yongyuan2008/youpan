package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	defaultWindowWidth  = 760
	defaultWindowHeight = 700
	minWindowWidth      = 560
	minWindowHeight     = 510
	maxWindowWidth      = 3840
	maxWindowHeight     = 2160
	minOpacity          = 0.50
	maxOpacity          = 1.00
)

// Appearance is deliberately stored separately from data.json and
// planner.json. Older builds therefore keep reading the exact schemas they
// already understand.
type Appearance struct {
	Width           int              `json:"width"`
	Height          int              `json:"height"`
	Opacity         float64          `json:"opacity"`
	Names           map[Kind]string  `json:"names"`
	DesktopPinned   bool             `json:"desktopPinned,omitempty"`
	DesktopPosition *DesktopPosition `json:"desktopPosition,omitempty"`
	path            string
	blocked         bool
}

type DesktopPosition struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

func defaultAppearance() Appearance {
	return Appearance{Width: defaultWindowWidth, Height: defaultWindowHeight, Opacity: 1, Names: map[Kind]string{}}
}

func appearancePath() (string, error) {
	dir, err := storageDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "appearance.json"), nil
}

func LoadAppearance() (Appearance, error) {
	p, err := appearancePath()
	if err != nil {
		a := defaultAppearance()
		a.blocked = true
		return a, err
	}
	return loadAppearancePath(p)
}

func loadAppearancePath(p string) (Appearance, error) {
	a := defaultAppearance()
	a.path = p
	b, err := readJSONFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return a, nil
	}
	if err != nil {
		a.blocked = true
		return a, err
	}
	if err = json.Unmarshal(b, &a); err != nil {
		a = defaultAppearance()
		a.path, a.blocked = p, true
		return a, fmt.Errorf("读取外观设置失败: %w", err)
	}
	a.path = p
	if a.Names == nil {
		a.Names = map[Kind]string{}
	}
	if err = validateAppearance(a); err != nil {
		safe := defaultAppearance()
		safe.path, safe.blocked = p, true
		return safe, fmt.Errorf("外观设置无效: %w", err)
	}
	return a, nil
}

func validateAppearance(a Appearance) error {
	if a.Width < minWindowWidth || a.Width > maxWindowWidth || a.Height < minWindowHeight || a.Height > maxWindowHeight {
		return fmt.Errorf("窗口尺寸须在 %dx%d 至 %dx%d 之间", minWindowWidth, minWindowHeight, maxWindowWidth, maxWindowHeight)
	}
	if math.IsNaN(a.Opacity) || math.IsInf(a.Opacity, 0) || a.Opacity < minOpacity || a.Opacity > maxOpacity {
		return errors.New("窗口透明度须在 50% 至 100% 之间")
	}
	for k, name := range a.Names {
		if !validKind(k) {
			return errors.New("名称设置包含未知事项")
		}
		if err := validateCustomName(name); err != nil {
			return fmt.Errorf("%s名称无效: %w", k.Title(), err)
		}
		if name != strings.TrimSpace(name) {
			return fmt.Errorf("%s名称前后不能有空白", k.Title())
		}
	}
	return nil
}

func validateCustomName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("名称不能为空")
	}
	if !utf8.ValidString(name) {
		return errors.New("名称不是有效的 Unicode 文本")
	}
	if n := utf8.RuneCountInString(name); n > 24 {
		return errors.New("名称不能超过 24 个字符")
	}
	return nil
}

func copyAppearance(a Appearance) Appearance {
	n := a
	n.Names = make(map[Kind]string, len(a.Names))
	for k, name := range a.Names {
		n.Names[k] = name
	}
	return n
}

func (a *Appearance) save(next Appearance) error {
	if a.blocked {
		return errors.New("原有外观设置读取失败，为避免覆盖，请先处理 appearance.json")
	}
	if err := validateAppearance(next); err != nil {
		return err
	}
	p := a.path
	if p == "" {
		var err error
		p, err = appearancePath()
		if err != nil {
			return err
		}
	}
	next.path = ""
	next.blocked = false
	if err := writeJSONFile(p, next); err != nil {
		return err
	}
	next.path = p
	*a = next
	return nil
}

func (a *Appearance) SetWindow(width, height int) error {
	n := copyAppearance(*a)
	n.Width, n.Height = width, height
	return a.save(n)
}

func (a *Appearance) SetOpacity(opacity float64) error {
	n := copyAppearance(*a)
	n.Opacity = opacity
	return a.save(n)
}

func (a *Appearance) SetDesktopPinned(pinned bool) error {
	n := copyAppearance(*a)
	n.DesktopPinned = pinned
	return a.save(n)
}

func (a *Appearance) SetDesktopPosition(x, y int32) error {
	n := copyAppearance(*a)
	n.DesktopPosition = &DesktopPosition{X: x, Y: y}
	return a.save(n)
}

func (a *Appearance) SetNames(names map[Kind]string) error {
	n := copyAppearance(*a)
	n.Names = make(map[Kind]string, len(names))
	for k, raw := range names {
		name := strings.TrimSpace(raw)
		if err := validateCustomName(name); err != nil {
			return fmt.Errorf("%s名称无效: %w", k.Title(), err)
		}
		if name != k.Title() {
			n.Names[k] = name
		}
	}
	return a.save(n)
}

func (a *Appearance) ResetNames() error {
	n := copyAppearance(*a)
	n.Names = map[Kind]string{}
	return a.save(n)
}

func (a *Appearance) ResetVisuals() error {
	n := copyAppearance(*a)
	n.Width, n.Height, n.Opacity = defaultWindowWidth, defaultWindowHeight, 1
	n.DesktopPinned = false
	return a.save(n)
}

func (a Appearance) Name(k Kind) (string, bool) {
	if name := strings.TrimSpace(a.Names[k]); name != "" {
		return name, true
	}
	return k.Title(), false
}
