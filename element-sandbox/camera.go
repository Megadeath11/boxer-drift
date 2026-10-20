package main

import (
	"fmt"
	"image/color"
	"math"

	"elementsandbox/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const maxZoom = 8

// ppc is screen pixels per world cell at the current zoom.
func (g *Game) ppc() float64 { return scale * g.zoom }

// toCell converts a screen position in the play area to a world cell.
func (g *Game) toCell(mx, my int) (int, int) {
	x := int(g.camX + float64(mx)/g.ppc())
	y := int(g.camY + float64(my)/g.ppc())
	return max(0, min(sim.W-1, x)), max(0, min(sim.H-1, y))
}

// toScreen converts a world position (in cells) to screen pixels.
func (g *Game) toScreen(x, y float64) (float32, float32) {
	return float32((x - g.camX) * g.ppc()), float32((y - g.camY) * g.ppc())
}

func (g *Game) clampCamera() {
	vw, vh := simW/g.ppc(), simH/g.ppc()
	g.camX = math.Max(0, math.Min(g.camX, sim.W-vw))
	g.camY = math.Max(0, math.Min(g.camY, sim.H-vh))
}

// zoomAt zooms by factor keeping the world point under (mx,my) fixed.
func (g *Game) zoomAt(mx, my int, factor float64) {
	wx := g.camX + float64(mx)/g.ppc()
	wy := g.camY + float64(my)/g.ppc()
	g.zoom = math.Max(1, math.Min(maxZoom, g.zoom*factor))
	g.camX = wx - float64(mx)/g.ppc()
	g.camY = wy - float64(my)/g.ppc()
	g.clampCamera()
}

func (g *Game) resetCamera() { g.zoom, g.camX, g.camY = 1, 0, 0 }

// updateCamera pans with WASD or by dragging with the middle mouse button,
// zooms with + / - (also Z / X), and resets with Home or 0.
func (g *Game) updateCamera(mx, my int) {
	step := 6 / g.zoom
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.camY -= step
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.camY += step
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.camX -= step
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.camX += step
	}
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	if (inpututil.IsKeyJustPressed(ebiten.KeyZ) && !ctrl) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) { // Ctrl+Z is undo
		g.zoomAt(simW/2, simH/2, 1.5)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyX) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
		g.zoomAt(simW/2, simH/2, 1/1.5)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) || inpututil.IsKeyJustPressed(ebiten.Key0) {
		g.resetCamera()
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
		if g.panning {
			g.camX -= float64(mx-g.panMX) / g.ppc()
			g.camY -= float64(my-g.panMY) / g.ppc()
		}
		g.panning, g.panMX, g.panMY = true, mx, my
	} else {
		g.panning = false
	}
	g.clampCamera()
}

// drawMinimap shows the whole world with the visible area outlined.
func (g *Game) drawMinimap(screen *ebiten.Image) {
	const mw = 150.0
	f := mw / sim.W
	mh := sim.H * f
	x0, y0 := float64(simW)-mw-8, float64(simH)-mh-8
	vector.DrawFilledRect(screen, float32(x0-2), float32(y0-2), float32(mw+4), float32(mh+4), color.RGBA{8, 10, 14, 220}, false)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(f, f)
	op.GeoM.Translate(x0, y0)
	screen.DrawImage(g.img, op) // g.img still holds this frame's world
	vw, vh := simW/g.ppc(), simH/g.ppc()
	vector.StrokeRect(screen, float32(x0+g.camX*f), float32(y0+g.camY*f), float32(vw*f), float32(vh*f), 1, colAccent, false)
}

// drawClimate is the season / temperature / pollution panel.
func (g *Game) drawClimate(screen *ebiten.Image) {
	cl := &g.sim.Climate
	const w, h = 176, 58
	x, y := float32(simW-w-8), float32(8)
	vector.DrawFilledRect(screen, x, y, w, h, color.RGBA{10, 12, 18, 200}, false)

	season := "No seasons"
	if cl.Enabled {
		season = sim.SeasonNames[cl.Season()]
		if cl.Locked {
			season += " (locked)"
		}
	}
	g.text(screen, fmt.Sprintf("%s  %s", season, degrees(g.sim.Ambient)), int(x)+8, int(y)+5, colText)
	if cl.Enabled { // year dial
		for k := 0; k < 4; k++ {
			c := []color.RGBA{{120, 200, 90, 255}, {240, 200, 60, 255}, {220, 120, 40, 255}, {170, 200, 240, 255}}[k]
			vector.DrawFilledRect(screen, x+8+float32(k)*40, y+21, 39, 4, c, false)
		}
		vector.DrawFilledRect(screen, x+8+float32(cl.Phase)*160-1, y+18, 3, 10, colText, false)
	}

	ppm := cl.PPM
	col := color.RGBA{110, 220, 120, 255}
	label := "clean"
	switch {
	case ppm > sim.PPMUnhealthy:
		col, label = color.RGBA{255, 90, 70, 255}, "toxic"
	case ppm > 600:
		col, label = color.RGBA{250, 200, 70, 255}, "polluted"
	}
	if lim := g.ppmLimit(); lim > 0 && ppm > lim {
		col = color.RGBA{255, 140, 70, 255}
	}
	g.text(screen, fmt.Sprintf("CO2 %4.0f ppm  %s", ppm, label), int(x)+8, int(y)+31, col)
	// meter from 280 (pre-industrial) to 2000 ppm
	f := float32(math.Min(1, math.Max(0, float64(ppm-280)/1720)))
	vector.DrawFilledRect(screen, x+8, y+47, 160, 4, color.RGBA{40, 44, 56, 255}, false)
	vector.DrawFilledRect(screen, x+8, y+47, 160*f, 4, col, false)
	if lim := g.ppmLimit(); lim > 0 {
		lx := x + 8 + 160*float32((lim-280)/1720)
		vector.DrawFilledRect(screen, lx, y+44, 2, 10, colText, false)
	}
}

func (g *Game) ppmLimit() float32 {
	switch {
	case g.level >= 0:
		return levelAt(g.level).MaxPPM
	case g.level == levelCampaign && !g.campFree:
		return g.camp.Cur().MaxPPM
	}
	return 0
}
