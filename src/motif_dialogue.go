package main

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MotifDialogue is the top-level container for storing parsed dialogue data.
type MotifDialogue struct {
	enabled           bool
	active            bool
	initialized       bool
	counter           int32
	char              *Char
	faceParams        [2]FaceParams
	parsed            []DialogueParsedLine
	textNum           int
	lineFullyRendered bool
	charDelayCounter  int32
	activeSide        int
	wait              int
	switchCounter     int
	endCounter        int
}

type FaceParams struct {
	grp int
	idx int
	pn  int
}

type DialogueParsedLine struct {
	side     int
	text     string
	tokens   map[int][]DialogueToken
	typedCnt int
}

type DialogueToken struct {
	param       string
	side        int
	redirection string
	pn          int
	value       []interface{}
}

func (di *MotifDialogue) dialogueRedirection(redirect string) int {
	var redirection, val string
	if parts := strings.SplitN(redirect, "(", 2); len(parts) == 2 {
		redirection = strings.ToLower(strings.TrimSpace(parts[0]))
		val = strings.TrimSpace(strings.TrimSuffix(parts[1], ")"))
	} else {
		redirection = strings.ToLower(strings.TrimSpace(redirect))
	}
	switch redirection {
	case "self":
		return di.char.playerNo + 1
	case "playerno":
		pn := int(Atoi(val))
		if pn >= 1 && pn <= len(sys.chars) && len(sys.chars[pn-1]) > 0 {
			return pn
		}
	case "partner":
		if val == "" {
			val = "0"
		}
		partnerNum := Atoi(val)
		if partner := di.char.partner(partnerNum, true); partner != nil {
			return partner.playerNo + 1
		}
	case "enemy":
		if val == "" {
			val = "0"
		}
		enemyNum := Atoi(val)
		if enemy := di.char.enemy(enemyNum); enemy != nil {
			return enemy.playerNo + 1
		}
	case "enemyname":
		for i := int32(0); i < di.char.numEnemy(); i++ {
			if enemy := di.char.enemy(i); enemy != nil {
				if strings.EqualFold(enemy.name, val) {
					return enemy.playerNo + 1
				}
			}
		}
	case "partnername":
		for i := int32(0); i < di.char.numPartner(); i++ {
			if partner := di.char.partner(i, false); partner != nil {
				if strings.EqualFold(partner.name, val) {
					return partner.playerNo + 1
				}
			}
		}
	default:
	}
	return -1
}

func (di *MotifDialogue) parseTag(tag string) []DialogueToken {
	tag = strings.TrimSpace(tag)
	pOnlyRe := regexp.MustCompile(`^p(\d+)$`)
	if pOnlyRe.MatchString(tag) {
		matches := pOnlyRe.FindStringSubmatch(tag)
		if len(matches) == 2 {
			pnValue, _ := strconv.Atoi(matches[1])
			return []DialogueToken{{
				param: "p",
				side:  -1,
				pn:    pnValue,
			}}
		}
	}
	equalIndex := strings.Index(tag, "=")
	if equalIndex == -1 {
		return nil
	}
	paramPart := tag[:equalIndex]
	valuePart := tag[equalIndex+1:]
	side := -1
	param := paramPart
	redirection := ""
	pn := -1
	numValues := []interface{}{}
	pPrefixRe := regexp.MustCompile(`^p(\d+)([a-zA-Z]+)$`)
	if pPrefixRe.MatchString(paramPart) {
		subMatches := pPrefixRe.FindStringSubmatch(paramPart)
		if len(subMatches) == 3 {
			s, _ := strconv.Atoi(subMatches[1])
			side = s
			param = subMatches[2]
		}
	}
	parts := strings.Split(valuePart, ",")
	if len(parts) > 0 {
		if _, err := strconv.Atoi(parts[0]); err != nil {
			redirection = parts[0]
			parts = parts[1:]
		}
		for _, p := range parts {
			if val, err := strconv.ParseFloat(p, 32); err == nil {
				numValues = append(numValues, float32(val))
			} else {
				numValues = append(numValues, p)
			}
		}
		pn = di.dialogueRedirection(redirection)
	}
	return []DialogueToken{{
		param:       param,
		side:        side,
		redirection: redirection,
		pn:          pn,
		value:       numValues,
	}}
}

func (di *MotifDialogue) parseLine(line string) DialogueParsedLine {
	side := -1
	re := regexp.MustCompile(`<([^>]+)>`)
	var finalText strings.Builder
	tokensMap := make(map[int][]DialogueToken)
	offset := 0
	pos := 0
	matches := re.FindAllStringIndex(line, -1)
	for _, match := range matches {
		startIdx := match[0]
		endIdx := match[1]
		if startIdx > pos {
			substr := line[pos:startIdx]
			finalText.WriteString(substr)
			offset += utf8.RuneCountInString(substr)
		}
		tokenContent := line[startIdx+1 : endIdx-1]
		parsedTokens := di.parseTag(tokenContent)
		if len(parsedTokens) == 1 && parsedTokens[0].param == "p" && parsedTokens[0].pn != -1 {
			side = parsedTokens[0].pn
		} else {
			for _, tkn := range parsedTokens {
				tokensMap[offset] = append(tokensMap[offset], tkn)
			}
		}
		pos = endIdx
	}
	if pos < len(line) {
		substr := line[pos:]
		finalText.WriteString(substr)
		offset += utf8.RuneCountInString(substr)
	}
	return DialogueParsedLine{
		side:     side,
		text:     strings.TrimSpace(finalText.String()),
		tokens:   tokensMap,
		typedCnt: 0,
	}
}

func (di *MotifDialogue) parseAll(lines []string) []DialogueParsedLine {
	var result []DialogueParsedLine
	for _, line := range lines {
		parsedLine := di.parseLine(line)
		result = append(result, parsedLine)
	}
	return result
}

func (di *MotifDialogue) preprocessNames(lines []string) []string {
	result := make([]string, len(lines))
	nameRe := regexp.MustCompile(`<(displayname|name)=([^>]+)>`)
	for i, line := range lines {
		newLine := line
		for {
			loc := nameRe.FindStringSubmatchIndex(newLine)
			if loc == nil {
				break
			}
			fullMatch := newLine[loc[0]:loc[1]]
			paramType := newLine[loc[2]:loc[3]]
			redirectionValue := newLine[loc[4]:loc[5]]
			resolvedPn := di.dialogueRedirection(redirectionValue)
			replacementText := ""
			if resolvedPn != -1 {
				if paramType == "displayname" {
					replacementText = sys.chars[resolvedPn-1][0].gi().displayname
				} else {
					replacementText = sys.chars[resolvedPn-1][0].name
				}
			}
			newLine = strings.Replace(newLine, fullMatch, replacementText, 1)
		}
		result[i] = newLine
	}
	return result
}

func (di *MotifDialogue) getDialogueLines() ([]string, int, error) {
	pn := sys.dialogueForce
	if pn != 0 && (pn < 1 || pn > MaxSimul*2+MaxAttachedChar) {
		return nil, 0, fmt.Errorf("invalid player number: %v", pn)
	}
	if pn == 0 {
		var validPlayers []int
		for i, p := range sys.chars {
			if len(p) > 0 && len(p[0].dialogue) > 0 {
				validPlayers = append(validPlayers, i+1)
			}
		}
		if len(validPlayers) > 0 {
			pn = validPlayers[rand.Int()%len(validPlayers)]
		}
	}
	lines := []string{}
	if pn >= 1 && pn <= len(sys.chars) && len(sys.chars[pn-1]) > 0 {
		for _, line := range sys.chars[pn-1][0].dialogue {
			lines = append(lines, line)
		}
	}
	return lines, pn, nil
}

// reset re-initializes certain state and animations.
func (di *MotifDialogue) reset(m *Motif) {
	di.active = false
	di.initialized = false
	di.counter = 0
	di.textNum = 0
	di.wait = 0
	di.lineFullyRendered = false
	di.charDelayCounter = 0
	di.switchCounter = 0
	di.endCounter = 0

	m.DialogueInfo.P1.Bg.AnimData.Reset()
	m.DialogueInfo.P2.Bg.AnimData.Reset()
	m.DialogueInfo.P1.Face.AnimData.Reset()
	m.DialogueInfo.P2.Face.AnimData.Reset()
	m.DialogueInfo.P1.Active.AnimData.Reset()
	m.DialogueInfo.P2.Active.AnimData.Reset()

	m.DialogueInfo.P1.Text.TextSpriteData.text = ""
	m.DialogueInfo.P2.Text.TextSpriteData.text = ""
	// Dialogue uses its own typewriter logic, so disable the internal TextSprite typing.
	m.DialogueInfo.P1.Text.TextSpriteData.textDelay = 0
	m.DialogueInfo.P2.Text.TextSpriteData.textDelay = 0
}

func (di *MotifDialogue) clear(m *Motif) {
	for _, p := range sys.chars {
		if len(p) > 0 {
			p[0].dialogue = nil
		}
	}
	di.initialized = false
	sys.dialogueForce = 0
	sys.dialogueBarsFlg = false
	m.DialogueInfo.P1.Face.AnimData.anim = nil
	m.DialogueInfo.P2.Face.AnimData.anim = nil
}

func (di *MotifDialogue) init(m *Motif) {
	if !m.DialogueInfo.Enabled || !di.enabled {
		di.initialized = true
		return
	}

	di.reset(m)

	lines, pn, _ := di.getDialogueLines()
	di.char = sys.chars[pn-1][0]

	lines = di.preprocessNames(lines)
	di.parsed = di.parseAll(lines)

	/*for i, line := range di.parsed {
		fmt.Printf("\nLine %d, side=%d\nText: %q\nTokens:\n", i+1, line.side, line.text)
		for textPos, tokens := range line.tokens {
			for _, t := range tokens {
				fmt.Printf("  atPos=%d  -> Param=%q Side=%d Redir=%q Pn=%d Value=%v\n",
					textPos, t.param, t.side, t.redirection, t.pn, t.value)
			}
		}
	}*/

	di.active = true
	di.initialized = true
}

// applyTokens checks and applies tokens at the current typed length in the text.
func (di *MotifDialogue) applyTokens(m *Motif, line *DialogueParsedLine) {
	typedLen := int(line.typedCnt)
	runeCount := utf8.RuneCountInString(line.text)
	if typedLen > runeCount {
		typedLen = runeCount
	}

	for i := 0; i <= typedLen; i++ {
		if tokenList, exists := line.tokens[i]; exists && len(tokenList) > 0 {
			for idx := len(tokenList) - 1; idx >= 0; idx-- {
				token := tokenList[idx]
				applied := di.applyToken(m, line, token, i)
				if applied {
					// remove token
					tokenList = append(tokenList[:idx], tokenList[idx+1:]...)
				}
			}
			line.tokens[i] = tokenList
		}
	}
}

// setFace changes the face anim for the given side.
func (di *MotifDialogue) setFace(pn, grp, idx int) *Animation {
	if pn < 1 || pn > len(sys.chars) || len(sys.chars[pn-1]) == 0 {
		return nil
	}
	c := sys.chars[pn-1][0]
	a := NewAnim(nil, "")
	var ok bool
	if sp := c.gi().sff.GetSprite(uint16(grp), uint16(idx)); sp != nil {
		action := fmt.Sprintf("%d, %d, 0, 0, -1", grp, idx)
		a = NewAnim(c.gi().sff, action)
		ok = (a != nil)
	} else if grp >= 0 && idx == -1 {
		if a.anim = c.gi().animTable.get(int32(grp)); a.anim != nil {
			ok = true
		}
	}
	if ok {
		a.palfx = c.getPalfx()
		return a.anim
	}
	return nil
}

// applyToken handles the application of a single DialogueToken.
// Returns true if the token should be removed after application.
func (di *MotifDialogue) applyToken(m *Motif, line *DialogueParsedLine, token DialogueToken, index int) bool {
	switch token.param {
	case "clear":
		line.text = ""
	case "wait":
		if len(token.value) > 0 {
			if waitFrames, ok := token.value[0].(float32); ok {
				di.wait = int(waitFrames)
			}
		}
		return true
	case "face":
		if token.side == 1 || token.side == 2 {
			if len(token.value) >= 1 {
				if v1, ok := token.value[0].(float32); ok {
					grp := int(v1)
					idx := -1
					if len(token.value) >= 2 {
						if v2, ok := token.value[1].(float32); ok {
							idx = int(v2)
						}
					}
					if di.faceParams[token.side-1].pn != token.pn || di.faceParams[token.side-1].grp != grp ||
						di.faceParams[token.side-1].idx != idx {
						if anim := di.setFace(token.pn, grp, idx); anim != nil {
							if token.side == 1 {
								m.DialogueInfo.P2.Face.AnimData.anim = anim
							} else if token.side == 2 {
								m.DialogueInfo.P1.Face.AnimData.anim = anim
							}
							di.faceParams[token.side-1].pn = token.pn
							di.faceParams[token.side-1].grp = grp
							di.faceParams[token.side-1].idx = idx
						}
					}
				}
			}
		}
		return true
	case "name":
		if token.pn != -1 {
			name := sys.chars[token.pn-1][0].gi().displayname
			if token.side == 1 {
				m.DialogueInfo.P1.Name.TextSpriteData.text = name
			} else if token.side == 2 {
				m.DialogueInfo.P2.Name.TextSpriteData.text = name
			}
		} else if name, ok := token.value[0].(string); ok {
			m.DialogueInfo.P2.Name.TextSpriteData.text = name
		}
		return true
	case "sound":
		if len(token.value) >= 2 {
			f, lw, lp, stopgh, stopcs := false, false, false, false, false
			var g, n, ch, vo, priority, lc int32 = -1, 0, -1, 100, 0, 0
			var loopstart, loopend, startposition int = 0, 0, 0
			var p, fr float32 = 0, 1
			x := &sys.chars[token.pn-1][0].pos[0]
			ls := sys.chars[token.pn-1][0].localscl
			prefix := ""
			if f {
				prefix = "f"
			}
			if v1, ok1 := token.value[0].(float32); ok1 {
				g = int32(v1)
				if v2, ok2 := token.value[1].(float32); ok2 {
					n = int32(v2)
					if len(token.value) >= 3 {
						if v3, ok3 := token.value[2].(float32); ok3 {
							vo = int32(v3)
						}
					}
				}
			}
			if lc == 0 {
				if lp {
					sys.chars[token.pn-1][0].playSound(prefix, lw, -1, g, n, ch, vo, p, fr, ls, x, false, priority, loopstart, loopend, startposition, stopgh, stopcs)
				} else {
					sys.chars[token.pn-1][0].playSound(prefix, lw, 0, g, n, ch, vo, p, fr, ls, x, false, priority, loopstart, loopend, startposition, stopgh, stopcs)
				}
				// Otherwise, read the loopcount parameter directly
			} else {
				sys.chars[token.pn-1][0].playSound(prefix, lw, lc, g, n, ch, vo, p, fr, ls, x, false, priority, loopstart, loopend, startposition, stopgh, stopcs)
			}
		}
		return true
	case "anim":
		if len(token.value) >= 1 {
			if v, ok := token.value[0].(float32); ok {
				animNo := int32(v)
				if sys.chars[token.pn-1][0].selfAnimExist(BytecodeInt(animNo)) == BytecodeBool(true) {
					sys.chars[token.pn-1][0].changeAnim(animNo, token.pn-1, -1, "")
				}
			}
		}
		return true
	case "state":
		if len(token.value) >= 1 {
			if v, ok := token.value[0].(float32); ok {
				stateNo := int32(v)
				if stateNo == -1 {
					for _, ch := range sys.chars[token.pn-1] {
						ch.setSCF(SCF_disabled)
					}
				} else if sys.chars[token.pn-1][0].selfStatenoExist(BytecodeInt(stateNo)) == BytecodeBool(true) {
					for _, ch := range sys.chars[token.pn-1] {
						if ch.scf(SCF_disabled) {
							ch.unsetSCF(SCF_disabled)
						}
					}
					sys.chars[token.pn-1][0].changeState(int32(stateNo), -1, -1, "")
				}
			}
			return true
		}
		return true
	case "map":
		if len(token.value) >= 2 {
			mapName, ok1 := token.value[0].(string)
			mapVal, ok2 := token.value[1].(float32)
			if !ok1 || !ok2 {
				return false
			}
			mapOp := int32(0)
			if len(token.value) >= 3 {
				if op, ok3 := token.value[2].(string); ok3 && op == "add" {
					mapOp = 1
				}
			}
			sys.chars[token.pn-1][0].mapSet(mapName, mapVal, mapOp)
		}
		return true
	default:
		// Unrecognized token parameter.
	}
	return false
}

// step processes dialogue state each frame, handling timing, skipping, cancel, wrapping, etc.
func (di *MotifDialogue) step(m *Motif) {
	// If we have no lines, do nothing
	if len(di.parsed) == 0 {
		return
	}

	// If user presses "cancel", end the dialogue
	if m.button(m.DialogueInfo.Cancel.Key, -1) {
		di.active = false
		di.clear(m)
		return
	}

	// Update any background/face/active animations
	m.DialogueInfo.P1.Bg.AnimData.Update()
	m.DialogueInfo.P2.Bg.AnimData.Update()
	m.DialogueInfo.P1.Face.AnimData.Update()
	m.DialogueInfo.P2.Face.AnimData.Update()
	if di.activeSide == 1 {
		m.DialogueInfo.P1.Active.AnimData.Update()
	} else if di.activeSide == 2 {
		m.DialogueInfo.P2.Active.AnimData.Update()
	}

	// Check if we haven't reached StartTime yet
	if di.counter < m.DialogueInfo.StartTime {
		di.counter++
		return
	}

	// Check if we've gone past all lines
	if di.textNum >= len(di.parsed) {
		// If we haven't started EndCounter yet, do so
		if di.endCounter == 0 {
			di.endCounter = int(m.DialogueInfo.EndTime)
		} else {
			di.endCounter--
			if di.endCounter <= 0 {
				// Done
				di.active = false
				di.clear(m)
				return
			}
		}
		di.counter++
		return
	}

	// We have a valid line to render
	currentLine := &di.parsed[di.textNum]
	di.activeSide = currentLine.side
	prevLineFullyRendered := di.lineFullyRendered

	// Handle "skip" key (only after SkipTime)
	if di.counter >= m.DialogueInfo.SkipTime {
		if m.button(m.DialogueInfo.Skip.Key, -1) {
			if !di.lineFullyRendered {
				currentLine.typedCnt = utf8.RuneCountInString(currentLine.text)
				di.lineFullyRendered = true
				di.switchCounter = 0
				di.wait = 0
			} else {
				// If line is already fully rendered => move to next line
				di.advanceLine(m)
				return
			}
		}
	}

	// Determine the per-character delay for this side
	var charDelay float32
	if currentLine.side == 1 {
		charDelay = float32(m.DialogueInfo.P1.Text.TextDelay)
	} else if currentLine.side == 2 {
		charDelay = float32(m.DialogueInfo.P2.Text.TextDelay)
	} else {
		charDelay = 1
	}

	// Handle any explicit token-based wait
	if di.wait > 0 {
		di.wait--
	} else if !di.lineFullyRendered {
		// Otherwise, reveal letters one by one
		StepTypewriter(
			currentLine.text,
			&currentLine.typedCnt,
			&di.charDelayCounter,
			&di.lineFullyRendered,
			charDelay,
		)
	}

	// Apply any tokens for newly revealed characters
	di.applyTokens(m, currentLine)

	// Clamp typedLen so it doesn't exceed the line length
	typedLen := currentLine.typedCnt
	runeCount := utf8.RuneCountInString(currentLine.text)
	if typedLen > runeCount {
		typedLen = runeCount
	}

	// If we've just finished the line
	if typedLen >= runeCount && !prevLineFullyRendered {
		di.lineFullyRendered = true // StepTypewriter already set this, but keep explicit.
		di.switchCounter = 0
	}

	// If line is fully rendered, handle auto-switch after SwitchTime
	if di.lineFullyRendered {
		di.switchCounter++
		if di.switchCounter >= int(m.DialogueInfo.SwitchTime) {
			di.advanceLine(m)
			return
		}
	}

	if currentLine.side == 1 {
		m.DialogueInfo.P1.Text.TextSpriteData.wrapText(currentLine.text, typedLen)
		m.DialogueInfo.P1.Text.TextSpriteData.Update()
	} else if currentLine.side == 2 {
		m.DialogueInfo.P2.Text.TextSpriteData.wrapText(currentLine.text, typedLen)
		m.DialogueInfo.P2.Text.TextSpriteData.Update()
	}

	// Finally increment the global frame counter
	di.counter++
}

// advanceLine moves to the next line, clearing or preserving text depending on side.
func (di *MotifDialogue) advanceLine(m *Motif) {
	// Clear text if next line uses the same side; preserve if different side
	currentSide := -1
	if di.textNum < len(di.parsed) {
		currentSide = di.parsed[di.textNum].side
	}

	di.textNum++
	if di.textNum < len(di.parsed) {
		nextSide := di.parsed[di.textNum].side
		if nextSide == currentSide {
			// Same side => replace text with the new line, so clear now
			if currentSide == 1 {
				m.DialogueInfo.P1.Text.TextSpriteData.text = ""
			} else if currentSide == 2 {
				m.DialogueInfo.P2.Text.TextSpriteData.text = ""
			}
		}
	} else {
		// If we're out of lines, text is presumably done
	}

	// Reset state
	di.lineFullyRendered = false
	di.switchCounter = 0
	di.wait = 0
	di.charDelayCounter = 0
}

// draw renders the dialogue on the screen based on the current state.
func (di *MotifDialogue) draw(m *Motif, layerno int16) {
	// BG
	m.DialogueInfo.P1.Bg.AnimData.Draw(layerno)
	m.DialogueInfo.P2.Bg.AnimData.Draw(layerno)

	// If we haven't reached StartTime yet, or no lines, skip drawing text
	if di.counter < m.DialogueInfo.StartTime || len(di.parsed) == 0 {
		return
	}

	// Names
	m.DialogueInfo.P1.Name.TextSpriteData.Draw(layerno)
	m.DialogueInfo.P2.Name.TextSpriteData.Draw(layerno)

	// Faces
	m.DialogueInfo.P1.Face.AnimData.Draw(layerno)
	m.DialogueInfo.P2.Face.AnimData.Draw(layerno)

	// Text
	m.DialogueInfo.P1.Text.TextSpriteData.Draw(layerno)
	m.DialogueInfo.P2.Text.TextSpriteData.Draw(layerno)

	// Active anim highlight
	if di.activeSide == 1 {
		m.DialogueInfo.P1.Active.AnimData.Draw(layerno)
	} else if di.activeSide == 2 {
		m.DialogueInfo.P2.Active.AnimData.Draw(layerno)
	}
}
