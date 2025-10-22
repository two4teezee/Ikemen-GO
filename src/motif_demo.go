package main

import (
// "fmt"
)

type MotifDemo struct {
	enabled     bool
	active      bool
	initialized bool
	counter     int32
	endTimer    int32
}

func (de *MotifDemo) reset(m *Motif) {
	de.active = false
	de.initialized = false
	de.endTimer = -1
}

func (de *MotifDemo) init(m *Motif) {
	if !m.DemoMode.Enabled || !de.enabled || sys.gameMode != "demo" {
		de.initialized = true
		return
	}

	de.counter = 0

	// Override lifebar fading
	m.DemoMode.FadeIn.FadeData.init(sys.lifebar.ro.fadeIn, true)

	de.active = true
	de.initialized = true
}

func (de *MotifDemo) step(m *Motif) {
	if de.endTimer == -1 {
		cancel := (m.AttractMode.Enabled && sys.credits > 0) ||
			(!m.AttractMode.Enabled && (sys.anyHardButton() || sys.button("m") >= 0 || sys.button("s") >= 0))
		if de.counter == m.DemoMode.Fight.EndTime || cancel {
			startFadeOut(m.DemoMode.FadeOut.FadeData, sys.lifebar.ro.fadeOut, cancel, m.fadePolicy)
			de.endTimer = de.counter + sys.lifebar.ro.fadeOut.timeRemaining
		}
	}

	// Check if the sequence has ended
	if de.endTimer != -1 && de.counter >= de.endTimer {
		if sys.lifebar.ro.fadeOut != nil {
			sys.lifebar.ro.fadeOut.reset()
		}
		de.active = false
		sys.endMatch = true
		return
	}

	// Increment counter
	de.counter++
}

func (de *MotifDemo) draw(m *Motif, layerno int16) {
	// nothing to draw, may be expanded in future
}
