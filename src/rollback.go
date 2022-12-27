package main

import "math"

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
