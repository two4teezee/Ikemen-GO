package main

import (
	"fmt"
	"strings"
)

type MotifWin struct {
	winEnabled      bool
	loseEnabled     bool
	active          bool
	initialized     bool
	counter         int32
	endTimer        int32
	fadeIn          *Fade
	fadeOut         *Fade
	stateDone       bool
	soundsEnabled   bool
	fadeOutTime     int32
	time            int32
	keyCancel       []string
	p1State         []int32
	p1TeammateState []int32
	p2State         []int32
	p2TeammateState []int32
	stateTime       int32
	//winCount        int32
	//loseCnt         int32
}

// Assign state data to MotifWin
func (wi *MotifWin) assignStates(p1, p1Teammate, p2, p2Teammate []int32) {
	wi.p1State = p1
	wi.p1TeammateState = p1Teammate
	wi.p2State = p2
	wi.p2TeammateState = p2Teammate
}

func (wi *MotifWin) reset(m *Motif) {
	wi.active = false
	wi.initialized = false
	wi.stateDone = false
	wi.endTimer = -1
}

// Initialize the MotifWin based on the current game mode
func (wi *MotifWin) init(m *Motif) {
	if (wi.winEnabled && sys.winnerTeam() != 0 && sys.winnerTeam() != int32(sys.home)+1) ||
		(wi.loseEnabled && (sys.winnerTeam() == 0 || sys.winnerTeam() == int32(sys.home)+1)) {
		if ok := wi.initSurvival(m); ok {
		} else if ok := wi.initTimeAttack(m); ok {
		} else if ok := wi.initWinScreen(m); ok {
		} else {
			wi.initialized = true
			return
		}
	} else {
		wi.initialized = true
		return
	}

	if wi.soundsEnabled {
		sys.clearAllSound()
		sys.noSoundFlg = true
	}

	m.Music.Play("results", sys.motif.Def, false)

	wi.fadeIn.init(m.fadeIn, true)
	wi.counter = 0
	wi.active = true
	wi.initialized = true
}

// Handle survival mode initialization
func (wi *MotifWin) initSurvival(m *Motif) bool {
	if !strings.HasPrefix(sys.gameMode, "survival") || !m.SurvivalResultsScreen.Enabled {
		return false
	}

	m.SurvivalResultsBgDef.BGDef.Reset()

	m.SurvivalResultsScreen.WinsText.TextSpriteData.text = fmt.Sprintf(m.replaceFormatSpecifiers(m.SurvivalResultsScreen.WinsText.Text), sys.match-1)
	if sys.match >= m.SurvivalResultsScreen.RoundsToWin {
		wi.assignStates(m.SurvivalResultsScreen.P1.Win.State, m.SurvivalResultsScreen.P1.Teammate.Win.State, m.SurvivalResultsScreen.P2.Win.State, m.SurvivalResultsScreen.P2.Teammate.Win.State)
	} else {
		wi.assignStates(m.SurvivalResultsScreen.P1.State, m.SurvivalResultsScreen.P1.Teammate.State, m.SurvivalResultsScreen.P2.State, m.SurvivalResultsScreen.P2.Teammate.State)
	}
	wi.stateTime = m.SurvivalResultsScreen.State.Time
	wi.soundsEnabled = m.SurvivalResultsScreen.Sounds.Enabled

	wi.keyCancel = m.SurvivalResultsScreen.Cancel.Key
	wi.time = m.SurvivalResultsScreen.Show.Time
	wi.fadeOutTime = m.SurvivalResultsScreen.FadeOut.Time
	wi.fadeIn = m.SurvivalResultsScreen.FadeIn.FadeData
	wi.fadeOut = m.SurvivalResultsScreen.FadeOut.FadeData
	return true
}

// Handle time attack mode initialization
func (wi *MotifWin) initTimeAttack(m *Motif) bool {
	if sys.gameMode != "timeattack" || !m.TimeAttackResultsScreen.Enabled {
		return false
	}

	m.TimeAttackResultsBgDef.BGDef.Reset()

	m.TimeAttackResultsScreen.WinsText.TextSpriteData.text = FormatTimeText(m.TimeAttackResultsScreen.WinsText.Text, float64(timeTotal())/60)
	wi.assignStates(m.TimeAttackResultsScreen.P1.State, m.TimeAttackResultsScreen.P1.Teammate.State, m.TimeAttackResultsScreen.P2.State, m.TimeAttackResultsScreen.P2.Teammate.State)
	wi.stateTime = m.TimeAttackResultsScreen.State.Time
	wi.soundsEnabled = m.TimeAttackResultsScreen.Sounds.Enabled

	wi.keyCancel = m.TimeAttackResultsScreen.Cancel.Key
	wi.time = m.TimeAttackResultsScreen.Show.Time
	wi.fadeOutTime = m.TimeAttackResultsScreen.FadeOut.Time
	wi.fadeIn = m.TimeAttackResultsScreen.FadeIn.FadeData
	wi.fadeOut = m.TimeAttackResultsScreen.FadeOut.FadeData
	return true
}

// Handle win screen mode initialization
func (wi *MotifWin) initWinScreen(m *Motif) bool {
	if sys.home != 1 || !m.WinScreen.Enabled {
		return false
	}

	m.WinBgDef.BGDef.Reset()

	wi.assignStates(m.WinScreen.P1.State, m.WinScreen.P1.Teammate.State, m.WinScreen.P2.State, m.WinScreen.P2.Teammate.State)
	wi.stateTime = m.WinScreen.State.Time
	wi.soundsEnabled = m.WinScreen.Sounds.Enabled

	wi.keyCancel = m.WinScreen.Cancel.Key
	wi.time = m.WinScreen.Pose.Time
	wi.fadeOutTime = m.WinScreen.FadeOut.Time
	wi.fadeIn = m.WinScreen.FadeIn.FadeData
	wi.fadeOut = m.WinScreen.FadeOut.FadeData
	return true
}

// Process the step logic for MotifWin
func (wi *MotifWin) step(m *Motif) {
	if wi.endTimer == -1 {
		cancel := m.button(wi.keyCancel, -1)
		if cancel || wi.counter == wi.time {
			startFadeOut(wi.fadeOut, m.fadeOut, cancel, m.fadePolicy)
			wi.endTimer = wi.counter + m.fadeOut.timeRemaining
		}
	}

	// Handle state transitions
	if !wi.stateDone && wi.counter >= wi.stateTime {
		m.processStateTransitions(wi.p1State, wi.p1TeammateState, wi.p2State, wi.p2TeammateState)
		wi.stateDone = true
	}

	// Check if the sequence has ended
	if wi.endTimer != -1 && wi.counter >= wi.endTimer {
		if m.fadeOut != nil {
			m.fadeOut.reset()
		}
		wi.active = false
		if !wi.soundsEnabled {
			sys.noSoundFlg = false
		}
		return
	}

	// Increment counter
	wi.counter++
}

func (wi *MotifWin) draw(m *Motif, layerno int16) {
	if strings.HasPrefix(sys.gameMode, "survival") {
		if m.SurvivalResultsBgDef.BgClearColor[0] >= 0 {
			m.SurvivalResultsBgDef.RectData.Draw(layerno)
		}
		m.SurvivalResultsBgDef.BGDef.Draw(int32(layerno), 0, 0, 1)
		m.SurvivalResultsScreen.Overlay.RectData.Draw(layerno)
		if wi.counter >= m.SurvivalResultsScreen.WinsText.DisplayTime {
			m.SurvivalResultsScreen.WinsText.TextSpriteData.Draw(layerno)
		}
	} else if sys.gameMode == "timeattack" {
		if m.TimeAttackResultsBgDef.BgClearColor[0] >= 0 {
			m.TimeAttackResultsBgDef.RectData.Draw(layerno)
		}
		m.TimeAttackResultsBgDef.BGDef.Draw(int32(layerno), 0, 0, 1)
		m.TimeAttackResultsScreen.Overlay.RectData.Draw(layerno)
		if wi.counter >= m.TimeAttackResultsScreen.WinsText.DisplayTime {
			m.TimeAttackResultsScreen.WinsText.TextSpriteData.Draw(layerno)
		}
	} else {
		if m.WinBgDef.BgClearColor[0] >= 0 {
			m.WinBgDef.RectData.Draw(layerno)
		}
		m.WinBgDef.BGDef.Draw(int32(layerno), 0, 0, 1)
		m.WinScreen.Overlay.RectData.Draw(layerno)
		if wi.counter >= m.WinScreen.WinText.DisplayTime {
			m.WinScreen.WinText.TextSpriteData.Draw(layerno)
		}
	}
}
