package main

import (
	"math"
	"time"

	ggpo "github.com/assemblaj/GGPO-Go/pkg"
	lua "github.com/yuin/gopher-lua"
)

type RollbackSystem struct {
	session      *RollbackSession
	currentFight Fight
}

type RollbackConfig struct {
	FrameDelay            int  `json:"frameDelay"`
	DisconnectNotifyStart int  `json:"disconnectNotifyStart"`
	DisconnectTimeout     int  `json:"disconnectTimeout"`
	LogsEnabled           bool `json:"logsEnabled"`
}

func (rs *RollbackSystem) fight(s *System) bool {
	// Reset variables
	s.gameTime, s.paused, s.accel = 0, false, 1
	s.aiInput = [len(s.aiInput)]AiInput{}
	rs.currentFight = NewFight()
	// Defer resetting variables on return
	defer rs.currentFight.endFight()

	s.debugWC = sys.chars[0][0]
	// debugInput := func() {
	// 	select {
	// 	case cl := <-s.commandLine:
	// 		if err := s.luaLState.DoString(cl); err != nil {
	// 			s.errLog.Println(err.Error())
	// 		}
	// 	default:
	// 	}
	// }

	// Synchronize with external inputs (netplay, replays, etc)
	if err := s.synchronize(); err != nil {
		s.errLog.Println(err.Error())
		s.esc = true
	}
	if s.netInput != nil {
		defer s.netInput.Stop()
	}
	s.wincnt.init()

	// Initialize super meter values, and max power for teams sharing meter
	rs.currentFight.initSuperMeter()
	rs.currentFight.initTeamsLevels()

	rs.currentFight.initChars()

	//default bgm playback, used only in Quick VS or if externalized Lua implementaion is disabled
	if s.round == 1 && (s.gameMode == "" || len(sys.commonLua) == 0) {
		s.bgm.Open(s.stage.bgmusic, 1, int(s.stage.bgmvolume), int(s.stage.bgmloopstart), int(s.stage.bgmloopend), 0)
	}

	rs.currentFight.oldWins, rs.currentFight.oldDraws = s.wins, s.draws
	rs.currentFight.oldTeamLeader = s.teamLeader

	var running bool
	if rs.session != nil && sys.netInput != nil {
		if rs.session.host != "" {
			rs.session.InitP2(2, 7550, 7600, rs.session.host)
			rs.session.playerNo = 2
		} else {
			rs.session.InitP1(2, 7600, 7550, rs.session.remoteIp)
			rs.session.playerNo = 1
		}
		if !rs.session.IsConnected() {
			rs.session.backend.Idle(0)
		}
		sys.netInput.Close()
		sys.netInput = nil
	} else if sys.netInput == nil && rs.session == nil {
		rs.session.InitSyncTest(2)
	}

	rs.currentFight.reset()
	// Loop until end of match
	///fin := false
	for !s.endMatch {

		rs.session.now = time.Now().UnixMilli()
		err := rs.session.backend.Idle(
			int(math.Max(0, float64(rs.session.next-rs.session.now-1))))
		if err != nil {
			panic(err)
		}

		running = rs.runFrame(s)
		rs.session.next = rs.session.now + 1000/60

		if !running {
			break
		}

		rs.render(s)
		sys.update()
	}
	rs.session.SaveReplay()

	return false
}

func (rs *RollbackSystem) runFrame(s *System) bool {
	var buffer []byte
	var result error
	if rs.session.syncTest {
		buffer = getInputs(0)
		result = rs.session.backend.AddLocalInput(ggpo.PlayerHandle(0), buffer, len(buffer))
		buffer = getInputs(1)
		result = rs.session.backend.AddLocalInput(ggpo.PlayerHandle(1), buffer, len(buffer))
	} else {
		buffer = getInputs(0)
		result = rs.session.backend.AddLocalInput(rs.session.currentPlayerHandle, buffer, len(buffer))
	}

	if result == nil {

		var values [][]byte
		disconnectFlags := 0
		values, result = rs.session.backend.SyncInput(&disconnectFlags)
		inputs := decodeInputs(values)
		if result == nil {
			s.step = false
			rs.runShortcutScripts(s)

			// If next round
			if !rs.runNextRound(s) {
				return false
			}

			// If frame is ready to tick and not paused
			rs.updateStage(s)

			// Update game state
			rs.action(s, inputs)

			if rs.handleFlags(s) {
				return true
			}

			if !rs.updateEvents(s) {
				return false
			}

			// Break if finished
			if rs.currentFight.fin && (!s.postMatchFlg || len(s.commonLua) == 0) {
				return false
			}

			// Update system; break if update returns false (game ended)
			//if !s.update() {
			//	return false
			//}

			// If end match selected from menu/end of attract mode match/etc
			if s.endMatch {
				s.esc = true
				return false
			} else if s.esc {
				s.endMatch = s.netInput != nil || len(sys.commonLua) == 0
				return false
			}
			err := rs.session.backend.AdvanceFrame()
			if err != nil {
				panic(err)
			}

		}
	}
	return true

}

func (rs *RollbackSystem) runShortcutScripts(s *System) {
	for _, v := range s.shortcutScripts {
		if v.Activate {
			if err := s.luaLState.DoString(v.Script); err != nil {
				s.errLog.Println(err.Error())
			}
		}
	}
}

func (rs *RollbackSystem) runNextRound(s *System) bool {
	if s.roundOver() && !rs.currentFight.fin {
		s.round++
		for i := range s.roundsExisted {
			s.roundsExisted[i]++
		}
		s.clearAllSound()
		tbl_roundNo := s.luaLState.NewTable()
		for _, p := range s.chars {
			if len(p) > 0 && p[0].teamside != -1 {
				tmp := s.luaLState.NewTable()
				tmp.RawSetString("name", lua.LString(p[0].name))
				tmp.RawSetString("id", lua.LNumber(p[0].id))
				tmp.RawSetString("memberNo", lua.LNumber(p[0].memberNo))
				tmp.RawSetString("selectNo", lua.LNumber(p[0].selectNo))
				tmp.RawSetString("teamside", lua.LNumber(p[0].teamside))
				tmp.RawSetString("life", lua.LNumber(p[0].life))
				tmp.RawSetString("lifeMax", lua.LNumber(p[0].lifeMax))
				tmp.RawSetString("winquote", lua.LNumber(p[0].winquote))
				tmp.RawSetString("aiLevel", lua.LNumber(p[0].aiLevel()))
				tmp.RawSetString("palno", lua.LNumber(p[0].palno()))
				tmp.RawSetString("ratiolevel", lua.LNumber(p[0].ocd().ratioLevel))
				tmp.RawSetString("win", lua.LBool(p[0].win()))
				tmp.RawSetString("winKO", lua.LBool(p[0].winKO()))
				tmp.RawSetString("winTime", lua.LBool(p[0].winTime()))
				tmp.RawSetString("winPerfect", lua.LBool(p[0].winPerfect()))
				tmp.RawSetString("winSpecial", lua.LBool(p[0].winType(WT_S)))
				tmp.RawSetString("winHyper", lua.LBool(p[0].winType(WT_H)))
				tmp.RawSetString("drawgame", lua.LBool(p[0].drawgame()))
				tmp.RawSetString("ko", lua.LBool(p[0].scf(SCF_ko)))
				tmp.RawSetString("ko_round_middle", lua.LBool(p[0].scf(SCF_ko_round_middle)))
				tmp.RawSetString("firstAttack", lua.LBool(p[0].firstAttack))
				tbl_roundNo.RawSetInt(p[0].playerNo+1, tmp)
				p[0].firstAttack = false
			}
		}
		s.matchData.RawSetInt(int(s.round-1), tbl_roundNo)
		s.scoreRounds = append(s.scoreRounds, [2]float32{s.lifebar.sc[0].scorePoints, s.lifebar.sc[1].scorePoints})
		rs.currentFight.oldTeamLeader = s.teamLeader

		if !s.matchOver() && (s.tmode[0] != TM_Turns || s.chars[0][0].win()) &&
			(s.tmode[1] != TM_Turns || s.chars[1][0].win()) {
			/* Prepare for the next round */
			for i, p := range s.chars {
				if len(p) > 0 {
					if s.tmode[i&1] != TM_Turns || !p[0].win() {
						p[0].life = p[0].lifeMax
					} else if p[0].life <= 0 {
						p[0].life = 1
					}
					p[0].redLife = 0
					rs.currentFight.copyVar(i)
				}
			}
			rs.currentFight.oldWins, rs.currentFight.oldDraws = s.wins, s.draws
			rs.currentFight.oldStageVars.copyStageVars(s.stage)
			rs.currentFight.reset()
		} else {
			/* End match, or prepare for a new character in turns mode */
			for i, tm := range s.tmode {
				if s.chars[i][0].win() || !s.chars[i][0].lose() && tm != TM_Turns {
					for j := i; j < len(s.chars); j += 2 {
						if len(s.chars[j]) > 0 {
							if s.chars[j][0].win() {
								s.chars[j][0].life = Max(1, int32(math.Ceil(math.Pow(rs.currentFight.lvmul,
									float64(rs.currentFight.level[i]))*float64(s.chars[j][0].life))))
							} else {
								s.chars[j][0].life = Max(1, s.cgi[j].data.life)
							}
						}
					}
					//} else {
					//	s.chars[i][0].life = 0
				}
			}
			// If match isn't over, presumably this is turns mode,
			// so break to restart fight for the next character
			if !s.matchOver() {
				return false
			}

			// Otherwise match is over
			s.postMatchFlg = true
			rs.currentFight.fin = true
		}
	}
	return true

}

func (rs *RollbackSystem) updateStage(s *System) {
	if s.tickFrame() && (s.super <= 0 || !s.superpausebg) &&
		(s.pause <= 0 || !s.pausebg) {
		// Update stage
		s.stage.action()
	}
}

func (rs *RollbackSystem) action(sys *System, input []InputBits) {

}

func (rs *RollbackSystem) handleFlags(s *System) bool {
	// F4 pressed to restart round
	if s.roundResetFlg && !s.postMatchFlg {
		rs.currentFight.reset()
	}
	// Shift+F4 pressed to restart match
	if s.reloadFlg {
		return true
	}
	return false

}

func (rs *RollbackSystem) updateEvents(s *System) bool {
	if !s.addFrameTime(s.turbo) {
		if !s.eventUpdate() {
			return false
		}
		return false
	}
	return true
}

func (rs *RollbackSystem) updateCamera(sys *System) {

}

func (rs *RollbackSystem) render(s *System) {
	if !s.frameSkip {
		x, y, scl := s.cam.Pos[0], s.cam.Pos[1], s.cam.Scale/s.cam.BaseScale()
		dx, dy, dscl := x, y, scl
		if s.enableZoomstate {
			if !s.debugPaused() {
				s.zoomPosXLag += ((s.zoomPos[0] - s.zoomPosXLag) * (1 - s.zoomlag))
				s.zoomPosYLag += ((s.zoomPos[1] - s.zoomPosYLag) * (1 - s.zoomlag))
				s.drawScale = s.drawScale / (s.drawScale + (s.zoomScale*scl-s.drawScale)*s.zoomlag) * s.zoomScale * scl
			}
			if s.zoomCameraBound {
				dscl = MaxF(s.cam.MinScale, s.drawScale/s.cam.BaseScale())
				dx = s.cam.XBound(dscl, x+s.zoomPosXLag/scl)
			} else {
				dscl = s.drawScale / s.cam.BaseScale()
				dx = x + s.zoomPosXLag/scl
			}
			dy = y + s.zoomPosYLag
		} else {
			s.zoomlag = 0
			s.zoomPosXLag = 0
			s.zoomPosYLag = 0
			s.zoomScale = 1
			s.zoomPos = [2]float32{0, 0}
			s.drawScale = s.cam.Scale
		}
		s.draw(dx, dy, dscl)
	}
	//Lua code executed before drawing fade, clsns and debug
	for _, str := range s.commonLua {
		if err := s.luaLState.DoString(str); err != nil {
			s.luaLState.RaiseError(err.Error())
		}
	}
	// Render debug elements
	if !s.frameSkip {
		s.drawTop()
		s.drawDebug()
	}
}

func (rs *RollbackSystem) commandUpdate(ib []InputBits, sys *System) {
	for i, p := range sys.chars {
		if len(p) > 0 {
			r := p[0]
			if (r.ctrlOver() && !r.sf(CSF_postroundinput)) || r.sf(CSF_noinput) ||
				(r.aiLevel() > 0 && !r.alive()) {
				for j := range r.cmd {
					r.cmd[j].BufReset()
				}
				continue
			}
			act := true
			if sys.super > 0 {
				act = r.superMovetime != 0
			} else if sys.pause > 0 && r.pauseMovetime == 0 {
				act = false
			}
			if act && !r.sf(CSF_noautoturn) &&
				(r.ss.no == 0 || r.ss.no == 11 || r.ss.no == 20) {
				r.turn()
			}

			for _, c := range p {
				if c.helperIndex == 0 ||
					c.helperIndex > 0 && &c.cmd[0] != &r.cmd[0] {
					if i < len(ib) {
						// if we have an input from the players
						// update the command buffer based on that.
						c.cmd[0].Buffer.InputBits(ib[i], int32(c.facing))
					} else {
						// Otherwise, this will ostensibly update the buffers based on AIInput
						c.cmd[0].Input(c.key, int32(c.facing), sys.com[i], c.inputFlag)
					}
					hp := c.hitPause() && c.gi().constants["input.pauseonhitpause"] != 0
					buftime := Btoi(hp && c.gi().ver[0] != 1)
					if sys.super > 0 {
						if !act && sys.super <= sys.superendcmdbuftime {
							hp = true
						}
					} else if sys.pause > 0 {
						if !act && sys.pause <= sys.pauseendcmdbuftime {
							hp = true
						}
					}
					for j := range c.cmd {
						c.cmd[j].Step(int32(c.facing), c.key < 0, hp, buftime+Btoi(hp))
					}
				}
			}
		}
	}
}

func (rs *RollbackSystem) rollbackAction(sys *System, cl *CharList, ib []InputBits,
	x float32, cvmin, cvmax, highest, lowest, leftest, rightest *float32) {
	rs.commandUpdate(ib, sys)
	// Prepare characters before performing their actions
	for i := 0; i < len(cl.runOrder); i++ {
		cl.runOrder[i].actionPrepare()
	}
	// Run character state controllers
	// Process priority based on movetype: A > I > H (or anything else)
	for i := 0; i < len(cl.runOrder); i++ {
		if cl.runOrder[i].ss.moveType == MT_A {
			cl.runOrder[i].actionRun()
		}
	}
	for i := 0; i < len(cl.runOrder); i++ {
		if cl.runOrder[i].ss.moveType == MT_I {
			cl.runOrder[i].actionRun()
		}
	}
	for i := 0; i < len(cl.runOrder); i++ {
		cl.runOrder[i].actionRun()
	}
	// Finish performing character actions
	for i := 0; i < len(cl.runOrder); i++ {
		cl.runOrder[i].actionFinish()
	}
	// Update chars
	sys.charUpdate(cvmin, cvmax, highest, lowest, leftest, rightest)
}

func getAIInputs(player int) []byte {
	var ib InputBits
	ib.SetInputAI(player)
	return writeI32(int32(ib))
}

func (ib *InputBits) SetInputAI(in int) {
	*ib = InputBits(Btoi(sys.aiInput[in].U()) |
		Btoi(sys.aiInput[in].D())<<1 |
		Btoi(sys.aiInput[in].L())<<2 |
		Btoi(sys.aiInput[in].R())<<3 |
		Btoi(sys.aiInput[in].a())<<4 |
		Btoi(sys.aiInput[in].b())<<5 |
		Btoi(sys.aiInput[in].c())<<6 |
		Btoi(sys.aiInput[in].x())<<7 |
		Btoi(sys.aiInput[in].y())<<8 |
		Btoi(sys.aiInput[in].z())<<9 |
		Btoi(sys.aiInput[in].s())<<10 |
		Btoi(sys.aiInput[in].d())<<11 |
		Btoi(sys.aiInput[in].w())<<12 |
		Btoi(sys.aiInput[in].m())<<13)
}

type Fight struct {
	fin                          bool
	oldTeamLeader                [2]int
	oldWins                      [2]int32
	oldDraws                     int32
	oldStageVars                 Stage
	level                        []int32
	lvmul                        float64
	life, pow, gpow, spow, rlife []int32
	ivar                         [][]int32
	fvar                         [][]float32
	dialogue                     [][]string
	mapArray                     []map[string]float32
	remapSpr                     []RemapPreset
}

func (f *Fight) copyVar(pn int) {
	f.life[pn] = sys.chars[pn][0].life
	f.pow[pn] = sys.chars[pn][0].power
	f.gpow[pn] = sys.chars[pn][0].guardPoints
	f.spow[pn] = sys.chars[pn][0].dizzyPoints
	f.rlife[pn] = sys.chars[pn][0].redLife
	if len(f.ivar[pn]) < len(sys.chars[pn][0].ivar) {
		f.ivar[pn] = make([]int32, len(sys.chars[pn][0].ivar))
	}
	copy(f.ivar[pn], sys.chars[pn][0].ivar[:])
	if len(f.fvar[pn]) < len(sys.chars[pn][0].fvar) {
		f.fvar[pn] = make([]float32, len(sys.chars[pn][0].fvar))
	}
	copy(f.fvar[pn], sys.chars[pn][0].fvar[:])
	copy(f.dialogue[pn], sys.chars[pn][0].dialogue[:])
	f.mapArray[pn] = make(map[string]float32)
	for k, v := range sys.chars[pn][0].mapArray {
		f.mapArray[pn][k] = v
	}
	f.remapSpr[pn] = make(RemapPreset)
	for k, v := range sys.chars[pn][0].remapSpr {
		f.remapSpr[pn][k] = v
	}
	// Reset hitScale.
	sys.chars[pn][0].defaultHitScale = newHitScaleArray()
	sys.chars[pn][0].activeHitScale = make(map[int32][3]*HitScale)
	sys.chars[pn][0].nextHitScale = make(map[int32][3]*HitScale)

}

func (f *Fight) reset() {
	sys.wins, sys.draws = f.oldWins, f.oldDraws
	sys.teamLeader = f.oldTeamLeader
	for i, p := range sys.chars {
		if len(p) > 0 {
			p[0].life = f.life[i]
			p[0].power = f.pow[i]
			p[0].guardPoints = f.gpow[i]
			p[0].dizzyPoints = f.spow[i]
			p[0].redLife = f.rlife[i]
			copy(p[0].ivar[:], f.ivar[i])
			copy(p[0].fvar[:], f.fvar[i])
			copy(p[0].dialogue[:], f.dialogue[i])
			p[0].mapArray = make(map[string]float32)
			for k, v := range f.mapArray[i] {
				p[0].mapArray[k] = v
			}
			p[0].remapSpr = make(RemapPreset)
			for k, v := range f.remapSpr[i] {
				p[0].remapSpr[k] = v
			}

			// Reset hitScale
			p[0].defaultHitScale = newHitScaleArray()
			p[0].activeHitScale = make(map[int32][3]*HitScale)
			p[0].nextHitScale = make(map[int32][3]*HitScale)
		}
	}
	sys.stage.copyStageVars(&f.oldStageVars)
	sys.resetFrameTime()
	sys.nextRound()
	sys.roundResetFlg, sys.introSkipped = false, false
	sys.reloadFlg, sys.reloadStageFlg, sys.reloadLifebarFlg = false, false, false
	sys.cam.Update(sys.cam.startzoom, 0, 0)
}

func (f *Fight) endFight() {
	sys.oldNextAddTime = 1
	sys.nomusic = false
	sys.allPalFX.clear()
	sys.allPalFX.enable = false
	for i, p := range sys.chars {
		if len(p) > 0 {
			sys.playerClear(i, sys.matchOver() || (sys.tmode[i&1] == TM_Turns && p[0].life <= 0))
		}
	}
	sys.wincnt.update()
}

func (f *Fight) initChars() {
	// Initialize each character
	f.lvmul = math.Pow(2, 1.0/12)
	for i, p := range sys.chars {
		if len(p) > 0 {
			// Get max life, and adjust based on team mode
			var lm float32
			if p[0].ocd().lifeMax != -1 {
				lm = float32(p[0].ocd().lifeMax) * p[0].ocd().lifeRatio * sys.lifeMul
			} else {
				lm = float32(p[0].gi().data.life) * p[0].ocd().lifeRatio * sys.lifeMul
			}
			if p[0].teamside != -1 {
				switch sys.tmode[i&1] {
				case TM_Single:
					switch sys.tmode[(i+1)&1] {
					case TM_Simul, TM_Tag:
						lm *= sys.team1VS2Life
					case TM_Turns:
						if sys.numTurns[(i+1)&1] < sys.matchWins[(i+1)&1] && sys.lifeShare[i&1] {
							lm = lm * float32(sys.numTurns[(i+1)&1]) /
								float32(sys.matchWins[(i+1)&1])
						}
					}
				case TM_Simul, TM_Tag:
					switch sys.tmode[(i+1)&1] {
					case TM_Simul, TM_Tag:
						if sys.numSimul[(i+1)&1] < sys.numSimul[i&1] && sys.lifeShare[i&1] {
							lm = lm * float32(sys.numSimul[(i+1)&1]) / float32(sys.numSimul[i&1])
						}
					case TM_Turns:
						if sys.numTurns[(i+1)&1] < sys.numSimul[i&1]*sys.matchWins[(i+1)&1] && sys.lifeShare[i&1] {
							lm = lm * float32(sys.numTurns[(i+1)&1]) /
								float32(sys.numSimul[i&1]*sys.matchWins[(i+1)&1])
						}
					default:
						if sys.lifeShare[i&1] {
							lm /= float32(sys.numSimul[i&1])
						}
					}
				case TM_Turns:
					switch sys.tmode[(i+1)&1] {
					case TM_Single:
						if sys.matchWins[i&1] < sys.numTurns[i&1] && sys.lifeShare[i&1] {
							lm = lm * float32(sys.matchWins[i&1]) / float32(sys.numTurns[i&1])
						}
					case TM_Simul, TM_Tag:
						if sys.numSimul[(i+1)&1]*sys.matchWins[i&1] < sys.numTurns[i&1] && sys.lifeShare[i&1] {
							lm = lm * sys.team1VS2Life *
								float32(sys.numSimul[(i+1)&1]*sys.matchWins[i&1]) /
								float32(sys.numTurns[i&1])
						}
					case TM_Turns:
						if sys.numTurns[(i+1)&1] < sys.numTurns[i&1] && sys.lifeShare[i&1] {
							lm = lm * float32(sys.numTurns[(i+1)&1]) / float32(sys.numTurns[i&1])
						}
					}
				}
			}
			foo := math.Pow(f.lvmul, float64(-f.level[i]))
			p[0].lifeMax = Max(1, int32(math.Floor(foo*float64(lm))))

			if p[0].roundsExisted() > 0 {
				/* If character already existed for a round, presumably because of turns mode, just update life */
				p[0].life = Min(p[0].lifeMax, int32(math.Ceil(foo*float64(p[0].life))))
			} else if sys.round == 1 || sys.tmode[i&1] == TM_Turns {
				/* If round 1 or a new character in turns mode, initialize values */
				if p[0].ocd().life != -1 {
					p[0].life = p[0].ocd().life
				} else {
					p[0].life = p[0].lifeMax
				}
				if sys.round == 1 {
					if sys.maxPowerMode {
						p[0].power = p[0].powerMax
					} else if p[0].ocd().power != -1 {
						p[0].power = p[0].ocd().power
					} else {
						p[0].power = 0
					}
				}
				p[0].dialogue = []string{}
				p[0].mapArray = make(map[string]float32)
				for k, v := range p[0].mapDefault {
					p[0].mapArray[k] = v
				}
				p[0].remapSpr = make(RemapPreset)

				// Reset hitScale
				p[0].defaultHitScale = newHitScaleArray()
				p[0].activeHitScale = make(map[int32][3]*HitScale)
				p[0].nextHitScale = make(map[int32][3]*HitScale)
			}

			if p[0].ocd().guardPoints != -1 {
				p[0].guardPoints = p[0].ocd().guardPoints
			} else {
				p[0].guardPoints = p[0].guardPointsMax
			}
			if p[0].ocd().dizzyPoints != -1 {
				p[0].dizzyPoints = p[0].ocd().dizzyPoints
			} else {
				p[0].dizzyPoints = p[0].dizzyPointsMax
			}
			p[0].redLife = 0
			f.copyVar(i)
		}
	}
}
func (f *Fight) initSuperMeter() {
	for i, p := range sys.chars {
		if len(p) > 0 {
			p[0].clear2()
			f.level[i] = sys.wincnt.getLevel(i)
			if sys.powerShare[i&1] && p[0].teamside != -1 {
				pmax := Max(sys.cgi[i&1].data.power, sys.cgi[i].data.power)
				for j := i & 1; j < MaxSimul*2; j += 2 {
					if len(sys.chars[j]) > 0 {
						sys.chars[j][0].powerMax = pmax
					}
				}
			}
		}
	}
}

func (f *Fight) initTeamsLevels() {
	minlv, maxlv := f.level[0], f.level[0]
	for i, lv := range f.level[1:] {
		if len(sys.chars[i+1]) > 0 {
			minlv = Min(minlv, lv)
			maxlv = Max(maxlv, lv)
		}
	}
	if minlv > 0 {
		for i := range f.level {
			f.level[i] -= minlv
		}
	} else if maxlv < 0 {
		for i := range f.level {
			f.level[i] -= maxlv
		}
	}
}
func NewFight() Fight {
	f := Fight{}
	f.oldStageVars.copyStageVars(sys.stage)
	f.life = make([]int32, len(sys.chars))
	f.pow = make([]int32, len(sys.chars))
	f.gpow = make([]int32, len(sys.chars))
	f.spow = make([]int32, len(sys.chars))
	f.rlife = make([]int32, len(sys.chars))
	f.ivar = make([][]int32, len(sys.chars))
	f.fvar = make([][]float32, len(sys.chars))
	f.dialogue = make([][]string, len(sys.chars))
	f.mapArray = make([]map[string]float32, len(sys.chars))
	f.remapSpr = make([]RemapPreset, len(sys.chars))
	f.level = make([]int32, len(sys.chars))
	return f
}
func readI32(b []byte) int32 {
	if len(b) < 4 {
		return 0
	}
	//fmt.Printf("b[0] %d b[1] %d b[2] %d b[3] %d\n", b[0], b[1], b[2], b[3])
	return int32(b[0]) | int32(b[1])<<8 | int32(b[2])<<16 | int32(b[3])<<24
}

func decodeInputs(buffer [][]byte) []InputBits {
	var inputs = make([]InputBits, len(buffer))
	for i, b := range buffer {
		inputs[i] = InputBits(readI32(b))
	}
	return inputs
}

// HACK: So you won't be playing eachothers characters
func reverseInputs(inputs []InputBits) []InputBits {
	for i, j := 0, len(inputs)-1; i < j; i, j = i+1, j-1 {
		inputs[i], inputs[j] = inputs[j], inputs[i]
	}
	return inputs
}

func writeI32(i32 int32) []byte {
	b := []byte{byte(i32), byte(i32 >> 8), byte(i32 >> 16), byte(i32 >> 24)}
	return b
}

func getInputs(player int) []byte {
	var ib InputBits
	ib.SetInput(player)
	return writeI32(int32(ib))
}

func (rs *RollbackSystem) roundState(s *System) int32 {
	switch {
	case s.postMatchFlg:
		return -1
	case s.intro > s.lifebar.ro.ctrl_time+1:
		return 0
	case s.lifebar.ro.cur == 0:
		return 1
	case s.intro >= 0 || s.finish == FT_NotYet:
		return 2
	case s.intro < -(s.lifebar.ro.over_hittime +
		s.lifebar.ro.over_waittime):
		return 4
	default:
		return 3
	}
}
