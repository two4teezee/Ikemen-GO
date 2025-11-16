package main

import (
// "fmt"
)

type MotifChallenger struct {
	enabled       bool
	active        bool
	initialized   bool
	counter       int32
	endTimer      int32
	controllerNo  int
	lifebarActive bool
}

func (ch *MotifChallenger) reset(m *Motif) {
	ch.active = false
	ch.initialized = false
	ch.endTimer = -1
	ch.controllerNo = -1
}

func (ch *MotifChallenger) init(m *Motif) {
	if !m.ChallengerInfo.Enabled || !ch.enabled {
		ch.initialized = true
		return
	}

	controllerNo := m.buttonController(m.ChallengerInfo.Key)
	if controllerNo == -1 || controllerNo == sys.chars[0][0].controller {
		return
	}
	ch.controllerNo = controllerNo

	if m.AttractMode.Enabled && sys.credits > 0 {
		sys.credits--
	}

	ch.lifebarActive = sys.lifebar.active
	sys.lifebar.active = false

	m.ChallengerBgDef.BGDef.Reset()
	m.ChallengerInfo.Bg.AnimData.Reset()

	m.ChallengerInfo.FadeIn.FadeData.init(m.fadeIn, true)
	ch.counter = 0
	ch.active = true
	ch.initialized = true
}

func (ch *MotifChallenger) step(m *Motif) {
	if ch.endTimer == -1 && ch.counter == m.ChallengerInfo.Time {
		startFadeOut(m.ChallengerInfo.FadeOut.FadeData, m.fadeOut, false, m.fadePolicy)
		ch.endTimer = ch.counter + m.fadeOut.timeRemaining
	}
	sys.setGSF(GSF_nobardisplay)
	sys.setGSF(GSF_nomusic)
	sys.setGSF(GSF_timerfreeze)
	if ch.counter == m.ChallengerInfo.Pause.Time {
		sys.pausetime = m.ChallengerInfo.Time + m.ChallengerInfo.FadeOut.Time
	}
	if ch.counter == m.ChallengerInfo.Snd.Time {
		m.Snd.play(m.ChallengerInfo.Snd.Snd, 100, 0, 0, 0, 0)
	}
	if ch.counter >= m.ChallengerInfo.Bg.Displaytime {
		m.ChallengerInfo.Bg.AnimData.Update()
	}

	//if ch.endTimer != -1 && ch.counter + 2 >= ch.endTimer {
	//	sys.endMatch = true
	//}

	// Check if the sequence has ended
	if ch.endTimer != -1 && ch.counter >= ch.endTimer {
		if m.fadeOut != nil {
			m.fadeOut.reset()
		}
		ch.active = false
		sys.lifebar.active = ch.lifebarActive
		sys.endMatch = true
		return
	}

	// Increment counter
	ch.counter++
}

func (ch *MotifChallenger) draw(m *Motif, layerno int16) {
	m.ChallengerInfo.Overlay.RectData.Draw(layerno)
	if m.ChallengerBgDef.BgClearColor[0] >= 0 {
		m.ChallengerBgDef.RectData.Draw(layerno)
	}
	m.ChallengerBgDef.BGDef.Draw(int32(layerno), 0, 0, 1)
	if ch.counter >= m.ChallengerInfo.Text.Displaytime {
		m.ChallengerInfo.Text.TextSpriteData.Draw(layerno)
	}
	if ch.counter >= m.ChallengerInfo.Bg.Displaytime {
		m.ChallengerInfo.Bg.AnimData.Draw(layerno)
	}
}
