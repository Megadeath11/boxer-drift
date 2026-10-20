package main

import (
	"bytes"
	"fmt"
	"image/color"
	"os"
	"path/filepath"

	"elementsandbox/campaign"
	"elementsandbox/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Values of Game.level that aren't level indexes.
const (
	levelSandbox  = -1
	levelCampaign = -2
)

// ---- starting, saving, restarting ----

// campaignPath is the campaign save next to the exe; with stage set, it's
// the copy taken at the start of the current stage (what R restarts from).
func campaignPath(stage bool) string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	name := "ElementSandbox-campaign.sav"
	if stage {
		name = "ElementSandbox-campaign-stage.sav"
	}
	return filepath.Join(filepath.Dir(exe), name)
}

// savedCampaignStage peeks at the save on disk: its stage, or -1.
func savedCampaignStage() int {
	p := campaignPath(false)
	if p == "" {
		return -1
	}
	f, err := os.Open(p)
	if err != nil {
		return -1
	}
	defer f.Close()
	c, err := campaign.Load(f, sim.New())
	if err != nil {
		return -1
	}
	return c.Stage
}

// startCampaign begins a new campaign, or (resume) continues the saved one.
func (g *Game) startCampaign(resume bool) {
	g.mode, g.level, g.inGame = modePlay, levelCampaign, true
	g.won, g.hold, g.paused, g.help, g.campFree = false, 0, false, false, false
	g.resetCamera()
	g.camp = nil
	if resume {
		if b, err := os.ReadFile(campaignPath(false)); err == nil {
			if c, err := campaign.Load(bytes.NewReader(b), g.sim); err == nil {
				g.camp, g.campStart = c, b
				g.campFree = c.Final()
				// restart from the start of this stage if that copy is there
				if sb, err := os.ReadFile(campaignPath(true)); err == nil {
					if sc, err := campaign.Load(bytes.NewReader(sb), sim.New()); err == nil && sc.Stage == c.Stage {
						g.campStart = sb
					}
				}
			}
		}
	}
	if g.camp == nil {
		g.camp = campaign.Start(g.sim)
		g.checkpoint()
	}
	g.brief = !g.campFree
	g.buildShop()
	g.brush = 0 // one-pixel wire is the cheapest
	g.waitRelease = true
}

// checkpoint saves the campaign as it stands, as the point R restarts from.
func (g *Game) checkpoint() {
	var b bytes.Buffer
	if g.camp.Save(&b, g.sim) == nil {
		g.campStart = b.Bytes()
		g.writeSave(g.campStart)
		if p := campaignPath(true); p != "" {
			_ = os.WriteFile(p, g.campStart, 0o644)
		}
	}
}

// saveCampaign writes the campaign in progress (on leaving or closing).
func (g *Game) saveCampaign() {
	if g.level != levelCampaign || g.camp == nil {
		return
	}
	var b bytes.Buffer
	if g.camp.Save(&b, g.sim) == nil {
		g.writeSave(b.Bytes())
	}
}

func (g *Game) writeSave(b []byte) {
	if p := campaignPath(false); p != "" && os.WriteFile(p, b, 0o644) == nil {
		g.campSaved = g.camp.Stage
	}
}

// askRestartStage restarts the stage on a second press of R.
func (g *Game) askRestartStage() {
	if g.restartAsk == 0 {
		g.restartAsk = 150
		g.say("Press R again to restart this stage")
		return
	}
	g.restartAsk, g.toastT = 0, 0
	if c, err := campaign.Load(bytes.NewReader(g.campStart), g.sim); err == nil {
		g.camp, g.won, g.brief = c, false, true
		g.buildShop()
	}
}

// ---- the shop ----

var kitColors = map[string]color.RGBA{
	campaign.WindTurbine.Name: {170, 180, 195, 255},
	campaign.SteamBoiler.Name: {200, 120, 70, 255},
	campaign.BatteryBank.Name: {60, 200, 90, 255},
}

func priceTag(p float64) string {
	if p < 1 {
		return fmt.Sprintf("$%.2f", p)
	}
	return fmt.Sprintf("$%.0f", p)
}

// buildShop lays out the sidebar for the current stage.
func (g *Game) buildShop() {
	g.clearUndo() // a new stage (or a restart): there's nothing to go back to
	var kits, mats, tools []entry
	for _, it := range g.camp.Cur().Shop {
		switch {
		case it.Kit != nil:
			kits = append(kits, entry{label: it.Kit.Name, tool: toolKit, kit: it.Kit, budget: -1,
				price: it.Kit.Cost(), col: kitColors[it.Kit.Name],
				desc: it.Kit.Desc + fmt.Sprintf(" Costs $%.0f.", it.Kit.Cost())})
		case it.Tool == campaign.ToolDemolish:
			tools = append(tools, entry{label: "Demolish", tool: toolDemolish, budget: -1, col: color.RGBA{200, 70, 60, 255},
				desc: fmt.Sprintf("Pull down your own buildings and wire for %.0f%% of their price back. (Right mouse button too.)", campaign.Refund*100)})
		case it.Tool != "":
			e := toolEntries[it.Tool]
			e.budget = -1
			tools = append(tools, e)
		default:
			e := matEntry(it.Mat, -1)
			e.label = it.Name()
			e.price = campaign.CellPrice(it.Mat)
			e.desc += fmt.Sprintf(" %s per pixel.", priceTag(e.price))
			mats = append(mats, e)
		}
	}
	prev := ""
	if g.sel < len(g.entries) {
		prev = g.entries[g.sel].label
	}
	g.entries, g.headers, g.scroll = nil, nil, 0
	y := 0
	for _, grp := range []struct {
		name  string
		items []entry
	}{{"BUILD", kits}, {"MATERIALS", mats}, {"TOOLS", tools}} {
		if len(grp.items) == 0 {
			continue
		}
		g.headers = append(g.headers, header{grp.name, y})
		y += 14
		for _, e := range grp.items {
			e.x, e.y, e.w = simW+2, y, 2*btnW+1
			g.entries = append(g.entries, e)
			y += btnH + 1
		}
		y += 4
	}
	g.listH = y
	g.sel = 0
	for k, e := range g.entries {
		if e.label == prev {
			g.sel = k
		}
	}
	// the tutorial starts with nothing useful selected, so picking the
	// turbine is a deliberate first step
	if g.camp.Stage == 0 && g.camp.Step == 0 {
		for k, e := range g.entries {
			if e.tool == toolDemolish {
				g.sel = k
			}
		}
	}
}

// buyKit buys and places a kit, reporting whether it was built.
func (g *Game) buyKit(k *campaign.Kit, x, y int) bool {
	switch {
	case g.camp.Money < k.Cost():
		g.say(fmt.Sprintf("A %s costs $%.0f: you have $%.0f. Demolish something for a refund.", k.Name, k.Cost(), g.camp.Money))
	case !g.camp.BuyKit(g.sim, k, x, y):
		g.say("It doesn't fit there: it needs open space, standing on the ground.")
	default:
		g.say(fmt.Sprintf("Built a %s for $%.0f", k.Name, k.Cost()))
		return true
	}
	return false
}

func (g *Game) demolish(x, y, r int) {
	if got := g.camp.Demolish(g.sim, x, y, r); got > 0 {
		g.say(fmt.Sprintf("Demolished: +$%.0f", got))
	}
}

// ---- overlays ----

// updateCampaignOverlays handles the briefing and stage-complete screens;
// it returns true while one is up (the world is frozen).
func (g *Game) updateCampaignOverlays() bool {
	click := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	enter := inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace)
	switch {
	case g.brief:
		if click || enter {
			g.brief, g.waitRelease = false, true
		}
		return true
	case g.camp.Won && !g.campFree:
		if !g.won { // just won: give the player a moment before a click counts
			g.won, g.wonT = true, 0
		}
		g.wonT++
		if g.wonT < 60 || !(click || enter) {
			return true
		}
		g.won, g.waitRelease = false, true
		if g.camp.Next(g.sim) {
			g.checkpoint()
			g.brief = true
			g.buildShop()
		} else {
			g.campFree = true
			g.saveCampaign()
		}
		return true
	}
	return false
}

func (g *Game) drawCampaign(screen *ebiten.Image) {
	c := g.camp
	st := c.Cur()
	stats := g.sim.Stats

	// what's under the cursor: a ghost of the kit about to be bought
	mx, my := ebiten.CursorPosition()
	if e := g.entries[g.sel]; e.tool == toolKit && mx < simW && my < simH && !g.brief && !c.Won {
		x, y := g.toCell(mx, my)
		yy, ok := e.kit.Drop(g.sim, x, y)
		ok = ok && c.Money >= e.kit.Cost()
		for _, p := range e.kit.Pieces {
			col := sim.Info(p.M).Color
			if !ok {
				col = color.RGBA{220, 60, 50, 255}
			}
			// a see-through ghost (colours are premultiplied by alpha)
			col = color.RGBA{col.R / 2, col.G / 2, col.B / 2, 128}
			sx, sy := g.toScreen(float64(x+p.X), float64(yy+p.Y))
			s := float32(g.ppc())
			vector.DrawFilledRect(screen, sx, sy, s, s, col, false)
		}
	}

	// objectives panel
	lines := []struct {
		text string
		col  color.RGBA
	}{}
	add := func(t string, col color.RGBA) {
		lines = append(lines, struct {
			text string
			col  color.RGBA
		}{t, col})
	}
	green := color.RGBA{110, 220, 120, 255}
	if g.campFree {
		add("Campaign complete: build freely!", green)
	} else {
		add(fmt.Sprintf("Houses lit: %d / %d", stats.BeaconOn, stats.Beacons), colText)
	}
	add(fmt.Sprintf("Power: making %.0f, using %.0f", stats.Generated, stats.Used), colDim)
	if stats.Capacity > 0 {
		add(fmt.Sprintf("Batteries: %.0f%% charged", 100*stats.Stored/stats.Capacity), colDim)
	}
	if st.MaxPPM > 0 && !g.campFree {
		col := green
		if g.sim.Climate.PPM > st.MaxPPM {
			col = color.RGBA{255, 140, 70, 255}
		}
		add(fmt.Sprintf("CO2: %.0f ppm (goal %.0f or less)", g.sim.Climate.PPM, st.MaxPPM), col)
	}
	if !g.campFree {
		add("", colText)
		for k, step := range st.Steps {
			switch {
			case k < c.Step:
				add("[x] "+step.Text, colDim)
			case k == c.Step:
				add("[ ] "+step.Text, colAccent)
			}
		}
		if c.Step >= len(st.Steps) {
			goal := fmt.Sprintf("Keep every house lit for %d s", st.Hold/60)
			if st.MaxPPM > 0 {
				goal += " with clean air"
			}
			add(">> "+goal, colAccent)
		}
		for _, t := range st.Tips {
			add("   "+t, colDim)
		}
	}
	w := 220
	for _, l := range lines {
		w = max(w, len(l.text)*7)
	}
	x, y := float32(8), float32(8)
	h := float32(44 + len(lines)*14)
	if !g.campFree {
		h += 10
	}
	vector.DrawFilledRect(screen, x, y, float32(w+16), h, color.RGBA{10, 12, 18, 210}, false)
	title := "CAMPAIGN"
	if !g.campFree {
		title = st.Name
	}
	g.text(screen, title, int(x)+8, int(y)+5, colAccent)
	money := fmt.Sprintf("$%.0f", c.Money)
	g.text(screen, money, int(x)+w+8-len(money)*7, int(y)+5, color.RGBA{255, 230, 120, 255})
	for k, l := range lines {
		g.text(screen, l.text, int(x)+8, int(y)+24+k*14, l.col)
	}
	if !g.campFree {
		bw := float32(w)
		by := y + h - 12
		vector.DrawFilledRect(screen, x+8, by, bw, 5, color.RGBA{40, 44, 56, 255}, false)
		vector.DrawFilledRect(screen, x+8, by, bw*min(1, float32(c.Hold)/float32(st.Hold)), 5, color.RGBA{90, 255, 140, 255}, false)
	}

	switch {
	case g.brief:
		lines := append([]string{st.Name, ""}, st.Brief...)
		if c.Paid > 0 {
			lines = append(lines, "", fmt.Sprintf("You were paid $%.0f for your buildings that were in the way.", c.Paid))
		}
		lines = append(lines, "", fmt.Sprintf("You have $%.0f to spend.", c.Money),
			"Esc saves and returns to the menu. R twice restarts the stage.", "", "Click or press Enter to start.")
		g.drawPanel(screen, lines, colAccent)
	case c.Won && !g.campFree:
		lines := []string{"STAGE COMPLETE!", "", "The town has the power it needs."}
		if c.Stage+1 < len(campaign.Stages) {
			next := campaign.Stages[c.Stage+1]
			lines = append(lines, "", fmt.Sprintf("Next: %s  (+$%.0f)", next.Name, next.Grant), "", "Click or press Enter to carry on.")
		} else {
			lines = append(lines, "", "That was the last stage: the valley is powered!",
				"Carry on building as you like.", "", "Click or press Enter.")
		}
		g.drawPanel(screen, lines, color.RGBA{90, 255, 140, 255})
	}
}
