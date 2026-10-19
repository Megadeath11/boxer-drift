// Element Sandbox: a 2D physics/chemistry sandbox.
package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"strings"

	"elementsandbox/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

const (
	scale    = 3
	simW     = sim.W * scale
	simH     = sim.H * scale
	sideW    = 200
	statusH  = 26
	screenW  = simW + sideW
	screenH  = simH + statusH
	btnW     = 98
	btnH     = 15
	maxBrush = 40
)

type tool int

const (
	toolMat tool = iota
	toolErase
	toolFire
	toolHeat
	toolCool
	toolWind
	toolBomb
)

type entry struct {
	label string
	tool  tool
	mat   sim.Mat
	col   color.RGBA
	desc  string
	x, y  int
}

type header struct {
	label string
	y     int
}

type Game struct {
	sim      *sim.Sim
	img      *ebiten.Image
	pix      []byte
	face     text.Face
	entries  []entry
	headers  []header
	infoY    int
	sel      int
	brush    int
	paused   bool
	view     int
	showWind bool
	help     bool
	lastX    int
	lastY    int
	drawing  bool
}

func newGame() *Game {
	g := &Game{
		sim:   sim.New(),
		img:   ebiten.NewImage(sim.W, sim.H),
		pix:   make([]byte, sim.W*sim.H*4),
		face:  text.NewGoXFace(basicfont.Face7x13),
		brush: 4,
		help:  true,
	}
	g.buildPalette()
	return g
}

func (g *Game) buildPalette() {
	y := 24
	addGroup := func(name string, items []entry) {
		g.headers = append(g.headers, header{name, y})
		y += 14
		for k, e := range items {
			e.x = simW + 2 + (k%2)*(btnW+1)
			e.y = y + (k/2)*(btnH+1)
			g.entries = append(g.entries, e)
		}
		y += (len(items)+1)/2*(btnH+1) + 4
	}
	addGroup("TOOLS", []entry{
		{label: "Erase", tool: toolErase, col: color.RGBA{60, 60, 70, 255}, desc: "Remove anything. (Right mouse button also erases.)"},
		{label: "Fire", tool: toolFire, col: color.RGBA{255, 140, 30, 255}, desc: "A flame: lights and heats whatever it touches."},
		{label: "Heat", tool: toolHeat, col: color.RGBA{230, 60, 40, 255}, desc: "Pump heat in. Melt, boil, ignite."},
		{label: "Cool", tool: toolCool, col: color.RGBA{60, 140, 255, 255}, desc: "Pull heat out. Freeze, condense, liquefy."},
		{label: "Wind", tool: toolWind, col: color.RGBA{150, 200, 210, 255}, desc: "Drag to blow air in that direction."},
		{label: "Bomb", tool: toolBomb, col: color.RGBA{200, 200, 60, 255}, desc: "Click for an explosion (size = brush)."},
	})
	groups := []struct {
		name string
		st   sim.State
	}{{"SOLIDS", sim.SSolid}, {"POWDERS", sim.SPowder}, {"LIQUIDS", sim.SLiquid}, {"GASES", sim.SGas}}
	for _, gr := range groups {
		var items []entry
		for m := sim.Mat(1); m < sim.NumMats; m++ {
			info := sim.Info(m)
			if info.State == gr.st {
				items = append(items, entry{label: info.Name, tool: toolMat, mat: m, col: info.Color, desc: info.Desc})
			}
		}
		addGroup(gr.name, items)
	}
	g.infoY = y + 2
	for k, e := range g.entries {
		if e.tool == toolMat && e.mat == sim.Sand {
			g.sel = k
		}
	}
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.paused = !g.paused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		g.view ^= 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		g.showWind = !g.showWind
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyH) || inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.help = !g.help
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyO) {
		g.sim.OpenTop = !g.sim.OpenTop
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.sim.Clear()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.sim.Wind.Global = float32(math.Min(float64(g.sim.Wind.Global+0.5), 5))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.sim.Wind.Global = float32(math.Max(float64(g.sim.Wind.Global-0.5), -5))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.sim.Wind.Global = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) || inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
		g.brush = min(g.brush+1, maxBrush)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) || inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
		g.brush = max(g.brush-1, 0)
	}
	if _, wy := ebiten.Wheel(); wy != 0 {
		if wy > 0 {
			g.brush = min(g.brush+1, maxBrush)
		} else {
			g.brush = max(g.brush-1, 0)
		}
	}

	mx, my := ebiten.CursorPosition()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && mx >= simW {
		for k, e := range g.entries {
			if mx >= e.x && mx < e.x+btnW && my >= e.y && my < e.y+btnH {
				g.sel = k
			}
		}
	}

	left := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	right := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	inSim := mx >= 0 && mx < simW && my >= 0 && my < simH
	if inSim && (left || right) {
		sx, sy := mx/scale, my/scale
		if !g.drawing {
			g.lastX, g.lastY = sx, sy
			if left && g.entries[g.sel].tool == toolBomb {
				g.sim.Explode(sx, sy, 4+float32(g.brush)*1.2)
			}
		}
		g.applyTool(g.lastX, g.lastY, sx, sy, right)
		g.lastX, g.lastY = sx, sy
		g.drawing = true
	} else {
		g.drawing = false
	}

	if !g.paused || inpututil.IsKeyJustPressed(ebiten.KeyN) {
		g.sim.Step()
	}
	return nil
}

// applyTool applies the selected tool along the stroke from (x0,y0) to (x1,y1).
func (g *Game) applyTool(x0, y0, x1, y1 int, erase bool) {
	e := g.entries[g.sel]
	r := g.brush
	if e.tool == toolWind && !erase {
		fx := clampf(float32(x1-x0)*0.25, -3, 3)
		fy := clampf(float32(y1-y0)*0.25, -3, 3)
		g.sim.Wind.AddForce(x1, y1, float32(r+3), fx, fy)
		return
	}
	dist := math.Hypot(float64(x1-x0), float64(y1-y0))
	steps := int(dist/math.Max(1, float64(r)/2)) + 1
	replace := ebiten.IsKeyPressed(ebiten.KeyShift)
	for k := 0; k < steps; k++ {
		t := float64(k+1) / float64(steps)
		x := x0 + int(math.Round(float64(x1-x0)*t))
		y := y0 + int(math.Round(float64(y1-y0)*t))
		if erase {
			g.sim.Erase(x, y, r)
			continue
		}
		switch e.tool {
		case toolMat:
			g.sim.Paint(x, y, r, e.mat, replace)
		case toolErase:
			g.sim.Erase(x, y, r)
		case toolFire:
			g.sim.Ignite(x, y, r)
		case toolHeat:
			g.sim.AddHeat(x, y, r, 40)
		case toolCool:
			g.sim.AddHeat(x, y, r, -40)
		}
	}
}

func clampf(v, lo, hi float32) float32 {
	return float32(math.Max(float64(lo), math.Min(float64(hi), float64(v))))
}

var (
	colPanel  = color.RGBA{24, 26, 34, 255}
	colText   = color.RGBA{220, 222, 230, 255}
	colDim    = color.RGBA{130, 135, 150, 255}
	colAccent = color.RGBA{255, 200, 80, 255}
)

func (g *Game) text(dst *ebiten.Image, s string, x, y int, c color.Color) {
	s = strings.ReplaceAll(s, "°", "") // the bitmap font is ASCII-only
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, g.face, op)
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.sim.Render(g.pix, g.view)
	g.img.WritePixels(g.pix)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	screen.DrawImage(g.img, op)

	if g.showWind {
		g.drawWind(screen)
	}

	mx, my := ebiten.CursorPosition()
	if mx < simW && my < simH {
		r := float32(g.brush*scale) + 1.5
		vector.StrokeCircle(screen, float32(mx), float32(my), r, 1, color.RGBA{255, 255, 255, 140}, true)
	}

	g.drawSidebar(screen, mx, my)
	g.drawStatus(screen, mx, my)
	if g.help {
		g.drawHelp(screen)
	}
}

var whitePixel = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}()

// drawWind draws the air-flow field as one batched set of line segments.
func (g *Game) drawWind(screen *ebiten.Image) {
	var vs []ebiten.Vertex
	var is []uint16
	flush := func() {
		if len(is) > 0 {
			screen.DrawTriangles(vs, is, whitePixel, &ebiten.DrawTrianglesOptions{})
		}
		vs, is = vs[:0], is[:0]
	}
	for j := 0; j < sim.FH; j++ {
		for i := 0; i < sim.FW; i++ {
			if g.sim.Wind.Solid(i, j) {
				continue
			}
			u, v := g.sim.Wind.U[j*sim.FW+i], g.sim.Wind.V[j*sim.FW+i]
			mag := float32(math.Hypot(float64(u), float64(v)))
			if mag < 0.03 {
				continue
			}
			cx := float32((i*sim.CS + sim.CS/2) * scale)
			cy := float32((j*sim.CS + sim.CS/2) * scale)
			var p vector.Path
			p.MoveTo(cx, cy)
			p.LineTo(cx+u*6+0.01, cy+v*6)
			n := len(vs)
			vs, is = p.AppendVerticesAndIndicesForStroke(vs, is, &vector.StrokeOptions{Width: 1.2})
			a := float32(math.Min(1, float64(0.35+mag*0.5)))
			for k := n; k < len(vs); k++ {
				vs[k].SrcX, vs[k].SrcY = 1, 1
				vs[k].ColorR, vs[k].ColorG, vs[k].ColorB, vs[k].ColorA = 0.47*a, 0.86*a, a, a
			}
			if len(vs) > 60000 {
				flush()
			}
		}
	}
	flush()
}

func (g *Game) drawSidebar(screen *ebiten.Image, mx, my int) {
	vector.DrawFilledRect(screen, simW, 0, sideW, screenH, colPanel, false)
	g.text(screen, "ELEMENT SANDBOX", simW+8, 5, colAccent)
	for _, h := range g.headers {
		g.text(screen, h.label, simW+4, h.y, colDim)
	}
	hover := -1
	for k, e := range g.entries {
		x, y := float32(e.x), float32(e.y)
		bg := color.RGBA{36, 39, 50, 255}
		if mx >= e.x && mx < e.x+btnW && my >= e.y && my < e.y+btnH {
			bg = color.RGBA{52, 56, 72, 255}
			hover = k
		}
		vector.DrawFilledRect(screen, x, y, btnW, btnH, bg, false)
		vector.DrawFilledRect(screen, x+2, y+3, 9, 9, e.col, false)
		tc := colText
		if k == g.sel {
			vector.StrokeRect(screen, x+0.5, y+0.5, btnW-1, btnH-1, 1, colAccent, false)
			tc = colAccent
		}
		g.text(screen, e.label, e.x+14, e.y+1, tc)
	}
	show := g.sel
	if hover >= 0 {
		show = hover
	}
	y := g.infoY
	g.text(screen, g.entries[show].label, simW+6, y, colAccent)
	for _, line := range wrap(g.entries[show].desc, 26) {
		y += 13
		g.text(screen, line, simW+6, y, colText)
	}
}

func (g *Game) drawStatus(screen *ebiten.Image, mx, my int) {
	vector.DrawFilledRect(screen, 0, simH, simW, statusH, colPanel, false)
	wind := "calm"
	if gw := g.sim.Wind.Global; gw > 0 {
		wind = fmt.Sprintf("%.1f ->", gw)
	} else if gw < 0 {
		wind = fmt.Sprintf("<- %.1f", -gw)
	}
	state := "RUNNING"
	if g.paused {
		state = "PAUSED (N = step)"
	}
	top := "open"
	if !g.sim.OpenTop {
		top = "closed"
	}
	view := "normal"
	if g.view == sim.ViewHeat {
		view = "heat"
	}
	s := fmt.Sprintf("%s | Brush %d | Wind %s | Sky %s | View %s | H = help | %.0f FPS",
		state, g.brush, wind, top, view, ebiten.ActualFPS())
	g.text(screen, s, 8, simH+7, colText)
	if mx >= 0 && mx < simW && my >= 0 && my < simH {
		c := g.sim.At(mx/scale, my/scale)
		cell := fmt.Sprintf("%s  %.0f C", sim.Info(c.M).Name, c.T)
		g.text(screen, cell, simW-len(cell)*7-8, simH+7, colAccent)
	}
}

var helpLines = []string{
	"ELEMENT SANDBOX - controls",
	"",
	"Left mouse      paint selected material / use tool",
	"Right mouse     erase",
	"Shift + paint   replace existing material",
	"Mouse wheel [ ] brush size",
	"Left / Right    global wind  (Down = calm)",
	"Space           pause    N  single step",
	"T               heat (temperature) view",
	"V               show wind / air flow",
	"O               open / close the sky for gases",
	"C               clear everything",
	"F11             fullscreen",
	"H / F1          show / hide this help",
	"",
	"Try: gasoline vs diesel with Fire, salt on ice,",
	"water on lava, thermite on iron, hydrogen + fire,",
	"dry ice fog, steam rising and raining back down.",
}

func (g *Game) drawHelp(screen *ebiten.Image) {
	w, h := float32(400), float32(len(helpLines)*14+20)
	x, y := float32(simW)/2-w/2, float32(simH)/2-h/2
	vector.DrawFilledRect(screen, x, y, w, h, color.RGBA{10, 12, 18, 225}, false)
	vector.StrokeRect(screen, x, y, w, h, 1, colAccent, false)
	for k, l := range helpLines {
		c := colText
		if k == 0 {
			c = colAccent
		}
		g.text(screen, l, int(x)+14, int(y)+10+k*14, c)
	}
}

func wrap(s string, n int) []string {
	var lines []string
	cur := ""
	for _, w := range strings.Fields(s) {
		if cur != "" && len(cur)+1+len(w) > n {
			lines = append(lines, cur)
			cur = w
			continue
		}
		if cur != "" {
			cur += " "
		}
		cur += w
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

func (g *Game) Layout(int, int) (int, int) { return screenW, screenH }

func main() {
	w, h := screenW, screenH
	if mw, mh := ebiten.Monitor().Size(); mw > 0 && mh > 0 {
		f := math.Min(1, math.Min(float64(mw)*0.95/float64(w), float64(mh)*0.88/float64(h)))
		w, h = int(float64(w)*f), int(float64(h)*f)
	}
	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowTitle("Element Sandbox")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(newGame()); err != nil {
		log.Fatal(err)
	}
}
