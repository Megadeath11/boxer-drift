package main

import (
	"fmt"
	"maps"

	"elementsandbox/campaign"
	"elementsandbox/sim"
)

// maxUndo is how many steps Ctrl+Z can go back (each holds a copy of the
// world, a few MB).
const maxUndo = 15

// undoStep is everything needed to go back to just before one action.
type undoStep struct {
	world   *sim.Snapshot
	budgets []int // level palette budgets, in palette order
	placed  [sim.NumMats]int
	lframe  int
	hold    int
	camp    *campaign.State
}

// pushUndo remembers the game as it is now, before the player acts.
func (g *Game) pushUndo() {
	st := undoStep{world: g.sim.Snapshot(), placed: g.placed, lframe: g.lframe, hold: g.hold}
	for _, e := range g.entries {
		st.budgets = append(st.budgets, e.budget)
	}
	if g.level == levelCampaign && g.camp != nil {
		c := *g.camp
		c.Built = maps.Clone(g.camp.Built)
		st.camp = &c
	}
	if len(g.undo) == maxUndo {
		g.undo = append(g.undo[:0], g.undo[1:]...)
	}
	g.undo = append(g.undo, st)
}

// dropUndo forgets the last step (an action that turned out to do nothing).
func (g *Game) dropUndo() {
	if n := len(g.undo); n > 0 {
		g.undo = g.undo[:n-1]
	}
}

// clearUndo forgets the history (a new level, stage or world).
func (g *Game) clearUndo() { g.undo = nil }

// undoLast goes back to just before the player's last action.
func (g *Game) undoLast() {
	n := len(g.undo)
	if n == 0 {
		g.say("Nothing to undo")
		return
	}
	st := g.undo[n-1]
	g.undo = g.undo[:n-1]
	g.sim.Restore(st.world)
	if len(st.budgets) == len(g.entries) {
		for k := range g.entries {
			g.entries[k].budget = st.budgets[k]
		}
	}
	g.placed, g.lframe, g.hold = st.placed, st.lframe, st.hold
	if st.camp != nil && g.level == levelCampaign {
		*g.camp = *st.camp
	}
	g.drawing = false
	g.waitRelease = true
	if n-1 == 0 {
		g.say("Undone (that was the last step you can undo)")
	} else {
		g.say(fmt.Sprintf("Undone (%d more step(s) you can undo)", n-1))
	}
}
