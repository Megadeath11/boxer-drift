package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"elementsandbox/campaign"
	"elementsandbox/levels"
	"elementsandbox/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ---- main menu ----

const (
	menuBtnW = 330 // one column; the levels sit in two columns
	menuBtnH = 27
	menuGap  = 4
	menuTop  = 196
	menuWide = 2*menuBtnW + menuGap
)

type menuItem struct {
	label   string
	level   int // -1 sandbox, -2 quit, -3 resume
	locked  bool
	x, y, w int
}

const (
	itemSandbox = -1
	itemQuit    = -2
	itemResume  = -3
	itemCampNew = -4
	itemCampOn  = -5
)

// menuItems lays the menu out: Resume (when a game is in progress) and
// Sandbox on wide rows, the levels in two columns, and Quit at the bottom.
func (g *Game) menuItems() []menuItem {
	left := screenW/2 - menuWide/2
	row := 0
	y := func() int { return menuTop + row*(menuBtnH+menuGap) }
	var items []menuItem
	wide := func(label string, level int) {
		items = append(items, menuItem{label: label, level: level, x: left, y: y(), w: menuWide})
		row++
	}
	if g.inGame {
		name := "Sandbox"
		switch {
		case g.level >= 0:
			name = levels.All[g.level].Name
		case g.level == levelCampaign:
			name = "Campaign, " + g.camp.Cur().Name
		}
		wide("Resume  ("+name+")", itemResume)
	}
	if g.campSaved >= 0 {
		wide("Campaign: continue  ("+campaign.Stages[g.campSaved].Name+")", itemCampOn)
		wide("Campaign: start again  (replaces your save)", itemCampNew)
	} else {
		wide("Campaign  (story mode: start here, with a tutorial)", itemCampNew)
	}
	wide("Sandbox  (free play, every material)", itemSandbox)
	rows := (len(levels.All) + 1) / 2
	for k, l := range levels.All {
		col, r := k/rows, k%rows
		items = append(items, menuItem{label: l.Name, level: k, locked: k > g.unlocked,
			x: left + col*(menuBtnW+menuGap), y: menuTop + (row+r)*(menuBtnH+menuGap), w: menuBtnW})
	}
	row += rows
	wide("Quit", itemQuit)
	return items
}

func (g *Game) updateMenu() error {
	stepDemo(g.demo)
	if g.inGame && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.mode, g.waitRelease = modePlay, true
		return nil
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}
	mx, my := ebiten.CursorPosition()
	for _, it := range g.menuItems() {
		if mx < it.x || mx >= it.x+it.w || my < it.y || my >= it.y+menuBtnH || it.locked {
			continue
		}
		switch it.level {
		case itemQuit:
			return ebiten.Termination
		case itemResume:
			g.mode, g.waitRelease = modePlay, true
		case itemSandbox:
			g.startSandbox()
		case itemCampNew:
			g.startCampaign(false)
		case itemCampOn:
			g.startCampaign(true)
		default:
			g.startLevel(it.level)
		}
	}
	return nil
}

func (g *Game) drawMenu(screen *ebiten.Image) {
	g.drawWorld(screen, g.demo, sim.ViewNormal, true)
	vector.DrawFilledRect(screen, 0, 0, screenW, screenH, color.RGBA{8, 10, 14, 150}, false)
	items := g.menuItems()
	lastY := items[len(items)-1].y
	vector.DrawFilledRect(screen, float32(screenW/2-menuWide/2-30), 50, menuWide+60, float32(lastY+menuBtnH+30-50), color.RGBA{10, 12, 18, 170}, false)
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
	for _, it := range items {
		x, y, w := it.x, it.y, it.w
		hover := mx >= x && mx < x+w && my >= y && my < y+menuBtnH
		bg := color.RGBA{30, 33, 44, 255}
		if hover && !it.locked {
			bg = color.RGBA{50, 55, 74, 255}
		}
		vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), menuBtnH, bg, false)
		tc := colText
		if it.level == itemCampNew || it.level == itemCampOn {
			tc = colAccent // the place to start
		}
		label := it.label
		switch {
		case it.locked:
			tc = colDim
			label += "   (locked)"
		case it.level >= 0 && it.level < g.unlocked:
			label += "   [done]"
		}
		if hover && !it.locked {
			vector.StrokeRect(screen, float32(x)+0.5, float32(y)+0.5, float32(w)-1, menuBtnH-1, 1, colAccent, false)
		}
		g.text(screen, label, x+14, y+7, tc)
	}
	hint := "F11 fullscreen   Esc in game returns here"
	if g.inGame {
		hint += "   Esc here resumes"
	}
	g.text(screen, hint, 12, screenH-20, colDim)
}

// ---- starting modes ----

func (g *Game) startSandbox() {
	g.mode, g.level, g.inGame = modePlay, levelSandbox, true
	g.won, g.brief, g.hold, g.paused = false, false, 0, false
	g.help = true
	g.resetCamera()
	g.sim.Climate = sim.Climate{Enabled: true, Weather: true, YearFrames: sandboxYear}
	g.sim.Climate.SetSeason(sim.Spring)
	g.sim.Climate.Phase += 0.125 // mid-spring: mild, not the tail of winter
	g.sim.Clear()
	g.sim.Wind.Global = 0
	g.sim.AddRoof(roofCells)
	g.buildPalette(nil)
	g.waitRelease = true
}

func (g *Game) startLevel(k int) {
	l := levels.All[k]
	g.mode, g.level, g.inGame = modePlay, k, true
	g.placed = [sim.NumMats]int{}
	g.won, g.brief, g.hold, g.paused, g.help = false, true, 0, false, false
	g.lframe = 0
	g.resetCamera()
	g.sim.Climate = sim.Climate{} // levels opt in to seasons in Build
	g.sim.Ambient = sim.DefaultAmbient
	g.sim.Clear()
	g.sim.Wind.Global = l.Wind
	g.sim.OpenTop = l.OpenTop
	l.Build(g.sim)
	g.sim.Step() // so the HUD knows how many beacons there are
	g.buildPalette(l.Allow)
	g.brush = 3
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
		g.wonT++
		if g.wonT < 60 { // don't let a click meant for the level skip the win screen
			return true
		}
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
	if levelAt(g.level).Won(g.sim) {
		g.hold++
	} else {
		g.hold = 0
	}
	if g.hold >= levelAt(g.level).HoldTime() && !g.won {
		g.won, g.wonT = true, 0
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
	extra := float32(0)
	if l.MaxPPM > 0 {
		extra = 14
	}
	vector.DrawFilledRect(screen, x, y, 300, 62+extra, color.RGBA{10, 12, 18, 200}, false)
	g.text(screen, l.Name, int(x)+8, int(y)+5, colAccent)
	g.text(screen, fmt.Sprintf("Beacons powered: %d / %d", st.BeaconOn, st.Beacons), int(x)+8, int(y)+21, colText)
	g.text(screen, fmt.Sprintf("Power: making %.1f, using %.1f", st.Generated, st.Used), int(x)+8, int(y)+35, colDim)
	if l.MaxPPM > 0 {
		c := color.RGBA{110, 220, 120, 255}
		if g.sim.Climate.PPM > l.MaxPPM {
			c = color.RGBA{255, 140, 70, 255}
		}
		g.text(screen, fmt.Sprintf("CO2: %.0f ppm (goal: %.0f or less)", g.sim.Climate.PPM, l.MaxPPM), int(x)+8, int(y)+49, c)
	}
	bw := float32(284)
	vector.DrawFilledRect(screen, x+8, y+51+extra, bw, 5, color.RGBA{40, 44, 56, 255}, false)
	vector.DrawFilledRect(screen, x+8, y+51+extra, bw*min(1, float32(g.hold)/float32(l.HoldTime())), 5, color.RGBA{90, 255, 140, 255}, false)

	switch {
	case g.brief:
		lines := append([]string{l.Name, ""}, l.Brief...)
		secs := l.HoldTime() / 60
		goal := fmt.Sprintf("Goal: keep every beacon powered for %d seconds.", secs)
		if l.MaxPPM > 0 {
			goal = fmt.Sprintf("Goal: beacons on and CO2 <= %.0f ppm for %d seconds.", l.MaxPPM, secs)
		}
		lines = append(lines, "", goal,
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

const (
	sandboxYear = 60 * 60 * 4 // a sandbox year lasts 4 minutes
	roofCells   = 6
)

func levelAt(k int) *levels.Level { return &levels.All[k] }
