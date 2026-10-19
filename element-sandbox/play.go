package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"elementsandbox/levels"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ---- main menu ----

const (
	menuBtnW = 360
	menuBtnH = 30
	menuTop  = 230
)

type menuItem struct {
	label  string
	level  int // -1 sandbox, -2 quit
	locked bool
}

func (g *Game) menuItems() []menuItem {
	items := []menuItem{{label: "Sandbox  (free play, every material)", level: -1}}
	for k, l := range levels.All {
		items = append(items, menuItem{label: l.Name, level: k, locked: k > g.unlocked})
	}
	return append(items, menuItem{label: "Quit", level: -2})
}

func menuBtnRect(k int) (x, y int) {
	return screenW/2 - menuBtnW/2, menuTop + k*(menuBtnH+6)
}

func (g *Game) updateMenu() error {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}
	mx, my := ebiten.CursorPosition()
	for k, it := range g.menuItems() {
		x, y := menuBtnRect(k)
		if mx < x || mx >= x+menuBtnW || my < y || my >= y+menuBtnH || it.locked {
			continue
		}
		switch it.level {
		case -2:
			return ebiten.Termination
		case -1:
			g.startSandbox()
		default:
			g.startLevel(it.level)
		}
	}
	return nil
}

func (g *Game) drawMenu(screen *ebiten.Image) {
	screen.Fill(color.RGBA{14, 16, 22, 255})
	op := &text.DrawOptions{}
	op.GeoM.Scale(5, 5)
	title := "ELEMENT SANDBOX"
	op.GeoM.Translate(float64(screenW/2-len(title)*7*5/2), 70)
	op.ColorScale.ScaleWithColor(colAccent)
	text.Draw(screen, title, g.face, op)
	sub := "Boil it. Burn it. Power it."
	g.text(screen, sub, screenW/2-len(sub)*7/2, 150, colText)
	sub = "Make power from steam, heat and wind to light every beacon."
	g.text(screen, sub, screenW/2-len(sub)*7/2, 170, colDim)

	mx, my := ebiten.CursorPosition()
	for k, it := range g.menuItems() {
		x, y := menuBtnRect(k)
		hover := mx >= x && mx < x+menuBtnW && my >= y && my < y+menuBtnH
		bg := color.RGBA{30, 33, 44, 255}
		if hover && !it.locked {
			bg = color.RGBA{50, 55, 74, 255}
		}
		vector.DrawFilledRect(screen, float32(x), float32(y), menuBtnW, menuBtnH, bg, false)
		tc := colText
		label := it.label
		switch {
		case it.locked:
			tc = colDim
			label += "   (locked)"
		case it.level >= 0 && it.level < g.unlocked:
			label += "   [done]"
		}
		if hover && !it.locked {
			vector.StrokeRect(screen, float32(x)+0.5, float32(y)+0.5, menuBtnW-1, menuBtnH-1, 1, colAccent, false)
		}
		g.text(screen, label, x+14, y+9, tc)
	}
	g.text(screen, "F11 fullscreen   Esc in game returns here", 12, screenH-20, colDim)
}

// ---- starting modes ----

func (g *Game) startSandbox() {
	g.mode, g.level = modePlay, -1
	g.won, g.brief, g.hold, g.paused = false, false, 0, false
	g.help = true
	g.sim.Clear()
	g.sim.Wind.Global = 0
	g.sim.OpenTop = true
	g.buildPalette(nil)
	g.waitRelease = true
}

func (g *Game) startLevel(k int) {
	l := levels.All[k]
	g.mode, g.level = modePlay, k
	g.won, g.brief, g.hold, g.paused, g.help = false, true, 0, false, false
	g.sim.Clear()
	g.sim.Wind.Global = l.Wind
	g.sim.OpenTop = l.OpenTop
	l.Build(g.sim)
	g.sim.Step() // so the HUD knows how many beacons there are
	g.buildPalette(l.Allow)
	g.brush = 2
	g.waitRelease = true
}

// updateLevelOverlays handles the briefing and win screens; it returns true
// while one of them is up (the world is frozen).
func (g *Game) updateLevelOverlays() bool {
	click := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	enter := inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace)
	switch {
	case g.brief:
		if click || enter {
			g.brief = false
			g.waitRelease = true
		}
		return true
	case g.won:
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			g.startLevel(g.level)
		} else if click || enter {
			if g.level+1 < len(levels.All) {
				g.startLevel(g.level + 1)
			} else {
				g.mode = modeMenu
			}
		}
		return true
	}
	return false
}

func (g *Game) checkGoal() {
	st := g.sim.Stats
	if st.Beacons > 0 && st.BeaconOn == st.Beacons {
		g.hold++
	} else {
		g.hold = 0
	}
	if g.hold >= levels.HoldFrames && !g.won {
		g.won = true
		if g.level+1 > g.unlocked {
			g.unlocked = g.level + 1
			saveProgress(g.unlocked)
		}
	}
}

func (g *Game) drawLevelHUD(screen *ebiten.Image) {
	l := levels.All[g.level]
	st := g.sim.Stats
	x, y := float32(8), float32(8)
	vector.DrawFilledRect(screen, x, y, 300, 62, color.RGBA{10, 12, 18, 200}, false)
	g.text(screen, l.Name, int(x)+8, int(y)+5, colAccent)
	g.text(screen, fmt.Sprintf("Beacons powered: %d / %d", st.BeaconOn, st.Beacons), int(x)+8, int(y)+21, colText)
	g.text(screen, fmt.Sprintf("Power: making %.1f, using %.1f", st.Generated, st.Used), int(x)+8, int(y)+35, colDim)
	bw := float32(284)
	vector.DrawFilledRect(screen, x+8, y+51, bw, 5, color.RGBA{40, 44, 56, 255}, false)
	vector.DrawFilledRect(screen, x+8, y+51, bw*float32(g.hold)/levels.HoldFrames, 5, color.RGBA{90, 255, 140, 255}, false)

	switch {
	case g.brief:
		lines := append([]string{l.Name, ""}, l.Brief...)
		lines = append(lines, "", "Goal: keep every beacon powered for 3 seconds.",
			"R restarts, Esc for menu, H for controls.", "", "Click or press Enter to start.")
		g.drawPanel(screen, lines, colAccent)
	case g.won:
		next := "Click or press Enter for the next level."
		if g.level+1 >= len(levels.All) {
			next = "That was the last level! Click to return to the menu."
		}
		g.drawPanel(screen, []string{"LEVEL COMPLETE!", "", "Every beacon is lit.", "", next, "R to replay."},
			color.RGBA{90, 255, 140, 255})
	}
}

func (g *Game) drawPanel(screen *ebiten.Image, lines []string, titleCol color.RGBA) {
	w := 0
	for _, l := range lines {
		w = max(w, len(l)*7)
	}
	pw, ph := float32(w+40), float32(len(lines)*15+24)
	x, y := float32(simW)/2-pw/2, float32(simH)/2-ph/2
	vector.DrawFilledRect(screen, x, y, pw, ph, color.RGBA{10, 12, 18, 235}, false)
	vector.StrokeRect(screen, x, y, pw, ph, 1, titleCol, false)
	for k, l := range lines {
		c := colText
		if k == 0 {
			c = titleCol
		}
		g.text(screen, l, int(x)+20, int(y)+12+k*15, c)
	}
}

// ---- progress, saved next to the exe so the game stays portable ----

func progressPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "ElementSandbox-progress.txt")
}

func loadProgress() int {
	p := progressPath()
	if p == "" {
		return 0
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || n < 0 {
		return 0
	}
	return min(n, len(levels.All))
}

func saveProgress(n int) {
	if p := progressPath(); p != "" {
		_ = os.WriteFile(p, []byte(strconv.Itoa(n)), 0o644)
	}
}
