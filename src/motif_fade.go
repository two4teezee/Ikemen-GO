package main

import (
// "fmt"
)

type Fade struct {
	active        bool
	time          int32
	col           [3]int32
	colEncoded    uint32
	animData      *Anim
	isFadeIn      bool
	timeRemaining int32
	snd           [2]int32
}

func newFade() *Fade {
	return &Fade{
		time: 30,
		snd:  [2]int32{-1, 0},
	}
}

func (fa *Fade) reset() {
	fa.timeRemaining = fa.time
	fa.active = false
	if fa.animData != nil {
		fa.animData.Reset()
	}
}

func (fa *Fade) init(fade *Fade, isFadeIn bool) {
	fa.reset()
	fa.colEncoded = uint32(fa.col[0]&0xff<<16 | fa.col[1]&0xff<<8 | fa.col[2]&0xff)
	fa.isFadeIn = isFadeIn
	fa.active = true
	*fade = *fa
}

func (fa *Fade) step() {
	if !fa.active || (sys.gameRunning && !sys.tickFrame() && !sys.motif.me.active) {
		return
	}
	if fa.animData != nil {
		fa.animData.Update()
	}
	fa.timeRemaining--
	if fa.timeRemaining < 0 {
		fa.reset()
	}
}

func (fa *Fade) drawRect(rect [4]int32, color uint32, alpha int32) {
	src := alpha>>uint(Btoi(sys.clsnDisplay)) + Btoi(sys.clsnDisplay)*128
	dst := 255 - src
	FillRect(rect, color, [2]int32{src, dst})
}

func (fa *Fade) draw() {
	if !fa.active || fa.timeRemaining < 0 || fa.time <= 0 {
		return
	}
	if fa.animData != nil && fa.animData.anim != nil {
		fa.animData.Draw(fa.animData.layerno)
	} else if fa.isFadeIn {
		fa.drawRect(sys.scrrect, fa.colEncoded, 256*fa.timeRemaining/fa.time)
	} else {
		fa.drawRect(sys.scrrect, fa.colEncoded, 256*(fa.time-fa.timeRemaining)/fa.time)
	}
}

func (fa *Fade) isActive() bool {
	return fa.active && fa.timeRemaining > 0
}

// Policies for starting a new fade when a fade-in might be active.
//   - Continue  : start now, preserving the current visual darkness.
//   - Replace   : start now from full length, cancelling any fade-in immediately.
//   - Wait      : start after the current fade-in completes (no overlap, full length).
//   - Stop      : interrupt both current fade-in and skip fade-out.
type FadeStartPolicy int

const (
	FadeReplace FadeStartPolicy = iota
	FadeContinue
	FadeWait
	FadeStop
)

// startFadeOut starts an outgoing fade using the given policy.
// - overrideBlack: treat caller as a user interruption and force immediate cut for FadeStop.
// - policy: controls how an in-progress fade-in is handled.
func startFadeOut(tmpl *Fade, dest *Fade, overrideBlack bool, policy FadeStartPolicy) {
	// On user interruption, force black/no-anim fade request.
	if overrideBlack {
		tmpl.col = [3]int32{0, 0, 0}
		tmpl.animData = nil
	}
	fi := sys.motif.fadeIn

	// FadeStop semantics:
	// If this is an explicit user interruption OR a fade-in is active, cut immediately.
	if policy == FadeStop && (overrideBlack || (fi != nil && fi.isActive())) {
		if fi != nil {
			fi.reset()
		}
		if dest != nil {
			dest.reset()
			dest.timeRemaining = 0
		}
		return
	}

	// Helper to (re)start a full-length fade-out.
	startFresh := func() { tmpl.init(dest, false) }

	// If no fade-in is active, all policies behave the same here: start now.
	if fi == nil || !fi.isActive() {
		startFresh()
		return
	}

	switch policy {
	case FadeReplace:
		fi.reset()
		startFresh()
	case FadeWait:
		// Defer: caller should call again after fade-in completes.
	case FadeContinue, FadeStop:
		startFresh()
		// Match current darkness, then cancel fade-in.
		trfi := fi.timeRemaining
		if trfi < 0 {
			trfi = 0
		}
		if fi.time > 0 && dest.time > 0 {
			dest.timeRemaining = dest.time - (dest.time*trfi)/fi.time
			if dest.timeRemaining < 0 {
				dest.timeRemaining = 0
			}
			if dest.timeRemaining > dest.time {
				dest.timeRemaining = dest.time
			}
		}
		fi.reset()
	}
}
