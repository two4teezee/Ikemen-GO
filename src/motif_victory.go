package main

import (
	"fmt"
	"math/rand"
	"unicode/utf8"
)

type MotifVictory struct {
	enabled           bool
	active            bool
	initialized       bool
	counter           int32
	endTimer          int32
	stateDone         bool
	text              string
	lineFullyRendered bool
	charDelayCounter  int32
	typedCnt          int
}

func (vi *MotifVictory) reset(m *Motif) {
	vi.active = false
	vi.initialized = false
	vi.stateDone = false
	vi.lineFullyRendered = false
	vi.charDelayCounter = 0
	vi.typedCnt = 0
	// Victory screen uses its own typewriter logic, so disable the internal TextSprite typing.
	m.VictoryScreen.WinQuote.TextSpriteData.textDelay = 0
	vi.endTimer = -1
	vi.clear(m)
}

func (vi *MotifVictory) clearProps(props *PlayerVictoryProperties) {
	props.AnimData = NewAnim(nil, "")
	props.Face2.AnimData = NewAnim(nil, "")
	props.Name.TextSpriteData.text = ""
}

func (vi *MotifVictory) clear(m *Motif) {
	vi.clearProps(&m.VictoryScreen.P1)
	vi.clearProps(&m.VictoryScreen.P2)
	vi.clearProps(&m.VictoryScreen.P3)
	vi.clearProps(&m.VictoryScreen.P4)
	vi.clearProps(&m.VictoryScreen.P5)
	vi.clearProps(&m.VictoryScreen.P6)
	vi.clearProps(&m.VictoryScreen.P7)
	vi.clearProps(&m.VictoryScreen.P8)
}

func (vi *MotifVictory) getVictoryQuote(m *Motif) string {
	p := sys.chars[sys.winnerTeam()-1][0]
	quoteIndex := int(p.winquote)
	playerQuotes := sys.cgi[p.playerNo].quotes

	//fmt.Printf("[Victory] Winner team=%d playerNo=%d initialWinquote=%d\n", sys.winnerTeam(), p.playerNo, quoteIndex)

	// Check if the quote index is out of range
	if quoteIndex < 0 || quoteIndex >= MaxQuotes {
		// Collect available quote indices
		availableQuotes := []int{}
		for i, quote := range playerQuotes {
			if quote != "" {
				availableQuotes = append(availableQuotes, i)
			}
		}

		// Select a random available quote if any exist
		if len(availableQuotes) > 0 {
			quoteIndex = availableQuotes[rand.Intn(len(availableQuotes))]
		} else {
			quoteIndex = -1
		}
	}

	// Return the selected quote if valid, otherwise fall back to the default
	//fmt.Printf("[Victory] Using quoteIndex=%d (MaxQuotes=%d). Fallback text present=%v\n", quoteIndex, MaxQuotes, m.VictoryScreen.WinQuote.Text != "")
	if quoteIndex != -1 && len(playerQuotes) == MaxQuotes {
		return playerQuotes[quoteIndex]
	}
	return m.VictoryScreen.WinQuote.Text
}

// victoryEntry represents one slot to render on the victory screen
// (either a loaded character or not loaded Turns member).
type victoryEntry struct {
	side     int   // 0 or 1
	memberNo int   // 0-based team order
	cn       int   // index into sys.sel.charlist
	pal      int   // 1-based palette number
	c        *Char // non-nil if this member is currently loaded
}

// buildSideOrder reconstructs the list of members to render for a side
//   - Winner: last hitter leader, then other winners (alive first unless allowKO)
//   - Loser : first encountered leader, then other losers
//   - Fill with not loaded Turns members from the original selection
func (vi *MotifVictory) buildSideOrder(side int, allowKO bool, maxNum int) []victoryEntry {
	winnerSide := int(sys.winnerTeam() - 1)
	if maxNum <= 0 {
		//fmt.Printf("[Victory] buildSideOrder side=%d allowKO=%v maxNum=%d winnerSide=%d -> SKIP (num=0)\n", side, allowKO, maxNum, winnerSide)
		return nil
	}
	out := make([]victoryEntry, 0, maxNum)
	usedMember := map[int]bool{}
	//fmt.Printf("[Victory] buildSideOrder side=%d allowKO=%v maxNum=%d winnerSide=%d\n", side, allowKO, maxNum, winnerSide)

	// Helper to push a loaded char
	pushLoaded := func(c *Char) {
		if c == nil || int(c.teamside) != side {
			return
		}
		mn := int(c.memberNo)
		if usedMember[mn] || len(out) >= maxNum {
			return
		}
		out = append(out, victoryEntry{
			side:     side,
			memberNo: mn,
			cn:       int(c.selectNo),
			pal:      int(c.gi().palno),
			c:        c,
		})
		usedMember[mn] = true
		//fmt.Printf("[Victory] -> pushLoaded: side=%d memberNo=%d cn=%d pal=%d alive=%v leader=%v\n", side, mn, int(c.selectNo), int(c.gi().palno), c.alive(), len(out) == 1)
	}

	// 1) Choose leader
	if side == winnerSide {
		leaderPn := sys.lastHitter[side]
		if leaderPn < 0 {
			leaderPn = sys.teamLeader[side]
		}
		if leaderPn >= 0 && leaderPn < MaxPlayerNo && len(sys.chars[leaderPn]) > 0 {
			pushLoaded(sys.chars[leaderPn][0])
		}
	} else {
		// Loser: first encountered from this side
		for i := 0; i < MaxPlayerNo && len(out) < 1; i++ {
			if len(sys.chars[i]) == 0 {
				continue
			}
			if int(sys.chars[i][0].teamside) == side {
				pushLoaded(sys.chars[i][0])
				break
			}
		}
	}

	// 2) Append remaining loaded members from this side
	for i := 0; i < MaxPlayerNo && len(out) < maxNum; i++ {
		if len(sys.chars[i]) == 0 {
			continue
		}
		c := sys.chars[i][0]
		if int(c.teamside) != side {
			continue
		}
		// Skip if already used as leader
		if len(out) > 0 && out[0].c == c {
			continue
		}
		// Winner: prefer alive unless allowKO
		if side == winnerSide {
			if c.alive() || allowKO {
				pushLoaded(c)
			}
		} else {
			// Loser: include regardless of alive status (matches legacy loop)
			pushLoaded(c)
		}
	}

	// 3) Fill with un-loaded Turns team members from original select order
	if len(out) < maxNum {
		sel := sys.sel.selected[side]
		leaderMember := -1
		if len(out) > 0 {
			leaderMember = out[0].memberNo
		}
		for k := 0; k < len(sel) && len(out) < maxNum; k++ {
			if usedMember[k] {
				continue
			}
			if !allowKO && leaderMember != -1 && k <= leaderMember {
				continue
			}
			cn := int(sel[k][0])
			pl := int(sel[k][1])
			out = append(out, victoryEntry{
				side:     side,
				memberNo: k,
				cn:       cn,
				pal:      pl,
				c:        nil, // not loaded this round
			})
			usedMember[k] = true
		}
	}
	if len(out) > maxNum {
		//fmt.Printf("[Victory] Truncating out to %d (had %d)\n", maxNum, len(out))
		out = out[:maxNum]
	}
	return out
}

// buildSingleFrameFromSFF creates a 1-frame Animation from a raw sprite (grp, idx).
// Used when a motif references .spr (group/index) and the preloaded table lacks it.
func buildSingleFrameFromSFF(sff *Sff, grp, idx int32) *Animation {
	if sff == nil || sff.GetSprite(uint16(grp), uint16(idx)) == nil {
		return nil
	}
	anim := newAnimation(sff, &sff.palList)
	anim.mask = 0
	af := newAnimFrame()
	af.Group, af.Number = grp, idx
	af.Time = 1 // stable single-frame
	anim.frames = append(anim.frames, *af)
	return anim
}

// tryGetPortrait tries a sequence of (group,index) pairs first from preloaded
// SelectChar anims, then by building a single-frame Animation from the owner SFF.
// Returns the first non-nil *Animation and a label describing where it came from.
func tryGetPortrait(sc *SelectChar, ownerC *Char, pairs [][2]int32) (anim *Animation, from string) {
	for _, p := range pairs {
		grp, idx := p[0], p[1]
		if sc != nil {
			if a := sc.anims.get(grp, idx); a != nil {
				return a, fmt.Sprintf("preloaded(%d,%d)", grp, idx)
			}
		}
		if ownerC != nil && ownerC.playerNo >= 0 && ownerC.playerNo < len(sys.cgi) && sys.cgi[ownerC.playerNo].sff != nil {
			if a := buildSingleFrameFromSFF(sys.cgi[ownerC.playerNo].sff, grp, idx); a != nil {
				return a, fmt.Sprintf("sff(%d,%d)", grp, idx)
			}
		}
	}
	return nil, ""
}

// victoryPortraitAnim builds a *Anim for a character select entry and positions it.
// It uses per-character preloaded animations (sys.sel.charlist[cn].anims).
// If the requested anim/spr is missing, it falls back to (9000,1) then (9000,0).
func victoryPortraitAnim(m *Motif, sc *SelectChar, slot string,
	animNo int32, spr [2]int32,
	localcoord [2]int32, layerno int16, facing int32,
	scale [2]float32, window [4]int32,
	x, y float32, applyPal bool, pal int, ownerC *Char) *Anim {

	//fmt.Printf("[Victory] buildPortrait slot=%s scNil=%v animNo=%d spr=(%d,%d) pos=(%.1f,%.1f) scale=(%.3f,%.3f) localcoord=(%d,%d) window=(%d,%d,%d,%d) applyPal=%v pal=%d\n", slot, sc == nil, animNo, spr[0], spr[1], x, y, scale[0], scale[1], localcoord[0], localcoord[1], window[0], window[1], window[2], window[3], applyPal, pal)

	if sc == nil {
		return nil
	}
	var animCopy *Animation
	if animNo >= 0 {
		// First: explicit animation number
		animCopy = sc.anims.get(animNo, -1)
		if animCopy == nil {
			// if the specific anim is missing, try default big portrait
			if a, _ /*from*/ := tryGetPortrait(sc, ownerC, [][2]int32{{9000, 1} /*, {9000, 0}*/}); a != nil {
				animCopy = a
				//fmt.Printf("[Victory] slot=%s -> fallback from anim %d to %s\n", slot, animNo/*, from*/)
			}
		}
	} else if spr[0] >= 0 {
		// Try requested (grp,idx) first (preloaded or SFF-build), then fall back to 9000,1
		want := [][2]int32{{spr[0], spr[1]}, {9000, 1} /*, {9000, 0}*/}
		if a, _ /*from*/ := tryGetPortrait(sc, ownerC, want); a != nil {
			animCopy = a
		} else {
			// Detailed failure logs for the first requested pair
			if ownerC != nil && ownerC.playerNo >= 0 && ownerC.playerNo < len(sys.cgi) && sys.cgi[ownerC.playerNo].sff != nil {
				if sys.cgi[ownerC.playerNo].sff.GetSprite(uint16(spr[0]), uint16(spr[1])) == nil {
					//fmt.Printf("[Victory] slot=%s -> FAILED to build 1-frame anim: sprite not in SFF (spr=%d,%d)\n", slot, spr[0], spr[1])
				}
			} else {
				//fmt.Printf("[Victory] slot=%s -> owner SFF is nil; cannot build 1-frame anim (spr=%d,%d)\n", slot, spr[0], spr[1])
			}
		}
	}
	// Always return a non-nil *Anim. If we couldn't resolve a real anim, fall back to a safe dummy created by NewAnim.
	a := NewAnim(nil, "")
	if animCopy != nil {
		a.anim = animCopy
	} else {
		//fmt.Printf("[Victory] slot=%s -> animCopy=nil (animNo=%d spr=%d,%d). Check if your portraits are defined as an ANIM or plain SPR.\n", slot, animNo, spr[0], spr[1])
	}
	// Localcoord / window / layer / facing
	//a.SetLocalcoord(float32(localcoord[0]), float32(localcoord[1]))
	if localcoord[0] > 0 && localcoord[1] > 0 {
		a.SetLocalcoord(float32(localcoord[0]), float32(localcoord[1]))
	} else {
		//fmt.Printf("[Victory] slot=%s -> skip SetLocalcoord (0,0); using default engine localcoord\n", slot)
	}
	a.layerno = layerno
	a.SetFacing(float32(facing))
	//a.SetWindow([4]float32{float32(window[0]), float32(window[1]), float32(window[2]), float32(window[3])})
	if window[2] > window[0] && window[3] > window[1] {
		a.SetWindow([4]float32{float32(window[0]), float32(window[1]), float32(window[2]), float32(window[3])})
	} else {
		//fmt.Printf("[Victory] slot=%s -> skip SetWindow (no clipping)\n", slot)
	}
	// Position
	a.SetPos(x, y)
	// Scale: include character portraitscale and coord conversion similar to hiscore
	sx := scale[0] * sc.portraitscale * float32(sys.motif.Info.Localcoord[0]) / sc.localcoord[0]
	sy := scale[1] * sc.portraitscale * float32(sys.motif.Info.Localcoord[0]) / sc.localcoord[0]
	a.SetScale(sx, sy)
	if sx == 0 || sy == 0 {
		//fmt.Printf("[Victory] slot=%s -> WARNING: zero scale sx=%.4f sy=%.4f (check portraitscale/localcoord)\n", slot, sx, sy)
	}
	// Palette for non-loaded (or force-apply if requested)
	if applyPal && pal > 0 && a.anim.sff != nil {
		if len(a.anim.sff.palList.paletteMap) > 0 {
			a.anim.sff.palList.paletteMap[0] = pal - 1
		}
		//fmt.Printf("[Victory] slot=%s -> applied palette %d\n", slot, pal)
	}
	return a
}

// applyEntry fills one PlayerVictoryProperties slot from a victoryEntry.
func (vi *MotifVictory) applyEntry(m *Motif, dst *PlayerVictoryProperties, e victoryEntry, slotName string) {
	// Name
	if e.c != nil {
		dst.Name.TextSpriteData.text = e.c.gi().displayname
	} else {
		sc := sys.sel.GetChar(e.cn)
		if sc != nil {
			name := sc.lifebarname
			if name == "" {
				name = sc.name
			}
			dst.Name.TextSpriteData.text = name
		}
	}
	//fmt.Printf("[Victory] applyEntry slot=%s side=%d memberNo=%d cn=%d pal=%d loaded=%v name=%q\n", slotName, e.side, e.memberNo, e.cn, e.pal, e.c != nil, dst.Name.TextSpriteData.text)
	// Resolve SelectChar (for portraits)
	sc := sys.sel.GetChar(e.cn)
	// Main face
	mainX := dst.Pos[0] + dst.Offset[0]
	mainY := dst.Pos[1] + dst.Offset[1]
	dst.AnimData = victoryPortraitAnim(
		m, sc, slotName+".main",
		dst.Anim, dst.Spr,
		dst.Localcoord, dst.Layerno, dst.Facing,
		dst.Scale, dst.Window,
		mainX, mainY,
		dst.ApplyPal || e.c == nil, // loaded chars already have their runtime pal; for un-loaded we must apply
		e.pal, e.c,
	)
	// Face2
	face2X := dst.Pos[0] + dst.Face2.Offset[0]
	face2Y := dst.Pos[1] + dst.Face2.Offset[1]
	dst.Face2.AnimData = victoryPortraitAnim(
		m, sc, slotName+".face2",
		dst.Face2.Anim, dst.Face2.Spr,
		dst.Face2.Localcoord, dst.Face2.Layerno, dst.Face2.Facing,
		dst.Face2.Scale, dst.Face2.Window,
		face2X, face2Y,
		dst.Face2.ApplyPal || e.c == nil,
		e.pal, e.c,
	)
	if dst.AnimData == nil && dst.Face2.AnimData == nil {
		//fmt.Printf("[Victory] slot=%s -> WARNING: both main and face2 animations are nil\n", slotName)
	}
}

func (vi *MotifVictory) init(m *Motif) {
	if !m.VictoryScreen.Enabled || !vi.enabled || sys.winnerTeam() < 1 || (sys.winnerTeam() == 2 && !m.VictoryScreen.Cpu.Enabled) {
		vi.initialized = true
		return
	}

	//fmt.Printf("[Victory] init: enabled=%v winnerTeam=%d cpu.enabled=%v p1.num=%d p2.num=%d\n", m.VictoryScreen.Enabled, sys.winnerTeam(), m.VictoryScreen.Cpu.Enabled, m.VictoryScreen.P1.Num, m.VictoryScreen.P2.Num)

	// Build orders for both sides
	winnerSide := int(sys.winnerTeam() - 1)
	loserSide := winnerSide ^ 1
	// How many portraits per side (respect motif p1_num / p2_num)
	maxW := int(Clamp(m.VictoryScreen.P1.Num, 0, 4))
	maxL := int(Clamp(m.VictoryScreen.P2.Num, 0, 4))
	wEntries := vi.buildSideOrder(winnerSide, m.VictoryScreen.Winner.TeamKo.Enabled, maxW)
	lEntries := vi.buildSideOrder(loserSide, true, maxL) // losers always allow KO display

	// Apply to motif slots: winners -> P1,P3,P5,P7 ; losers -> P2,P4,P6,P8
	wSlots := []*PlayerVictoryProperties{&m.VictoryScreen.P1, &m.VictoryScreen.P3, &m.VictoryScreen.P5, &m.VictoryScreen.P7}
	lSlots := []*PlayerVictoryProperties{&m.VictoryScreen.P2, &m.VictoryScreen.P4, &m.VictoryScreen.P6, &m.VictoryScreen.P8}
	wNames := []string{"P1", "P3", "P5", "P7"}
	lNames := []string{"P2", "P4", "P6", "P8"}
	for i := 0; i < len(wEntries) && i < len(wSlots); i++ {
		vi.applyEntry(m, wSlots[i], wEntries[i], wNames[i])
	}
	for i := 0; i < len(lEntries) && i < len(lSlots); i++ {
		vi.applyEntry(m, lSlots[i], lEntries[i], lNames[i])
	}

	vi.text = vi.getVictoryQuote(m)
	m.VictoryBgDef.BGDef.Reset()

	//fmt.Printf("[Victory] init done. Winners=%d entries, Losers=%d entries. WinQuote=%q\n", len(wEntries), len(lEntries), vi.text)

	if sys.winnerTeam() == 1 {
		m.processStateTransitions(m.VictoryScreen.P1.State, m.VictoryScreen.P1.Teammate.State, m.VictoryScreen.P2.State, m.VictoryScreen.P2.Teammate.State)
	} else if sys.winnerTeam() == 2 {
		m.processStateTransitions(m.VictoryScreen.P2.State, m.VictoryScreen.P2.Teammate.State, m.VictoryScreen.P1.State, m.VictoryScreen.P1.Teammate.State)
	}

	if m.VictoryScreen.Sounds.Enabled {
		sys.clearAllSound()
		sys.noSoundFlg = true
	}

	m.Music.Play("victory", sys.motif.Def, false)

	m.VictoryScreen.FadeIn.FadeData.init(m.fadeIn, true)
	vi.counter = 0
	vi.active = true
	vi.initialized = true
}

func (vi *MotifVictory) step(m *Motif) {
	cancelPressed := m.button(m.VictoryScreen.Cancel.Key, -1)
	skipPressed := m.button(m.VictoryScreen.Skip.Key, -1)
	prevLineFullyRendered := vi.lineFullyRendered
	//fmt.Printf("[Victory] step: counter=%d time=%d endTimer=%d typedCnt=%d lineFullyRendered=%v cancel=%v skip=%v\n", vi.counter, m.VictoryScreen.Time, vi.endTimer, vi.typedCnt, vi.lineFullyRendered, cancelPressed, skipPressed)

	m.VictoryScreen.P1.AnimData.Update()
	m.VictoryScreen.P2.AnimData.Update()
	m.VictoryScreen.P3.AnimData.Update()
	m.VictoryScreen.P4.AnimData.Update()
	m.VictoryScreen.P5.AnimData.Update()
	m.VictoryScreen.P6.AnimData.Update()
	m.VictoryScreen.P7.AnimData.Update()
	m.VictoryScreen.P8.AnimData.Update()

	m.VictoryScreen.P1.Face2.AnimData.Update()
	m.VictoryScreen.P2.Face2.AnimData.Update()
	m.VictoryScreen.P3.Face2.AnimData.Update()
	m.VictoryScreen.P4.Face2.AnimData.Update()
	m.VictoryScreen.P5.Face2.AnimData.Update()
	m.VictoryScreen.P6.Face2.AnimData.Update()
	m.VictoryScreen.P7.Face2.AnimData.Update()
	m.VictoryScreen.P8.Face2.AnimData.Update()

	// First press of Skip: fast-forward the text, but do NOT start fadeout yet.
	if skipPressed && !prevLineFullyRendered {
		totalRunes := utf8.RuneCountInString(vi.text)
		vi.typedCnt = totalRunes
		vi.lineFullyRendered = true
		vi.charDelayCounter = 0
		//fmt.Printf("[Victory] Skip pressed -> fast-forward winquote (totalRunes=%d)\n", totalRunes)
	}

	// While we haven't finished typing the quote, keep revealing characters
	// regardless of the global time limit. Fadeout will only start once the
	// line is fully rendered (see logic below).
	if !vi.lineFullyRendered {
		StepTypewriter(
			vi.text,
			&vi.typedCnt,
			&vi.charDelayCounter,
			&vi.lineFullyRendered,
			float32(m.VictoryScreen.WinQuote.TextDelay),
		)
	}

	// Clamp typedLen so it doesn't exceed the line length
	totalRunes := utf8.RuneCountInString(vi.text)
	typedLen := vi.typedCnt
	if typedLen > totalRunes {
		typedLen = totalRunes
	}

	m.VictoryScreen.WinQuote.TextSpriteData.wrapText(vi.text, typedLen)
	m.VictoryScreen.WinQuote.TextSpriteData.Update()

	// Decide when to start fadeout: Cancel key / Skip key / Time limit
	if vi.endTimer == -1 {
		userInterrupt := cancelPressed || (skipPressed && prevLineFullyRendered)
		timeUp := vi.lineFullyRendered && vi.counter >= m.VictoryScreen.Time

		if userInterrupt || timeUp {
			startFadeOut(m.VictoryScreen.FadeOut.FadeData, m.fadeOut, userInterrupt, m.fadePolicy)
			vi.endTimer = vi.counter + m.fadeOut.timeRemaining
			//fmt.Printf("[Victory] Starting fadeout: counter=%d time=%d endTimer=%d userInterrupt=%v timeUp=%v\n", vi.counter, m.VictoryScreen.Time, vi.endTimer, userInterrupt, timeUp)
		}
	}

	// Check if the sequence has ended
	if vi.endTimer != -1 && vi.counter >= vi.endTimer {
		if m.fadeOut != nil {
			m.fadeOut.reset()
		}
		vi.active = false
		if !m.VictoryScreen.Sounds.Enabled {
			sys.noSoundFlg = false
		}
		return
	}

	// Increment counter
	vi.counter++
}

func (vi *MotifVictory) draw(m *Motif, layerno int16) {
	// Overlay
	m.VictoryScreen.Overlay.RectData.Draw(layerno)

	// Background
	if m.VictoryBgDef.BgClearColor[0] >= 0 {
		m.VictoryBgDef.RectData.Draw(layerno)
	}
	m.VictoryBgDef.BGDef.Draw(int32(layerno), 0, 0, 1)

	// Face2 portraits
	m.VictoryScreen.P1.Face2.AnimData.Draw(layerno)
	m.VictoryScreen.P2.Face2.AnimData.Draw(layerno)
	m.VictoryScreen.P3.Face2.AnimData.Draw(layerno)
	m.VictoryScreen.P4.Face2.AnimData.Draw(layerno)
	m.VictoryScreen.P5.Face2.AnimData.Draw(layerno)
	m.VictoryScreen.P6.Face2.AnimData.Draw(layerno)
	m.VictoryScreen.P7.Face2.AnimData.Draw(layerno)
	m.VictoryScreen.P8.Face2.AnimData.Draw(layerno)

	// Face portraits
	m.VictoryScreen.P1.AnimData.Draw(layerno)
	m.VictoryScreen.P2.AnimData.Draw(layerno)
	m.VictoryScreen.P3.AnimData.Draw(layerno)
	m.VictoryScreen.P4.AnimData.Draw(layerno)
	m.VictoryScreen.P5.AnimData.Draw(layerno)
	m.VictoryScreen.P6.AnimData.Draw(layerno)
	m.VictoryScreen.P7.AnimData.Draw(layerno)
	m.VictoryScreen.P8.AnimData.Draw(layerno)

	// Name
	m.VictoryScreen.P1.Name.TextSpriteData.Draw(layerno)
	m.VictoryScreen.P2.Name.TextSpriteData.Draw(layerno)
	m.VictoryScreen.P3.Name.TextSpriteData.Draw(layerno)
	m.VictoryScreen.P4.Name.TextSpriteData.Draw(layerno)
	m.VictoryScreen.P5.Name.TextSpriteData.Draw(layerno)
	m.VictoryScreen.P6.Name.TextSpriteData.Draw(layerno)
	m.VictoryScreen.P7.Name.TextSpriteData.Draw(layerno)
	m.VictoryScreen.P8.Name.TextSpriteData.Draw(layerno)

	// Winquote
	m.VictoryScreen.WinQuote.TextSpriteData.Draw(layerno)
}
