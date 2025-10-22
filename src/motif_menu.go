package main

import (
// "fmt"
)

type MotifMenu struct {
	enabled     bool
	active      bool
	initialized bool
	counter     int32
	endTimer    int32
}

func (me *MotifMenu) reset(m *Motif) {
	me.active = false
	me.initialized = false
	me.endTimer = -1
}

func (me *MotifMenu) init(m *Motif) {
	if !m.MenuInfo.Enabled || !me.enabled {
		me.initialized = true
		return
	}
	if (!sys.esc && sys.button("m") == -1) || m.ch.active {
		return
	}

	if err := sys.luaLState.DoString("menuInit()"); err != nil {
		sys.luaLState.RaiseError("Error executing Lua code: %v\n", err.Error())
	}

	m.MenuInfo.FadeIn.FadeData.init(m.fadeIn, true)
	me.counter = 0
	me.active = true
	me.initialized = true
}

func (me *MotifMenu) step(m *Motif) {
	if me.endTimer == -1 && (sys.keyInput == KeyUnknown || sys.endMatch) {
		startFadeOut(m.MenuInfo.FadeOut.FadeData, m.fadeOut, false, m.fadePolicy)
		me.endTimer = me.counter + m.fadeOut.timeRemaining
	}

	// Check if the sequence has ended
	if me.endTimer != -1 && me.counter >= me.endTimer {
		if m.fadeOut != nil {
			m.fadeOut.reset()
		}
		me.active = false
		me.reset(m)
		sys.paused = false
		return
	}

	// Increment counter
	me.counter++
}

func (me *MotifMenu) draw(m *Motif, layerno int16) {
	if layerno == 2 {
		//if ok, err := ExecFunc(sys.luaLState, "menuRun"); err != nil {
		//	sys.luaLState.RaiseError("Error executing Lua function: %v\n", err.Error())
		//} else if !ok {
		//	me.reset(m)
		//}
		if err := sys.luaLState.DoString("menuRun()"); err != nil {
			sys.luaLState.RaiseError("Error executing Lua code: %v\n", err.Error())
		}
	}
}
