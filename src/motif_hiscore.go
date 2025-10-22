package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type MotifHiscore struct {
	enabled     bool
	active      bool
	initialized bool
	counter     int32
	endTimer    int32
	place       int32
	mode        string
	rows        []rankingRow
	statsRaw    string

	// Active-row blink state (Rank / Result / Name)
	rankActiveCount   int32
	resultActiveCount int32
	nameActiveCount   int32
	rankUseActive2    bool
	resultUseActive2  bool
	nameUseActive2    bool

	// Name entry & timer
	input       bool  // true while entering name
	letters     []int // 1-based indices into glyph list
	timerCount  int32 // remaining displayed timer count
	timerFrames int32 // frames left until next decrement
	haveSaved   bool  // true once we've persisted the entered name
}

// rankingRow is a minimal container built from dynamic JSON (no static decode).
type rankingRow struct {
	score int
	time  float64
	win   int
	name  string
	pals  []int32
	chars []string
	bgs   []*Anim
	faces []*Anim

	rankData          *TextSprite
	resultData        *TextSprite
	nameData          *TextSprite
	rankDataActive    *TextSprite
	rankDataActive2   *TextSprite
	resultDataActive  *TextSprite
	resultDataActive2 *TextSprite
	nameDataActive    *TextSprite
	nameDataActive2   *TextSprite
}

func (hi *MotifHiscore) reset(m *Motif) {
	hi.active = false
	hi.initialized = false
	hi.endTimer = -1
	hi.counter = 0
	hi.place = 0
	hi.mode = ""
	hi.rankActiveCount, hi.resultActiveCount, hi.nameActiveCount = 0, 0, 0
	hi.rankUseActive2, hi.resultUseActive2, hi.nameUseActive2 = false, false, false
	hi.rows = nil
	hi.input = false
	hi.letters = nil
	hi.timerCount = 0
	hi.timerFrames = 0
	hi.haveSaved = false
}

func (hi *MotifHiscore) init(m *Motif, mode string, place int32) {
	//if !m.HiscoreInfo.Enabled || !hi.enabled {
	//	hi.initialized = true
	//	return
	//}

	dataType, _ := m.HiscoreInfo.Ranking[mode]

	hi.reset(m)
	hi.place = place
	hi.mode = mode
	hi.input = (place > 0)
	if hi.input {
		// Start with one letter selected
		hi.letters = []int{1}
	}

	// Timer setup (only matters during input)
	hi.timerCount = m.HiscoreInfo.Timer.Count
	hi.timerFrames = m.HiscoreInfo.Timer.Framespercount

	// Parse and cache rows (read from JSON file)
	hi.rows = parseRankingRows("save/stats.json", mode)

	// Build portraits cache from cached rows (store on each row)
	visible := int(m.HiscoreInfo.Window.VisibleItems)
	if visible <= 0 || visible > len(hi.rows) {
		visible = len(hi.rows)
	}

	m.HiscoreInfo.Item.Face.AnimData.Reset()
	m.HiscoreInfo.Item.Face.Bg.AnimData.Reset()
	m.HiscoreInfo.Item.Face.Unknown.AnimData.Reset()
	baseX, baseY := m.HiscoreInfo.Pos[0], m.HiscoreInfo.Pos[1]
	itemOffX, itemOffY := m.HiscoreInfo.Item.Offset[0], m.HiscoreInfo.Item.Offset[1]
	for i := 0; i < visible; i++ {
		rowDefs := hi.rows[i].chars
		rowBgs := make([]*Anim, 0, len(rowDefs))
		rowFaces := make([]*Anim, 0, len(rowDefs))
		limit := int(m.HiscoreInfo.Item.Face.Num)
		if limit <= 0 {
			limit = math.MaxInt32
		}
		for j, def := range rowDefs {
			if j >= limit {
				break
			}
			// Compute the on-screen position for this face/background once.
			x := baseX + itemOffX + m.HiscoreInfo.Item.Face.Offset[0] +
				float32(i)*m.HiscoreInfo.Item.Spacing[0] +
				float32(j)*m.HiscoreInfo.Item.Face.Spacing[0]
			y := baseY + itemOffY + m.HiscoreInfo.Item.Face.Offset[1] +
				float32(i)*(m.HiscoreInfo.Item.Spacing[1]+m.HiscoreInfo.Item.Face.Spacing[1])

			// Background anim per face (clone so each slot has its own pos/state)
			if m.HiscoreInfo.Item.Face.Bg.AnimData != nil {
				bg := m.HiscoreInfo.Item.Face.Bg.AnimData.Copy()
				if bg != nil {
					bg.SetPos(x+m.HiscoreInfo.Item.Face.Bg.Offset[0], y+m.HiscoreInfo.Item.Face.Bg.Offset[1])
				}
				rowBgs = append(rowBgs, bg)
			} else {
				rowBgs = append(rowBgs, nil)
			}

			// Face anim (character or unknown)
			if a := hiscorePortraitAnim(def, m, x, y); a != nil {
				rowFaces = append(rowFaces, a)
			} else if m.HiscoreInfo.Item.Face.Unknown.AnimData != nil {
				// Use a cloned "unknown" placeholder per slot so we can position/update independently.
				ua := m.HiscoreInfo.Item.Face.Unknown.AnimData.Copy()
				if ua != nil {
					ua.SetPos(x+m.HiscoreInfo.Item.Face.Unknown.Offset[0], y+m.HiscoreInfo.Item.Face.Unknown.Offset[1])
				}
				rowFaces = append(rowFaces, ua)
			} else {
				rowFaces = append(rowFaces, nil)
			}

			//
		}
		hi.rows[i].bgs = rowBgs
		hi.rows[i].faces = rowFaces
	}

	// --------- Build per-row TextSprites (Rank / Result / Name) ----------
	for i := 0; i < visible; i++ {
		row := &hi.rows[i]
		// Rank
		if m.HiscoreInfo.Item.Rank.TextSpriteData != nil {
			ts := m.HiscoreInfo.Item.Rank.TextSpriteData.Copy()
			if ts != nil {
				x := baseX + itemOffX + m.HiscoreInfo.Item.Rank.Offset[0] +
					float32(i)*(m.HiscoreInfo.Item.Spacing[0]+m.HiscoreInfo.Item.Rank.Spacing[0])
				stepY := float32(math.Round(float64(
					(float32(ts.fnt.Size[1])+float32(ts.fnt.Spacing[1]))*ts.yscl +
						(m.HiscoreInfo.Item.Spacing[1] + m.HiscoreInfo.Item.Rank.Spacing[1]),
				)))
				y := baseY + itemOffY + m.HiscoreInfo.Item.Rank.Offset[1] + stepY*float32(i)
				ts.SetPos(x, y)
				rankKey := Itoa(i + 1)
				fmtStr, ok := m.HiscoreInfo.Item.Rank.Text[rankKey]
				if !ok || fmtStr == "" {
					fmtStr = m.HiscoreInfo.Item.Rank.Text["default"]
				}
				fmtStr = m.replaceFormatSpecifiers(fmtStr)
				ts.text = fmt.Sprintf(fmtStr, i+1)
				if m.HiscoreInfo.Item.Rank.Uppercase {
					ts.text = strings.ToUpper(ts.text)
				}
			}
			row.rankData = ts
		}
		// If this is the highlighted row, prepare Active/Active2 clones (same pos/text)
		if hi.place > 0 && int(hi.place-1) == i {
			if row.rankData != nil {
				row.rankDataActive = cloneWithFont(row.rankData, m.HiscoreInfo.Item.Rank.Active.Font, m.Fnt)
				row.rankDataActive2 = cloneWithFont(row.rankData, m.HiscoreInfo.Item.Rank.Active2.Font, m.Fnt)
			}
		}

		// Result
		if m.HiscoreInfo.Item.Result.TextSpriteData != nil {
			ts := m.HiscoreInfo.Item.Result.TextSpriteData.Copy()
			if ts != nil {
				x := baseX + itemOffX + m.HiscoreInfo.Item.Result.Offset[0] +
					float32(i)*(m.HiscoreInfo.Item.Spacing[0]+m.HiscoreInfo.Item.Result.Spacing[0])
				stepY := float32(math.Round(float64(
					(float32(ts.fnt.Size[1])+float32(ts.fnt.Spacing[1]))*ts.yscl +
						(m.HiscoreInfo.Item.Spacing[1] + m.HiscoreInfo.Item.Result.Spacing[1]),
				)))
				y := baseY + itemOffY + m.HiscoreInfo.Item.Result.Offset[1] + stepY*float32(i)
				ts.SetPos(x, y)

				fmtStr := m.HiscoreInfo.Item.Result.Text[dataType]
				fmtStr = m.replaceFormatSpecifiers(fmtStr)
				switch dataType {
				case "score":
					// e.g. "%08d" -> zero-padded to width 8
					fmtStr = fmt.Sprintf(fmtStr, row.score)
				case "win":
					// e.g. "Round %d"
					fmtStr = fmt.Sprintf(fmtStr, row.win)
				case "time":
					// e.g. "%m'%s''%x"
					fmtStr = FormatTimeText(fmtStr, row.time)
				}
				if m.HiscoreInfo.Item.Result.Uppercase {
					fmtStr = strings.ToUpper(fmtStr)
				}
				ts.text = fmtStr
			}
			row.resultData = ts
		}
		if hi.place > 0 && int(hi.place-1) == i {
			if row.resultData != nil {
				row.resultDataActive = cloneWithFont(row.resultData, m.HiscoreInfo.Item.Result.Active.Font, m.Fnt)
				row.resultDataActive2 = cloneWithFont(row.resultData, m.HiscoreInfo.Item.Result.Active2.Font, m.Fnt)
			}
		}

		// Name
		if m.HiscoreInfo.Item.Name.TextSpriteData != nil {
			ts := m.HiscoreInfo.Item.Name.TextSpriteData.Copy()
			if ts != nil {
				x := baseX + itemOffX + m.HiscoreInfo.Item.Name.Offset[0] +
					float32(i)*(m.HiscoreInfo.Item.Spacing[0]+m.HiscoreInfo.Item.Name.Spacing[0])
				stepY := float32(math.Round(float64(
					(float32(ts.fnt.Size[1])+float32(ts.fnt.Spacing[1]))*ts.yscl +
						(m.HiscoreInfo.Item.Spacing[1] + m.HiscoreInfo.Item.Name.Spacing[1]),
				)))
				y := baseY + itemOffY + m.HiscoreInfo.Item.Name.Offset[1] + stepY*float32(i)
				ts.SetPos(x, y)
				// If highlighted & input, start with glyph-based editable text; else use row.name from stats
				if hi.input && int(hi.place-1) == i {
					row.name = "" // TODO: this shouldn't be needed if we're appending new row
					name := buildNameFromLetters(m, hi.letters)
					fs := m.replaceFormatSpecifiers(m.HiscoreInfo.Item.Name.Text["default"])
					ts.text = fmt.Sprintf(fs, name)
				} else {
					fs := m.replaceFormatSpecifiers(m.HiscoreInfo.Item.Name.Text["default"])
					ts.text = fmt.Sprintf(fs, row.name)
				}
				if m.HiscoreInfo.Item.Name.Uppercase {
					ts.text = strings.ToUpper(ts.text)
				}
			}
			row.nameData = ts
		}
		if hi.place > 0 && int(hi.place-1) == i {
			if row.nameData != nil {
				row.nameDataActive = cloneWithFont(row.nameData, m.HiscoreInfo.Item.Name.Active.Font, m.Fnt)
				row.nameDataActive2 = cloneWithFont(row.nameData, m.HiscoreInfo.Item.Name.Active2.Font, m.Fnt)
			}
		}
	}

	// Initialize timer TextSprite (position & first value)
	if m.HiscoreInfo.Timer.TextSpriteData != nil {
		m.HiscoreInfo.Timer.TextSpriteData.Reset()
		m.HiscoreInfo.Timer.TextSpriteData.AddPos(m.HiscoreInfo.Pos[0], m.HiscoreInfo.Pos[1])
		ts := m.HiscoreInfo.Timer.Text
		ts = m.replaceFormatSpecifiers(ts)
		if hi.input && hi.timerCount >= 0 && m.HiscoreInfo.Timer.Count != -1 {
			m.HiscoreInfo.Timer.TextSpriteData.text = fmt.Sprintf(ts, hi.timerCount)
		} else {
			// Leave text as-is if timer disabled
			m.HiscoreInfo.Timer.TextSpriteData.text = fmt.Sprintf(ts, 0)
		}
	}

	m.HiscoreBgDef.BGDef.Reset()
	m.HiscoreInfo.FadeIn.FadeData.init(m.fadeIn, true)

	m.HiscoreInfo.Title.TextSpriteData.Reset()
	m.HiscoreInfo.Title.TextSpriteData.AddPos(m.HiscoreInfo.Pos[0], m.HiscoreInfo.Pos[1])
	if v, ok := m.HiscoreInfo.Title.Text[mode]; ok {
		m.HiscoreInfo.Title.TextSpriteData.text = v
	}

	m.HiscoreInfo.Title.Rank.TextSpriteData.Reset()
	m.HiscoreInfo.Title.Rank.TextSpriteData.AddPos(m.HiscoreInfo.Pos[0], m.HiscoreInfo.Pos[1])

	m.HiscoreInfo.Title.Result.TextSpriteData.Reset()
	m.HiscoreInfo.Title.Result.TextSpriteData.AddPos(m.HiscoreInfo.Pos[0], m.HiscoreInfo.Pos[1])

	m.HiscoreInfo.Title.Name.TextSpriteData.Reset()
	m.HiscoreInfo.Title.Name.TextSpriteData.AddPos(m.HiscoreInfo.Pos[0], m.HiscoreInfo.Pos[1])

	m.HiscoreInfo.Title.Face.TextSpriteData.Reset()
	m.HiscoreInfo.Title.Face.TextSpriteData.AddPos(m.HiscoreInfo.Pos[0], m.HiscoreInfo.Pos[1])

	m.Music.Play("hiscore", sys.motif.Def, false)

	//hi.counter = 0
	hi.active = true
	hi.initialized = true
}

func (hi *MotifHiscore) step(m *Motif) {
	// Begin fade-out on cancel or when time elapses.
	if hi.endTimer == -1 {
		cancel := sys.esc || sys.button("m") >= 0 || (!sys.gameRunning && sys.motif.AttractMode.Enabled && sys.credits > 0)
		if cancel || (!hi.input && hi.counter == m.HiscoreInfo.Time) {
			startFadeOut(m.HiscoreInfo.FadeOut.FadeData, m.fadeOut, cancel, m.fadePolicy)
			hi.endTimer = hi.counter + m.fadeOut.timeRemaining
		}
	}

	hi.handleBlinkers(m)

	// Advance animations for per-face backgrounds and faces
	for i := range hi.rows {
		row := &hi.rows[i]
		for _, bg := range row.bgs {
			if bg != nil {
				bg.Update()
			}
		}
		for _, face := range row.faces {
			if face != nil {
				face.Update()
			}
		}
	}

	// ---------------- Name input & timer (only for highlighted row) ----------------
	if hi.place > 0 {
		idx := int(hi.place - 1)
		if idx >= 0 && idx < len(hi.rows) {
			row := &hi.rows[idx]
			// Timer tick — only if enabled and input active
			if hi.input && m.HiscoreInfo.Timer.Count != -1 {
				if hi.timerFrames > 0 {
					hi.timerFrames--
				} else {
					if hi.timerCount > 0 {
						hi.timerCount--
					}
					hi.timerFrames = m.HiscoreInfo.Timer.Framespercount
				}
				// Update timer text each step
				if m.HiscoreInfo.Timer.TextSpriteData != nil {
					ts := m.HiscoreInfo.Timer.Text
					ts = m.replaceFormatSpecifiers(ts)
					m.HiscoreInfo.Timer.TextSpriteData.text = fmt.Sprintf(ts, hi.timerCount)
				}
				// When time runs out, auto-finish input
				if hi.timerCount <= 0 {
					m.Snd.play(m.HiscoreInfo.Done.Snd, 100, 0, 0, 0, 0)
					hi.input = false
					// Give a short tail
					hi.counter = m.HiscoreInfo.Time - m.HiscoreInfo.Done.Time
					hi.finalizeAndSave()
				}
			}

			// Handle name glyph entry while input is active
			if hi.input {
				controller := -1 // TODO: proper controller
				maxLen := initialsWidth(m.HiscoreInfo.Item.Name.Text["default"])
				glyphCount := len(m.HiscoreInfo.Glyphs)
				// Previous glyph
				if m.button(m.HiscoreInfo.Previous.Key, controller) {
					m.Snd.play(m.HiscoreInfo.Move.Snd, 100, 0, 0, 0, 0)
					if len(hi.letters) == 0 {
						hi.letters = []int{1}
					} else {
						last := len(hi.letters) - 1
						hi.letters[last]--
						if hi.letters[last] <= 0 {
							hi.letters[last] = glyphCount
						}
					}
					updateRowNameFromLetters(m, row, hi.letters)
					// Next glyph
				} else if m.button(m.HiscoreInfo.Next.Key, controller) {
					m.Snd.play(m.HiscoreInfo.Move.Snd, 100, 0, 0, 0, 0)
					if len(hi.letters) == 0 {
						hi.letters = []int{1}
					} else {
						last := len(hi.letters) - 1
						hi.letters[last]++
						if hi.letters[last] > glyphCount {
							hi.letters[last] = 1
						}
					}
					updateRowNameFromLetters(m, row, hi.letters)
					// Confirm / Add / Backspace
				} else if m.button(m.HiscoreInfo.Done.Key, controller) {
					// Current glyph meaning
					curGlyph := currentGlyph(m, hi.letters)
					if curGlyph == "<" {
						// Backspace
						m.Snd.play(m.HiscoreInfo.Cancel.Snd, 100, 0, 0, 0, 0)
						if len(hi.letters) > 1 {
							hi.letters = hi.letters[:len(hi.letters)-1]
						} else {
							hi.letters = []int{1}
						}
						updateRowNameFromLetters(m, row, hi.letters)
					} else if len(hi.letters) < maxLen {
						m.Snd.play(m.HiscoreInfo.Done.Snd, 100, 0, 0, 0, 0)
						lastIdx := hi.letters[len(hi.letters)-1]
						hi.letters = append(hi.letters, lastIdx)
						updateRowNameFromLetters(m, row, hi.letters)
					} else {
						// Finalize
						m.Snd.play(m.HiscoreInfo.Done.Snd, 100, 0, 0, 0, 0)
						hi.input = false
						hi.counter = m.HiscoreInfo.Time - m.HiscoreInfo.Done.Time
						hi.finalizeAndSave()
					}
				}
			}
		}
	}

	// Finish after fade-out completes
	if hi.endTimer != -1 && hi.counter >= hi.endTimer {
		if m.fadeOut != nil {
			m.fadeOut.reset()
		}
		hi.reset(m)
		//hi.active = false
		return
	}

	hi.counter++
}

func (hi *MotifHiscore) draw(m *Motif, layerno int16) {
	// Background
	if m.HiscoreBgDef.BgClearColor[0] >= 0 {
		m.HiscoreBgDef.RectData.Draw(layerno)
	}
	m.HiscoreBgDef.BGDef.Draw(int32(layerno), 0, 0, 1)

	// Title and subtitles
	m.HiscoreInfo.Title.TextSpriteData.Draw(layerno)
	m.HiscoreInfo.Title.Rank.TextSpriteData.Draw(layerno)
	m.HiscoreInfo.Title.Result.TextSpriteData.Draw(layerno)
	m.HiscoreInfo.Title.Name.TextSpriteData.Draw(layerno)
	m.HiscoreInfo.Title.Face.TextSpriteData.Draw(layerno)

	for i := 0; i < len(hi.rows); i++ {
		// Portraits bg
		for _, bg := range hi.rows[i].bgs {
			if bg != nil {
				bg.Draw(layerno)
			}
		}
		// Portraits
		for _, a := range hi.rows[i].faces {
			if a == nil {
				continue
			}
			a.Draw(layerno)
		}

		// Text sprites (blink only on highlighted row)
		row := &hi.rows[i]
		if hi.place > 0 && int(hi.place-1) == i {
			// Rank
			if hi.rankUseActive2 {
				row.rankDataActive2.Draw(layerno)
			} else {
				row.rankDataActive.Draw(layerno)
			}
			// Result
			if hi.resultUseActive2 {
				row.resultDataActive2.Draw(layerno)
			} else {
				row.resultDataActive.Draw(layerno)
			}
			// Name
			if hi.nameUseActive2 {
				row.nameDataActive2.Draw(layerno)
			} else {
				row.nameDataActive.Draw(layerno)
			}
		} else {
			row.rankData.Draw(layerno)
			row.resultData.Draw(layerno)
			row.nameData.Draw(layerno)
		}
	}

	// Timer (only when enabled & during input)
	if m.HiscoreInfo.Timer.Count != -1 && hi.input && m.HiscoreInfo.Timer.TextSpriteData != nil {
		m.HiscoreInfo.Timer.TextSpriteData.Draw(layerno)
	}

	// Overlay
	m.HiscoreInfo.Overlay.RectData.Draw(layerno)
}

func (hi *MotifHiscore) finalizeAndSave() {
	if hi.haveSaved || hi.place <= 0 || hi.mode == "" {
		return
	}
	idx := int(hi.place - 1)
	if idx < 0 || idx >= len(hi.rows) {
		return
	}
	name := strings.TrimSpace(hi.rows[idx].name)
	if name == "" {
		// Keep blank if user never entered anything (matches legacy behavior).
		hi.haveSaved = true
		return
	}
	data, err := os.ReadFile("save/stats.json")
	if err != nil {
		fmt.Println("hiscore: cannot read save/stats.json for name save:", err)
		hi.haveSaved = true
		return
	}
	path := fmt.Sprintf("modes.%s.ranking.%d.name", hi.mode, idx)
	out, err := sjson.SetBytes(data, path, name)
	if err != nil {
		fmt.Println("hiscore: sjson set failed:", err)
		hi.haveSaved = true
		return
	}
	// Pretty-print the JSON
	var buf bytes.Buffer
	if err := json.Indent(&buf, out, "", "  "); err == nil {
		out = buf.Bytes()
	} else {
		fmt.Println("hiscore: pretty print failed, writing compact JSON:", err)
	}
	if err := os.WriteFile("save/stats.json", out, 0644); err != nil {
		fmt.Println("hiscore: write save/stats.json failed:", err)
	}
	hi.haveSaved = true
}

func (hi *MotifHiscore) handleBlinkers(m *Motif) {
	// Toggle between Active and Active2 fonts based on switchtime values.
	if hi.place <= 0 {
		return
	}
	if hi.rankActiveCount < m.HiscoreInfo.Item.Rank.Active.Switchtime {
		hi.rankActiveCount++
	} else {
		hi.rankUseActive2 = !hi.rankUseActive2
		hi.rankActiveCount = 0
	}
	if hi.resultActiveCount < m.HiscoreInfo.Item.Result.Active.Switchtime {
		hi.resultActiveCount++
	} else {
		hi.resultUseActive2 = !hi.resultUseActive2
		hi.resultActiveCount = 0
	}
	if hi.nameActiveCount < m.HiscoreInfo.Item.Name.Active.Switchtime {
		hi.nameActiveCount++
	} else {
		hi.nameUseActive2 = !hi.nameUseActive2
		hi.nameActiveCount = 0
	}
}

// ----------------------- Portrait helpers -----------------------

// hiscorePortraitAnim returns a prepared *Anim for a character "def key" using preloaded
// animations/sprites from sys.sel.charlist. If nothing is found, it returns nil.
func hiscorePortraitAnim(defKey string, m *Motif, x, y float32) *Anim {
	if defKey == "" {
		return nil
	}
	defKey = normalizeDefKey(defKey)
	for i := range sys.sel.charlist {
		sc := &sys.sel.charlist[i]
		if !matchDef(sc.def, defKey) {
			continue
		}
		// Prefer explicit anim number if configured, else sprite tuple.
		var animCopy *Animation
		if m.HiscoreInfo.Item.Face.Anim >= 0 {
			animCopy = sc.anims.get(m.HiscoreInfo.Item.Face.Anim, -1)
		} else if m.HiscoreInfo.Item.Face.Spr[0] >= 0 {
			grp := m.HiscoreInfo.Item.Face.Spr[0]
			idx := m.HiscoreInfo.Item.Face.Spr[1]
			animCopy = sc.anims.get(grp, idx)
		}
		if animCopy == nil {
			// Not preloaded; fall back to unknown handled by caller.
			return nil
		}
		// Wrap *Animation into *Anim
		a := NewAnim(nil, "")
		a.anim = animCopy
		// Optional tuning (localcoord/scale/window/facing) from motif face settings.
		lc := m.HiscoreInfo.Item.Face.Localcoord
		a.SetLocalcoord(float32(lc[0]), float32(lc[1]))
		a.SetPos(x, y)
		sx := m.HiscoreInfo.Item.Face.Scale[0] * sc.portraitscale * float32(sys.motif.Info.Localcoord[0]) / sc.localcoord[0]
		sy := m.HiscoreInfo.Item.Face.Scale[1] * sc.portraitscale * float32(sys.motif.Info.Localcoord[0]) / sc.localcoord[0]
		a.SetScale(sx, sy)
		w := m.HiscoreInfo.Item.Face.Window
		a.SetWindow([4]float32{float32(w[0]), float32(w[1]), float32(w[2]), float32(w[3])})
		a.layerno = m.HiscoreInfo.Item.Face.Layerno
		a.SetFacing(float32(m.HiscoreInfo.Item.Face.Facing))
		return a
	}
	return nil
}

// normalizeDefKey turns "chars/kfm/kfm.def" or "kfm.def" or "kfm" into "kfm".
func normalizeDefKey(s string) string {
	s = strings.ReplaceAll(s, "\\", "/")
	base := filepath.Base(s)
	ext := filepath.Ext(base)
	if ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	return strings.ToLower(base)
}

// matchDef checks whether a select entry path matches a stats "def key".
func matchDef(selectDefPath, key string) bool {
	if key == "" {
		return false
	}
	selectDefPath = strings.ReplaceAll(selectDefPath, "\\", "/")
	base := normalizeDefKey(selectDefPath)
	if base == key {
		return true
	}
	// Also allow directory name match (e.g., chars/kfm_zss/char.def -> kfm_zss)
	dir := strings.ToLower(filepath.Base(filepath.Dir(selectDefPath)))
	return dir == key
}

// parseRankingRows reads <path> and converts modes.<mode>.ranking into []rankingRow.
func parseRankingRows(path, mode string) []rankingRow {
	if path == "" || mode == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("hiscore: read stats.json failed:", err)
		return nil
	}
	res := gjson.GetBytes(data, "modes."+mode+".ranking")
	if !res.Exists() || !res.IsArray() {
		return nil
	}
	out := make([]rankingRow, 0, int(res.Get("#").Int()))
	res.ForEach(func(_, v gjson.Result) bool {
		row := rankingRow{
			score: int(v.Get("score").Int()),
			time:  v.Get("time").Float(),
			win:   int(v.Get("win").Int()),
			name:  v.Get("name").Str,
		}
		if ca := v.Get("chars"); ca.Exists() && ca.IsArray() {
			ca.ForEach(func(_, c gjson.Result) bool {
				row.chars = append(row.chars, c.Str)
				return true
			})
		}
		out = append(out, row)
		return true
	})
	return out
}

func cloneWithFont(src *TextSprite, font [8]int32, fnt map[int]*Fnt) *TextSprite {
	if src == nil {
		return nil
	}
	dst := src.Copy()
	dst.ApplyFontTuple(font, fnt)
	return dst
}

// ---------------- Name-entry helpers ----------------

func initialsWidth(fmtStr string) int {
	// Extract %<N>s (default 3)
	re := regexp.MustCompile(`%([0-9]+)s`)
	m := re.FindStringSubmatch(fmtStr)
	if len(m) >= 2 {
		if n := Atoi(m[1]); n > 0 {
			return int(n)
		}
	}
	return 3
}

// currentGlyph returns current glyph character for the active letter slot.
func currentGlyph(mo *Motif, letters []int) string {
	if len(letters) == 0 {
		return ""
	}
	idx := letters[len(letters)-1] // 1-based
	if idx <= 0 || idx > len(mo.HiscoreInfo.Glyphs) {
		return ""
	}
	return mo.HiscoreInfo.Glyphs[idx-1]
}

// buildNameFromLetters converts indices into the configured glyph table.
func buildNameFromLetters(mo *Motif, letters []int) string {
	var b strings.Builder
	for _, i := range letters {
		if i <= 0 || i > len(mo.HiscoreInfo.Glyphs) {
			continue
		}
		ch := mo.HiscoreInfo.Glyphs[i-1]
		if ch == ">" {
			ch = " "
		}
		b.WriteString(ch)
	}
	return b.String()
}

// updateRowNameFromLetters updates row text sprites (normal + active variants) from letters.
func updateRowNameFromLetters(mo *Motif, row *rankingRow, letters []int) {
	name := buildNameFromLetters(mo, letters)
	if mo.HiscoreInfo.Item.Name.Uppercase {
		name = strings.ToUpper(name)
	}
	row.name = name
	fmtStr := mo.replaceFormatSpecifiers(mo.HiscoreInfo.Item.Name.Text["default"])
	if row.nameData != nil {
		row.nameData.text = fmt.Sprintf(fmtStr, name)
	}
	if row.nameDataActive != nil {
		row.nameDataActive.text = fmt.Sprintf(fmtStr, name)
	}
	if row.nameDataActive2 != nil {
		row.nameDataActive2.text = fmt.Sprintf(fmtStr, name)
	}
}
