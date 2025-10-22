// ----------------------------------------------------------------------------
// Music system – overview
// ----------------------------------------------------------------------------
//
// Goal
// -----
// Unify every place that can set BGM (screenpack/motif, stage, storyboard,
// select.def, and Lua launch parameters) into one runtime type.
//
// Core type
// ---------
// Music is a map[string][]*bgMusic keyed by a *prefix* (e.g. "", "title",
// "round1", "life", "victory", "final", "scene1", ...). Each key holds a list
// of candidate tracks with playback options. Read() randomly picks one entry
// for the prefix and returns the resolved file and options. Play() delegates
// to Read() and opens the BGM.
//
// Where values come from (all normalized into Music struct)
// --------------------------------------------------
// • Motif / Screenpack (system.def [Music])
//   Parsed by iniutils.parseMusicSection() during loadMotif().
//   Stored in sys.motif.Music.
//
// • Stage (stage.def [Music])
//   Parsed by parseMusicSection() into the stage definition.
//   Stored in sys.stage.music.
//
// • Storyboard (per-scene bgm, declared under [Scene X])
//   Parsed by parseMusicSection() into prefixes like "scene1", ...
//   Stored in sys.storyboard.Scene[X].Music
//
// • select.def – character parameters (Lua addChar)
//   script.go passes raw "k=v" params via AppendParams(...).
//   Stored in the sys.sel.charlist[X].music
//
//
// • select.def – stage parameters (Lua addStage):
//   script.go passes raw "k=v" params via AppendParams(...).
//   Stored in the sys.sel.stagelist[X].music
//
// • launchFight (Lua loadStart):
//   script.go passes raw "k=v" params via AppendParams(...).
//   Stored in sys.sel.music
//
// Normalization of naming
// -----------------------
// Elecbyte used inconsistent keys across files. Both the INI loader
// (parseMusicSection) and the runtime parameter path (AppendParams) accept
// synonyms and normalize them into bgMusic fields:
//
//   bgm / bgmusic / music
//   bgm.loop / bgmloop
//   bgm.volume / bgmvolume
//   bgm.loopstart / bgmloopstart
//   bgm.loopend / bgmloopend
//   bgm.startposition / bgmstartposition
//   bgm.freqmul / bgmfreqmul
//   bgm.loopcount / bgmloopcount
//
// Prefix handling
// ---------------
// Keys may be written as "<prefix>.<field>" (e.g. "round2.bgmusic").
// Everything before the first dot is the Music map key; the rest selects a
// bgMusic field. When no prefix is present the empty key "" is used.
//
// Resolution order (per character)
// --------------------------------------------
// The final playable list is computed and stored per character slot so that
// per-character or per-stage overrides can win over defaults; custom character
// music (or stage music coming from select.def) can override the stage's .def
// music, and launch-time Lua must be able to override both.

// In resetRoundState the code does (for each sys.chars index i):
// • 1. Clear
//   sys.cgi[i].music = make(Music)
// • 2. Base: stage.def [Music]
//   sys.cgi[i].music.Append(sys.stage.music)
// • 3. select.def stage params (addStage)
//   sys.cgi[i].music.Append(sys.stage.si().music)
// • 4. select.def character params (addChar) – override stage
//   sys.cgi[i].music.Override(p[0].si().music)
// • 5. launchFight params (loadStart) – last word
//   sys.cgi[i].music.Override(sys.sel.music)
//
// Runtime selection
// -----------------
// During a match, Music.act() reacts to state and tries to play a suitable
// prefix in this order:
// • round start: "final" (if decisive) or "round<sys.round>"
// • low life (team leader): "life"
// • victory (decisive, winner alive): "victory"
// tryPlay() checks if a prefix exists and has any defined track, then calls
// Play. Read() randomizes among multiple entries for the prefix and resolves
// the path using SearchFile.
//
// ----------------------------------------------------------------------------

package main

import (
	"fmt"
	"strings"
)

type MusicSource int32

const (
	MS_Match        MusicSource = iota // final, per-character, merged list (sys.cgi[i].music)
	MS_StageDef                        // stage.def [Music]
	MS_CharParams                      // select.def character params (Lua addChar)
	MS_StageParams                     // select.def stage params (Lua addStage)
	MS_LaunchParams                    // launchFight params (Lua loadStart)
	MS_Motif                           // system.def [Music] in the motif/screenpack
)

type bgMusic struct {
	bgmusic          string
	bgmloop          int32
	bgmvolume        int32
	bgmloopstart     int32
	bgmloopend       int32
	bgmstartposition int32
	bgmfreqmul       float32
	bgmloopcount     int32
}

func newBgMusic() *bgMusic {
	return &bgMusic{bgmloop: 1, bgmvolume: 100, bgmfreqmul: 1, bgmloopcount: -1}
}

// Music is the normalized store for all music data.
// Key: prefix (e.g. "round1", "life", "victory", "title", etc.)
// Value: ordered slice of candidates; Read() picks one at random.
type Music map[string][]*bgMusic

// Append merges another Music by concatenating candidate lists per prefix.
// Use when adding sources of equal or lower priority (e.g. stage.def base
// then select.def stage params).
func (m Music) Append(other Music) {
	for key, otherList := range other {
		m[key] = append(m[key], otherList...)
	}
}

// Override applies element-wise replacement per prefix (by index). If the
// overriding list is longer, it extends the target. Use when raising
// priority (e.g. char select params over stage lists, or launchFight over all).
func (m Music) Override(other Music) {
	for key, otherList := range other {
		if mList, exists := m[key]; exists {
			for i, otherBg := range otherList {
				if i < len(mList) {
					mList[i] = otherBg
				} else {
					mList = append(mList, otherBg)
				}
			}
			m[key] = mList
		} else {
			m[key] = otherList
		}
	}
}

// AppendParams parses comma-separated "key=value" pairs (as passed from
// Lua addChar/addStage/loadStart) and appends to the proper prefix list.
func (m Music) AppendParams(entries []string) {
	for _, c := range entries {
		if eqPos := strings.Index(c, "="); eqPos != -1 {
			key := strings.TrimSpace(c[:eqPos])
			value := strings.TrimSpace(c[eqPos+1:])
			prefix := ""
			field := key
			if dotPos := strings.Index(key, "."); dotPos != -1 {
				prefix = key[:dotPos]
				field = key[dotPos+1:]
			}
			if !strings.HasPrefix(field, "bgm") && field != "music" {
				continue
			}
			if len(m[prefix]) == 0 || field == "bgmusic" || field == "music" {
				m[prefix] = append(m[prefix], newBgMusic())
			}
			idx := len(m[prefix]) - 1
			switch field {
			case "bgmusic", "music":
				m[prefix][idx].bgmusic = value
			case "bgmloop":
				m[prefix][idx].bgmloop = Atoi(value)
			case "bgmvolume":
				m[prefix][idx].bgmvolume = Atoi(value)
			case "bgmloopstart":
				m[prefix][idx].bgmloopstart = Atoi(value)
			case "bgmloopend":
				m[prefix][idx].bgmloopend = Atoi(value)
			case "bgmstartposition":
				m[prefix][idx].bgmstartposition = Atoi(value)
			case "bgmfreqmul":
				m[prefix][idx].bgmfreqmul = float32(Atof(value))
			case "bgmloopcount":
				m[prefix][idx].bgmloopcount = Atoi(value)
			}
		}
	}
}

// Read resolves a concrete file and playback params for a key like
// "round1.bgmusic" by using its prefix ("round1") and picking one random
// candidate from the list.
func (m Music) Read(key, def string) (string, int, int, int, int, int, float32, int) {
	var bgm string
	var loop, volume, loopstart, loopend, startposition, loopcount int = 1, 100, 0, 0, 0, -1
	var freqmul float32 = 1.0
	prefix := ""
	if dotPos := strings.Index(key, "."); dotPos != -1 {
		prefix = key[:dotPos]
	}
	if len(m[prefix]) > 0 {
		idx := int(RandI(0, int32(len(m[prefix]))-1))
		bgm = SearchFile(m[prefix][idx].bgmusic, []string{def, "", "sound/"})
		loop = int(m[prefix][idx].bgmloop)
		volume = int(m[prefix][idx].bgmvolume)
		loopstart = int(m[prefix][idx].bgmloopstart)
		loopend = int(m[prefix][idx].bgmloopend)
		startposition = int(m[prefix][idx].bgmstartposition)
		freqmul = m[prefix][idx].bgmfreqmul
		loopcount = int(m[prefix][idx].bgmloopcount)
	}
	return bgm, loop, volume, loopstart, loopend, startposition, freqmul, loopcount
}

// Play opens the chosen track in the global BGM player. If force is true,
// it will open even when Read() finds nothing (used to explicitly stop/clear).
func (m Music) Play(key, path string, force bool) bool {
	if track, loop, volume, loopstart, loopend, startposition, freqmul, loopcount := m.Read(key, path); track != "" || force {
		sys.bgm.Open(track, loop, volume, loopstart, loopend, startposition, freqmul, loopcount)
		sys.playBgmFlg = true
		return true
	}
	return false
}

type BGMState byte

const (
	BGMStateIdle    BGMState = iota // no track chosen yet
	BGMStateRound                   // round start track chosen/playing
	BGMStateLowLife                 // low-life track chosen/playing
	BGMStateVictory                 // victory track chosen/playing
)

// tryPlay is a guarded helper used by act(): if a prefix exists and has at
// least one non-empty entry, it delegates to Play("<prefix>.bgmusic", def).
func (m Music) tryPlay(key, def string) bool {
	if dot := strings.Index(key, "."); dot != -1 {
		key = key[:dot]
	}
	lst, ok := m[key]
	if !ok || len(lst) == 0 {
		return false
	}
	hasDefined := false
	for _, v := range lst {
		if v != nil && strings.TrimSpace(v.bgmusic) != "" {
			hasDefined = true
			break
		}
	}
	if !hasDefined {
		return false
	}
	return m.Play(key+".bgmusic", def, false)
}

// act drives in-fight music state transitions:
//   - At round start: final.bgmusic (if decisive) else round{N}.bgmusic
//   - On leader low life: life.bgmusic
//   - On decisive victory: victory.bgmusic
func (m Music) act() {
	if sys.gameMode == "demo" && !sys.motif.DemoMode.Fight.PlayBgm {
		return
	}
	// Iterate players in order: P2, P2 teammates, then P1, P1 teammates.
	// Skips empty slots and ignores attached chars.
	for side := 1; side >= 0; side-- { // 1 = P2 side first, then 0 = P1 side
		for pn := side; pn < int(MaxSimul)*2; pn += 2 {
			if len(sys.chars[pn]) == 0 || sys.chars[pn][0] == nil {
				continue
			}
			c := sys.chars[pn][0] // root player in this slot

			// Round Start
			if c.teamside == sys.home &&
				sys.stage.bgmState == BGMStateIdle &&
				sys.tickCount == 0 &&
				!sys.roundResetFlg {

				roundKey := fmt.Sprintf("round%d.bgmusic", sys.round)
				switch {
				case sys.round > 1 && sys.decisiveRound[0] && sys.decisiveRound[1] && m.tryPlay("final.bgmusic", sys.stage.def):
				case m.tryPlay(roundKey, sys.stage.def):
				}
				sys.stage.bgmState = BGMStateRound
				continue
			}

			// Low Life (only team leader)
			if sys.stage.bgmState == BGMStateRound &&
				sys.roundState() == 2 &&
				c.playerNo == c.teamLeader() &&
				float32(c.life)/float32(c.lifeMax) <= 0.3 {

				if m.tryPlay("life.bgmusic", sys.stage.def) {
					sys.stage.bgmState = BGMStateLowLife
					continue
				}
			}

			// Victory (decisive round, winning & alive)
			if sys.stage.bgmState < BGMStateVictory &&
				c.win() && c.alive() &&
				sys.decisiveRound[c.teamside] {

				if m.tryPlay("victory.bgmusic", sys.stage.def) {
					sys.stage.bgmState = BGMStateVictory
					continue
				}
			}
		}
	}
}
