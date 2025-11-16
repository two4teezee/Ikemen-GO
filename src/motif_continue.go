package main

import (
	"fmt"
	"sort"
)

type MotifContinue struct {
	enabled     bool
	active      bool
	initialized bool
	counter     int32
	endTimer    int32
	credits     int32
	yesSide     bool
	selected    bool
	counts      []string
	pn          int
}

func (co *MotifContinue) reset(m *Motif) {
	sys.continueFlg = false
	co.active = false
	co.initialized = false
	co.yesSide = true
	co.selected = false
	co.endTimer = -1
}

func (co *MotifContinue) extractAndSortKeysDescending(m *Motif) []string {
	keys := make([]string, 0, len(m.ContinueScreen.Counter.MapCounts))
	for key := range m.ContinueScreen.Counter.MapCounts {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	return keys
}

func (co *MotifContinue) updateCreditsText(m *Motif) {
	formattedText := fmt.Sprintf(m.replaceFormatSpecifiers(m.ContinueScreen.Credits.Text), sys.credits)
	m.ContinueScreen.Credits.TextSpriteData.text = formattedText
	co.credits = sys.credits
}

func (co *MotifContinue) init(m *Motif) {
	if (!m.ContinueScreen.Enabled || !co.enabled || sys.cfg.Options.QuickContinue) ||
		(sys.winnerTeam() != 0 && sys.winnerTeam() != int32(sys.home)+1) {
		co.initialized = true
		return
	}

	co.pn = 1 // TODO: Initialize pn appropriately

	// Extract and sort keys in descending order
	co.counts = co.extractAndSortKeysDescending(m)

	m.ContinueBgDef.BGDef.Reset()

	m.ContinueScreen.Continue.TextSpriteData.Reset()
	m.ContinueScreen.Continue.TextSpriteData.AddPos(m.ContinueScreen.Pos[0], m.ContinueScreen.Pos[1])

	m.ContinueScreen.Yes.TextSpriteData.Reset()
	m.ContinueScreen.Yes.TextSpriteData.AddPos(m.ContinueScreen.Pos[0], m.ContinueScreen.Pos[1])

	m.ContinueScreen.Yes.Active.TextSpriteData.Reset()
	m.ContinueScreen.Yes.Active.TextSpriteData.AddPos(m.ContinueScreen.Pos[0], m.ContinueScreen.Pos[1])

	m.ContinueScreen.No.TextSpriteData.Reset()
	m.ContinueScreen.No.TextSpriteData.AddPos(m.ContinueScreen.Pos[0], m.ContinueScreen.Pos[1])

	m.ContinueScreen.No.Active.TextSpriteData.Reset()
	m.ContinueScreen.No.Active.TextSpriteData.AddPos(m.ContinueScreen.Pos[0], m.ContinueScreen.Pos[1])

	m.ContinueScreen.Counter.AnimData.Reset()
	//m.ContinueScreen.Counter.AnimData.Update()

	co.updateCreditsText(m)

	// Handle state transitions
	m.processStateTransitions(m.ContinueScreen.P2.State, m.ContinueScreen.P2.Teammate.State, m.ContinueScreen.P1.State, m.ContinueScreen.P1.Teammate.State)

	co.yesSide = true

	if m.ContinueScreen.Sounds.Enabled {
		sys.clearAllSound()
		sys.noSoundFlg = true
	}

	m.Music.Play("continue", sys.motif.Def, false)

	m.ContinueScreen.FadeIn.FadeData.init(m.fadeIn, true)
	co.counter = 0
	co.active = true
	co.initialized = true
}

func (co *MotifContinue) processSelection(m *Motif, continueSelected bool) {
	cs := m.ContinueScreen
	if continueSelected {
		m.processStateTransitions(
			cs.P2.Yes.State,
			cs.P2.Teammate.Yes.State,
			cs.P1.Yes.State,
			cs.P1.Teammate.Yes.State,
		)
		sys.continueFlg = true
		if sys.credits != -1 {
			sys.credits--
		}
	} else {
		m.processStateTransitions(
			cs.P2.No.State,
			cs.P2.Teammate.No.State,
			cs.P1.No.State,
			cs.P1.Teammate.No.State,
		)
	}
	startFadeOut(m.ContinueScreen.FadeOut.FadeData, m.fadeOut, false, m.fadePolicy)
	co.endTimer = co.counter + m.fadeOut.timeRemaining
	co.selected = true
}

func (co *MotifContinue) skipCounter(m *Motif) {
	for _, key := range co.counts {
		properties := m.ContinueScreen.Counter.MapCounts[key]
		if co.counter < properties.SkipTime {
			for co.counter < properties.SkipTime {
				co.counter++
				m.ContinueScreen.Counter.AnimData.Update()
			}
			break
		}
	}
}

func (co *MotifContinue) playCounterSounds(m *Motif) {
	for _, key := range co.counts {
		properties := m.ContinueScreen.Counter.MapCounts[key]
		if co.counter == properties.SkipTime {
			m.Snd.play(properties.Snd, 100, 0, 0, 0, 0)
			break
		}
	}
}

func (co *MotifContinue) step(m *Motif) {
	if co.credits != sys.credits {
		co.updateCreditsText(m)
		if !co.selected {
			co.counter = 0
			m.ContinueScreen.Counter.AnimData.Reset()
		}
	}

	if !co.selected {
		m.ContinueScreen.Counter.AnimData.Update()
		if m.ContinueScreen.LegacyMode.Enabled {
			if m.button(m.ContinueScreen.Move.Key, co.pn-1) {
				m.Snd.play(m.ContinueScreen.Move.Snd, 100, 0, 0, 0, 0)
				co.yesSide = !co.yesSide
			} else if m.button(m.ContinueScreen.Skip.Key, co.pn-1) || m.button(m.ContinueScreen.Done.Key, co.pn-1) {
				m.Snd.play(m.ContinueScreen.Done.Snd, 100, 0, 0, 0, 0)
				co.processSelection(m, co.yesSide)
			}
		} else {
			if co.counter < m.ContinueScreen.Counter.End.SkipTime {
				if (sys.credits == -1 || sys.credits > 0) && m.button(m.ContinueScreen.Done.Key, co.pn-1) {
					m.Snd.play(m.ContinueScreen.Done.Snd, 100, 0, 0, 0, 0)
					co.processSelection(m, true)
				} else if m.button(m.ContinueScreen.Skip.Key, co.pn-1) &&
					co.counter >= m.ContinueScreen.Counter.StartTime+m.ContinueScreen.Counter.SkipStart {
					co.skipCounter(m)
				}
				co.playCounterSounds(m)
			} else if co.counter == m.ContinueScreen.Counter.End.SkipTime {
				m.Snd.play(m.ContinueScreen.Counter.End.Snd, 100, 0, 0, 0, 0)
				co.processSelection(m, false)
			}
		}
	}

	// Check if the sequence has ended
	if co.selected && co.endTimer != -1 && co.counter >= co.endTimer {
		if m.fadeOut != nil {
			m.fadeOut.reset()
		}
		co.active = false
		if !m.ContinueScreen.Sounds.Enabled {
			sys.noSoundFlg = false
		}
		return
	}
	// Increment counter
	co.counter++
}

func (co *MotifContinue) drawLegacyMode(m *Motif, layerno int16) {
	// Continue
	m.ContinueScreen.Continue.TextSpriteData.Draw(layerno)
	// Yes / No
	if co.yesSide {
		m.ContinueScreen.Yes.Active.TextSpriteData.Draw(layerno)
		m.ContinueScreen.No.TextSpriteData.Draw(layerno)
	} else {
		m.ContinueScreen.Yes.TextSpriteData.Draw(layerno)
		m.ContinueScreen.No.Active.TextSpriteData.Draw(layerno)
	}
}

func (co *MotifContinue) draw(m *Motif, layerno int16) {
	// Overlay
	m.ContinueScreen.Overlay.RectData.Draw(layerno)
	// Background
	if m.ContinueBgDef.BgClearColor[0] >= 0 {
		m.ContinueBgDef.RectData.Draw(layerno)
	}
	m.ContinueBgDef.BGDef.Draw(int32(layerno), 0, 0, 1)
	// Mugen style
	if m.ContinueScreen.LegacyMode.Enabled {
		co.drawLegacyMode(m, layerno)
	} else if !co.selected {
		// Arcade style Counter
		m.ContinueScreen.Counter.AnimData.Draw(layerno)
	}
	// Credits
	if sys.credits != -1 && co.counter >= m.ContinueScreen.Counter.SkipStart {
		m.ContinueScreen.Credits.TextSpriteData.Draw(layerno)
	}
}
