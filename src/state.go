package main

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"time"

	glfw "github.com/fyne-io/glfw-js"
	lua "github.com/yuin/gopher-lua"
)

func (cs Char) String() string {
	str := fmt.Sprintf(`Char %s 
	RedLife             :%d 
	Juggle              :%d 
	Life                :%d 
	Key                 :%d  
	Localcoord          :%f 
	Localscl            :%f 
	Pos                 :%v 
	DrawPos             :%v 
	OldPos              :%v 
	Vel                 :%v  
	Facing              :%f
	Id                  :%d
	HelperId            :%d
	HelperIndex         :%d
	ParentIndex         :%d
	PlayerNo            :%d
	Teamside            :%d
	AnimPN              :%d
	AnimNo              :%d
	LifeMax             :%d
	PowerMax            :%d
	DizzyPoints         :%d
	GuardPoints         :%d
	FallTime            :%d
	ClsnScale           :%v
	HoIdx               :%d
	Mctime              :%d
	Targets             :%v
	TargetsOfHitdef     :%v
	Atktmp              :%d
	Hittmp              :%d
	Acttmp              :%d
	Minus               :%d
	GroundAngle          :%f
	ComboExtraFrameWindow :%d
	InheritJuggle         :%d
	Preserve              :%d
	Ivar            :%v
	Fvar            :%v
	Offset          :%v`,
		cs.name, cs.redLife, cs.juggle, cs.life, cs.key, cs.localcoord,
		cs.localscl, cs.pos, cs.drawPos, cs.oldPos, cs.vel, cs.facing,
		cs.id, cs.helperId, cs.helperIndex, cs.parentIndex, cs.playerNo,
		cs.teamside, cs.animPN, cs.animNo, cs.lifeMax, cs.powerMax, cs.dizzyPoints,
		cs.guardPoints, cs.fallTime, cs.clsnScale, cs.hoIdx, cs.mctime, cs.targets, cs.targetsOfHitdef,
		cs.atktmp, cs.hittmp, cs.acttmp, cs.minus, cs.groundAngle, cs.comboExtraFrameWindow, cs.inheritJuggle,
		cs.preserve, cs.ivar, cs.fvar, cs.offset)
	str += fmt.Sprintf("\nChildren of %s:", cs.name)
	if len(cs.children) == 0 {
		str += "None\n"
	} else {
		str += "{ \n"
		for i := 0; i < len(cs.children); i++ {
			if cs.children[i] != nil {
				str += cs.children[i].String()
			} else {
				str += "Nil Child"
			}
			str += "\n"
		}
		str += "}\n"

	}
	str += fmt.Sprintf("EnemyNear of %s:", cs.name)
	if len(cs.enemynear[0]) == 0 && len(cs.enemynear[1]) == 0 {
		str += "None\n"
	} else {
		str += "{ \n "
		for i := 0; i < len(cs.enemynear); i++ {
			for j := 0; j < len(cs.enemynear[i]); j++ {
				str += cs.enemynear[i][j].String()
				str += "\n"
			}
		}
		str += "}\n"

	}
	return str
}

func (gs *GameState) getID() string {
	return strconv.Itoa(int(gs.id))
}

func (gs *GameState) Checksum() int {
	//	buf := bytes.Buffer{}
	//	enc := gob.NewEncoder(&buf)
	//	err := enc.Encode(gs)
	//	if err != nil {
	//		panic(err)
	//	}
	//	gs.bytes = buf.Bytes()
	gs.bytes = []byte(gs.String())
	h := fnv.New32a()
	h.Write(gs.bytes)
	return int(h.Sum32())
}

func (gs *GameState) String() (str string) {
	str = fmt.Sprintf("Time: %d GameTime %d \n", gs.Time, gs.GameTime)
	str += fmt.Sprintf("bcStack: %v\n", gs.bcStack)
	str += fmt.Sprintf("bcVarStack: %v\n", gs.bcVarStack)
	str += fmt.Sprintf("bcVar: %v\n", gs.bcVar)
	str += fmt.Sprintf("workBe: %v\n", gs.workBe)
	for i := 0; i < len(gs.charData); i++ {
		for j := 0; j < len(gs.charData[i]); j++ {
			str += gs.charData[i][j].String()
			str += "\n"
		}
	}
	return
}

type GameState struct {
	bytes             []byte
	id                int
	saved             bool
	frame             int32
	randseed          int32
	Time              int32
	GameTime          int32
	projs             [MaxSimul*2 + MaxAttachedChar][]Projectile
	chars             [MaxSimul*2 + MaxAttachedChar][]*Char
	charData          [MaxSimul*2 + MaxAttachedChar][]Char
	explods           [MaxSimul*2 + MaxAttachedChar][]Explod
	explDrawlist      [MaxSimul*2 + MaxAttachedChar][]int
	topexplDrawlist   [MaxSimul*2 + MaxAttachedChar][]int
	underexplDrawlist [MaxSimul*2 + MaxAttachedChar][]int
	aiInput           [MaxSimul*2 + MaxAttachedChar]AiInput
	inputRemap        [MaxSimul*2 + MaxAttachedChar]int
	autoguard         [MaxSimul*2 + MaxAttachedChar]bool
	charList          CharList

	com                [MaxSimul*2 + MaxAttachedChar]float32 // UIT
	cam                Camera
	allPalFX           PalFX
	bgPalFX            PalFX
	pause              int32
	pausetime          int32
	pausebg            bool
	pauseendcmdbuftime int32
	pauseplayer        int
	super              int32
	supertime          int32
	superpausebg       bool
	superendcmdbuftime int32
	superplayer        int
	superdarken        bool
	superanim          *Animation
	superanimRef       *Animation
	superpmap          PalFX
	superpos           [2]float32
	superfacing        float32
	superp2defmul      float32

	envShake            EnvShake
	specialFlag         GlobalSpecialFlag // UIT
	envcol              [3]int32
	envcol_time         int32
	bcStack, bcVarStack BytecodeStack
	bcVar               []BytecodeValue
	workBe              []BytecodeExp

	scrrect                 [4]int32
	gameWidth, gameHeight   int32 // UIT
	widthScale, heightScale float32
	gameEnd, frameSkip      bool
	brightness              int32
	roundTime               int32 // UIT
	lifeMul                 float32
	team1VS2Life            float32
	turnsRecoveryRate       float32
	match                   int32 // UIT
	round                   int32 // UIT
	intro                   int32
	lastHitter              [2]int
	winTeam                 int // UIT
	winType                 [2]WinType
	winTrigger              [2]WinType // UIT
	matchWins, wins         [2]int32   // UIT
	roundsExisted           [2]int32
	draws                   int32
	tmode                   [2]TeamMode // UIT
	numSimul, numTurns      [2]int32    // UIT
	esc                     bool
	workingChar             *Char
	workingStateState       StateBytecode // UIT
	afterImageMax           int32
	comboExtraFrameWindow   int32
	envcol_under            bool
	helperMax               int32
	nextCharId              int32
	powerShare              [2]bool
	tickCount               int
	oldTickCount            int
	tickCountF              float32
	lastTick                float32
	nextAddTime             float32
	oldNextAddTime          float32
	screenleft              float32
	screenright             float32
	xmin, xmax              float32
	winskipped              bool
	paused, step            bool
	roundResetFlg           bool
	reloadFlg               bool
	reloadStageFlg          bool
	reloadLifebarFlg        bool
	reloadCharSlot          [MaxSimul*2 + MaxAttachedChar]bool
	turbo                   float32
	drawScale               float32
	zoomlag                 float32
	zoomScale               float32
	zoomPosXLag             float32
	zoomPosYLag             float32
	enableZoomstate         bool
	zoomCameraBound         bool
	zoomPos                 [2]float32
	finish                  FinishType // UIT
	waitdown                int32
	slowtime                int32
	shuttertime             int32
	fadeintime              int32
	fadeouttime             int32
	changeStateNest         int32
	drawc1                  ClsnRect
	drawc2                  ClsnRect
	drawc2sp                ClsnRect
	drawc2mtk               ClsnRect
	drawwh                  ClsnRect
	accel                   float32
	clsnDraw                bool
	statusDraw              bool
	explodMax               int
	workpal                 []uint32
	playerProjectileMax     int
	nomusic                 bool
	lifeShare               [2]bool
	keyConfig               []KeyConfig
	joystickConfig          []KeyConfig
	lifebar                 Lifebar
	redrawWait              struct{ nextTime, lastDraw time.Time }
	cgi                     [MaxSimul*2 + MaxAttachedChar]CharGlobalInfo

	// New 11/04/2022 all UIT
	timerStart      int32
	timerRounds     []int32
	teamLeader      [2]int
	stage           *Stage
	postMatchFlg    bool
	scoreStart      [2]float32
	scoreRounds     [][2]float32
	roundType       [2]RoundType
	sel             Select
	stringPool      [MaxSimul*2 + MaxAttachedChar]StringPool
	dialogueFlg     bool
	gameMode        string
	consecutiveWins [2]int32
	home            int

	// Non UIT, but adding them anyway just because
	// Used in Stage.go
	stageLoop bool

	// Sound
	panningRange  float32
	stereoEffects bool
	bgmVolume     int
	audioDucking  bool
	wavVolume     int

	// ByteCode
	dialogueBarsFlg bool
	dialogueForce   int
	playBgmFlg      bool

	// Input
	keyInput  glfw.Key
	keyString string

	// LifeBar
	timerCount []int32

	// Script
	commonLua    []string
	commonStates []string
	endMatch     bool
	matchData    *lua.LTable
	noSoundFlg   bool
	loseSimul    bool
	loseTag      bool
	continueFlg  bool

	stageLoopNo int

	// 11/5/2022
	fight Fight
}

func NewGameState() *GameState {
	return &GameState{
		id: int(time.Now().UnixMilli()),
	}
}

func (gs *GameState) Equal(other GameState) (equality bool) {

	if gs.randseed != other.randseed {
		fmt.Printf("Error on randseed: %d : %d ", gs.randseed, other.randseed)
		return false
	}

	if gs.Time != other.Time {
		fmt.Println("Error on time.")
		return false
	}

	if gs.GameTime != other.GameTime {
		fmt.Println("Error on gameTime.")
		return false
	}
	return true

}

func (gs *GameState) LoadState() {
	sys.randseed = gs.randseed
	sys.time = gs.Time // UIT
	sys.gameTime = gs.GameTime
	gs.loadCharData()
	gs.loadExplodData()
	sys.cam = gs.cam
	gs.loadPauseData()
	gs.loadSuperData()
	gs.loadPalFX()
	gs.loadProjectileData()
	sys.com = gs.com
	sys.envShake = gs.envShake
	sys.envcol_time = gs.envcol_time
	sys.specialFlag = gs.specialFlag
	sys.envcol = gs.envcol
	sys.bcStack = gs.bcStack
	sys.bcVarStack = gs.bcVarStack
	sys.bcVar = gs.bcVar
	//sys.stage.loadStageState(gs.stageState)
	sys.stage = gs.stage

	sys.aiInput = gs.aiInput
	sys.inputRemap = gs.inputRemap
	sys.autoguard = gs.autoguard
	sys.workBe = gs.workBe

	sys.finish = gs.finish
	sys.winTeam = gs.winTeam
	sys.winType = gs.winType
	sys.winTrigger = gs.winTrigger
	sys.lastHitter = gs.lastHitter
	sys.waitdown = gs.waitdown
	sys.slowtime = gs.slowtime

	sys.shuttertime = gs.shuttertime
	//sys.fadeintime = gs.fadeintime
	//sys.fadeouttime = gs.fadeouttime
	sys.winskipped = gs.winskipped

	sys.intro = gs.intro
	sys.time = gs.Time
	sys.nextCharId = gs.nextCharId

	sys.scrrect = gs.scrrect
	sys.gameWidth = gs.gameWidth
	sys.gameHeight = gs.gameHeight
	sys.widthScale = gs.widthScale
	sys.heightScale = gs.heightScale
	sys.gameEnd = gs.gameEnd
	sys.frameSkip = gs.frameSkip
	sys.brightness = gs.brightness
	sys.roundTime = gs.roundTime
	sys.lifeMul = gs.lifeMul
	sys.team1VS2Life = gs.team1VS2Life
	sys.turnsRecoveryRate = gs.turnsRecoveryRate

	sys.changeStateNest = gs.changeStateNest
	copy(sys.drawc1, gs.drawc1)
	copy(sys.drawc2, gs.drawc2)
	copy(sys.drawc2sp, gs.drawc2sp)
	copy(sys.drawc2mtk, gs.drawc2mtk)
	copy(sys.drawwh, gs.drawwh)
	sys.accel = gs.accel
	sys.clsnDraw = gs.clsnDraw
	//sys.statusDraw = gs.statusDraw
	sys.explodMax = gs.explodMax
	copy(sys.workpal, gs.workpal)
	sys.playerProjectileMax = gs.playerProjectileMax
	sys.nomusic = gs.nomusic
	sys.lifeShare = gs.lifeShare

	sys.turbo = gs.turbo
	sys.drawScale = gs.drawScale
	sys.zoomlag = gs.zoomlag
	sys.zoomScale = gs.zoomScale
	sys.zoomPosXLag = gs.zoomPosXLag
	sys.zoomPosYLag = gs.zoomPosYLag
	sys.enableZoomstate = gs.enableZoomstate
	sys.zoomCameraBound = gs.zoomCameraBound
	sys.zoomPos = gs.zoomPos

	sys.reloadCharSlot = gs.reloadCharSlot
	sys.turbo = gs.turbo
	sys.drawScale = gs.drawScale
	sys.zoomlag = gs.zoomlag
	sys.zoomScale = gs.zoomScale
	sys.zoomPosXLag = gs.zoomPosXLag

	sys.matchWins = gs.matchWins
	sys.wins = gs.wins
	sys.roundsExisted = gs.roundsExisted
	sys.draws = gs.draws
	sys.tmode = gs.tmode
	sys.numSimul = gs.numSimul
	sys.numTurns = gs.numTurns
	sys.esc = gs.esc
	sys.afterImageMax = gs.afterImageMax
	sys.comboExtraFrameWindow = gs.comboExtraFrameWindow
	sys.envcol_under = gs.envcol_under
	sys.helperMax = gs.helperMax
	sys.nextCharId = gs.nextCharId
	sys.powerShare = gs.powerShare
	sys.tickCount = gs.tickCount
	sys.oldTickCount = gs.oldTickCount
	sys.tickCountF = gs.tickCountF
	sys.lastTick = gs.lastTick
	sys.nextAddTime = gs.nextAddTime
	sys.oldNextAddTime = gs.oldNextAddTime
	sys.screenleft = gs.screenleft
	sys.screenright = gs.screenright
	sys.xmin = gs.xmin
	sys.xmax = gs.xmax
	sys.winskipped = gs.winskipped
	sys.paused = gs.paused
	sys.step = gs.step
	sys.roundResetFlg = gs.roundResetFlg
	sys.reloadFlg = gs.reloadFlg
	sys.reloadStageFlg = gs.reloadStageFlg
	sys.reloadLifebarFlg = gs.reloadLifebarFlg

	sys.match = gs.match
	sys.round = gs.round

	// bug, if a prior state didn't have this
	// Did the prior state actually have a working state
	if gs.workingStateState.stateType != 0 && gs.workingStateState.moveType != 0 {
		// if sys.workingState != nil {
		// 	*sys.workingState = gs.workingStateState
		// } else {
		sys.workingState = &gs.workingStateState
		// }
	}

	// else {
	// 	sys.workingState = &gs.workingStateState
	// }

	// copy(sys.keyConfig, gs.keyConfig)
	// copy(sys.joystickConfig, gs.joystickConfig)
	//sys.redrawWait = gs.redrawWait
	sys.lifebar = gs.lifebar

	sys.cgi = gs.cgi
	// for i := range sys.cgi {
	// 	for k, v := range gs.cgi[i].states {
	// 		sys.cgi[i].states[k] = v
	// 	}
	// }

	// New 11/04/2022
	sys.timerStart = gs.timerStart
	sys.timerRounds = gs.timerRounds
	sys.teamLeader = gs.teamLeader
	sys.postMatchFlg = gs.postMatchFlg
	sys.scoreStart = gs.scoreStart
	sys.scoreRounds = gs.scoreRounds
	sys.roundType = gs.roundType
	sys.sel = gs.sel
	sys.stringPool = gs.stringPool
	sys.dialogueFlg = gs.dialogueFlg
	sys.gameMode = gs.gameMode
	sys.consecutiveWins = gs.consecutiveWins
	sys.home = gs.home

	// Not UIT
	sys.stageLoop = gs.stageLoop
	sys.panningRange = gs.panningRange
	sys.stereoEffects = gs.stereoEffects
	sys.bgmVolume = gs.bgmVolume
	sys.audioDucking = gs.audioDucking
	sys.wavVolume = gs.wavVolume
	sys.dialogueBarsFlg = gs.dialogueBarsFlg
	sys.dialogueForce = gs.dialogueForce
	sys.playBgmFlg = gs.playBgmFlg
	//sys.keyState = gs.keyState
	sys.keyInput = gs.keyInput
	sys.keyString = gs.keyString
	sys.timerCount = gs.timerCount
	sys.commonLua = gs.commonLua
	sys.commonStates = gs.commonStates
	sys.endMatch = gs.endMatch

	// theoretically this shouldn't do anything.
	*sys.matchData = *gs.matchData

	sys.noSoundFlg = gs.noSoundFlg
	sys.loseSimul = gs.loseSimul
	sys.loseTag = gs.loseTag
	sys.continueFlg = gs.continueFlg
	sys.stageLoopNo = gs.stageLoopNo

	// 11/5/22
	sys.rollback.currentFight = gs.fight
}

func (gs *GameState) SaveState() {
	gs.cgi = sys.cgi
	// for i := range sys.cgi {
	// 	gs.cgi[i].states = make(map[int32]StateBytecode)
	// 	for k, v := range gs.cgi[i].states {
	// 		gs.cgi[i].states[k] = v
	// 	}
	// }

	gs.saved = true
	gs.frame = sys.frameCounter
	gs.randseed = sys.randseed
	gs.Time = sys.time
	gs.GameTime = sys.gameTime

	//timeBefore := time.Now().UnixMilli()
	gs.saveCharData()
	//timeAfter := time.Now().UnixMilli()
	//fmt.Printf("Time to save chars: %d\n", timeAfter-timeBefore)

	//timeBefore = time.Now().UnixMilli()
	gs.saveExplodData()
	//timeAfter = time.Now().UnixMilli()
	//fmt.Printf("Time to save explod data: %d\n", timeAfter-timeBefore)

	//timeBefore = time.Now().UnixMilli()
	gs.cam = sys.cam
	gs.savePauseData()
	gs.saveSuperData()
	gs.savePalFX()
	gs.saveProjectileData()
	//timeAfter = time.Now().UnixMilli()
	//fmt.Printf("Time to save blovk A: %d\n", timeAfter-timeBefore)

	//timeBefore = time.Now().UnixMilli()
	gs.com = sys.com
	gs.envShake = sys.envShake
	gs.envcol_time = sys.envcol_time
	gs.specialFlag = sys.specialFlag
	gs.envcol = sys.envcol
	gs.bcStack = make([]BytecodeValue, len(sys.bcStack))
	copy(gs.bcStack, sys.bcStack)

	gs.bcVarStack = make([]BytecodeValue, len(sys.bcVarStack))
	copy(gs.bcVarStack, sys.bcVarStack)

	gs.bcVar = make([]BytecodeValue, len(sys.bcVar))
	copy(gs.bcVar, sys.bcVar)

	//gs.stageState = sys.stage.getStageState()
	stage := sys.stage.Clone() // UIT
	gs.stage = &stage

	gs.aiInput = sys.aiInput
	gs.inputRemap = sys.inputRemap
	gs.autoguard = sys.autoguard
	gs.workBe = make([]BytecodeExp, len(sys.workBe))
	for i := 0; i < len(sys.workBe); i++ {
		gs.workBe[i] = make(BytecodeExp, len(sys.workBe[i]))
		copy(gs.workBe[i], sys.workBe[i])
	}

	//timeAfter = time.Now().UnixMilli()
	//fmt.Printf("Time to save block B: %d\n", timeAfter-timeBefore)

	//timeBefore = time.Now().UnixMilli()
	gs.finish = sys.finish
	gs.winTeam = sys.winTeam
	gs.winType = sys.winType
	gs.winTrigger = sys.winTrigger
	gs.lastHitter = sys.lastHitter
	gs.waitdown = sys.waitdown
	gs.slowtime = sys.slowtime
	gs.shuttertime = sys.shuttertime
	gs.fadeintime = sys.fadeintime
	gs.fadeouttime = sys.fadeouttime
	gs.winskipped = sys.winskipped
	gs.intro = sys.intro
	gs.Time = sys.time
	gs.nextCharId = sys.nextCharId

	gs.scrrect = sys.scrrect
	gs.gameWidth = sys.gameWidth
	gs.gameHeight = sys.gameHeight
	gs.widthScale = sys.widthScale
	gs.heightScale = sys.heightScale
	gs.gameEnd = sys.gameEnd
	gs.frameSkip = sys.frameSkip
	gs.brightness = sys.brightness
	gs.roundTime = sys.roundTime
	gs.lifeMul = sys.lifeMul
	gs.team1VS2Life = sys.team1VS2Life
	gs.turnsRecoveryRate = sys.turnsRecoveryRate

	gs.changeStateNest = sys.changeStateNest

	gs.drawc1 = make(ClsnRect, len(sys.drawc1))
	copy(gs.drawc1, sys.drawc1)
	gs.drawc2 = make(ClsnRect, len(sys.drawc2))
	copy(gs.drawc2, sys.drawc2)
	gs.drawc2sp = make(ClsnRect, len(sys.drawc2sp))
	copy(gs.drawc2sp, sys.drawc2sp)
	gs.drawc2mtk = make(ClsnRect, len(sys.drawc2mtk))
	copy(gs.drawc2mtk, sys.drawc2mtk)
	gs.drawwh = make(ClsnRect, len(sys.drawwh))
	copy(gs.drawwh, sys.drawwh)
	//timeAfter = time.Now().UnixMilli()
	//fmt.Printf("Time to save block C: %d\n", timeAfter-timeBefore)

	//timeBefore = time.Now().UnixMilli()
	gs.accel = sys.accel
	gs.clsnDraw = sys.clsnDraw
	gs.statusDraw = sys.statusDraw
	gs.explodMax = sys.explodMax
	gs.workpal = make([]uint32, len(sys.workpal))
	copy(gs.workpal, sys.workpal)
	gs.playerProjectileMax = sys.playerProjectileMax
	gs.nomusic = sys.nomusic
	gs.lifeShare = sys.lifeShare

	gs.turbo = sys.turbo
	gs.drawScale = sys.drawScale
	gs.zoomlag = sys.zoomlag
	gs.zoomScale = sys.zoomScale
	gs.zoomPosXLag = sys.zoomPosXLag
	gs.zoomPosYLag = sys.zoomPosYLag
	gs.enableZoomstate = sys.enableZoomstate
	gs.zoomCameraBound = sys.zoomCameraBound
	gs.zoomPos = sys.zoomPos

	gs.reloadCharSlot = sys.reloadCharSlot
	gs.turbo = sys.turbo
	gs.drawScale = sys.drawScale
	gs.zoomlag = sys.zoomlag
	gs.zoomScale = sys.zoomScale
	gs.zoomPosXLag = sys.zoomPosXLag

	gs.matchWins = sys.matchWins
	gs.wins = sys.wins
	gs.roundsExisted = sys.roundsExisted
	gs.draws = sys.draws
	gs.tmode = sys.tmode
	gs.numSimul = sys.numSimul
	gs.numTurns = sys.numTurns
	gs.esc = sys.esc
	gs.afterImageMax = sys.afterImageMax
	gs.comboExtraFrameWindow = sys.comboExtraFrameWindow
	gs.envcol_under = sys.envcol_under
	gs.helperMax = sys.helperMax
	gs.nextCharId = sys.nextCharId
	gs.powerShare = sys.powerShare
	gs.tickCount = sys.tickCount
	gs.oldTickCount = sys.oldTickCount
	gs.tickCountF = sys.tickCountF
	gs.lastTick = sys.lastTick
	gs.nextAddTime = sys.nextAddTime
	gs.oldNextAddTime = sys.oldNextAddTime
	gs.screenleft = sys.screenleft
	gs.screenright = sys.screenright
	gs.xmin = sys.xmin
	gs.xmax = sys.xmax
	gs.winskipped = sys.winskipped
	gs.paused = sys.paused
	gs.step = sys.step
	gs.roundResetFlg = sys.roundResetFlg
	gs.reloadFlg = sys.reloadFlg
	gs.reloadStageFlg = sys.reloadStageFlg
	gs.reloadLifebarFlg = sys.reloadLifebarFlg

	gs.match = sys.match
	gs.round = sys.round

	// bug, if a prior state didn't have this
	if sys.workingState != nil {
		gs.workingStateState = sys.workingState.Clone()
	}

	// gs.keyConfig = make([]KeyConfig, len(sys.keyConfig))
	// copy(gs.keyConfig, sys.keyConfig)

	// gs.joystickConfig = make([]KeyConfig, len(sys.joystickConfig))
	// copy(gs.joystickConfig, sys.joystickConfig)
	//timeAfter = time.Now().UnixMilli()
	//fmt.Printf("Time to save The rest: %d\n", timeAfter-timeBefore)

	gs.lifebar = sys.lifebar.Clone()
	gs.redrawWait = sys.redrawWait

	// New 11/04/2022
	// UIT
	gs.timerStart = sys.timerStart
	gs.timerRounds = make([]int32, len(sys.timerRounds))
	copy(gs.timerRounds, sys.timerRounds)
	gs.teamLeader = sys.teamLeader
	gs.postMatchFlg = sys.postMatchFlg
	gs.scoreStart = sys.scoreStart
	gs.scoreRounds = make([][2]float32, len(sys.scoreRounds))
	copy(gs.scoreRounds, sys.scoreRounds)
	gs.roundType = sys.roundType
	gs.sel = sys.sel.Clone()
	for i := 0; i < len(sys.stringPool); i++ {
		gs.stringPool[i] = sys.stringPool[i].Clone()
	}
	gs.dialogueFlg = sys.dialogueFlg
	gs.gameMode = sys.gameMode
	gs.consecutiveWins = sys.consecutiveWins

	// Not UIT
	gs.stageLoop = sys.stageLoop
	gs.panningRange = sys.panningRange
	gs.stereoEffects = sys.stereoEffects
	gs.bgmVolume = sys.bgmVolume
	gs.audioDucking = sys.audioDucking
	gs.wavVolume = sys.wavVolume
	gs.dialogueBarsFlg = sys.dialogueBarsFlg
	gs.dialogueForce = sys.dialogueForce
	gs.playBgmFlg = sys.playBgmFlg

	gs.keyInput = sys.keyInput
	gs.keyString = sys.keyString

	gs.timerCount = make([]int32, len(sys.timerCount))
	copy(gs.timerCount, sys.timerCount)
	gs.commonLua = make([]string, len(sys.commonLua))
	copy(gs.commonLua, sys.commonLua)
	gs.commonStates = make([]string, len(sys.commonStates))
	copy(gs.commonStates, sys.commonStates)

	gs.endMatch = sys.endMatch

	// can't deep copy because its members are private
	matchData := *sys.matchData
	gs.matchData = &matchData

	gs.noSoundFlg = sys.noSoundFlg
	gs.loseSimul = sys.loseSimul
	gs.loseTag = sys.loseTag
	gs.continueFlg = sys.continueFlg
	gs.stageLoopNo = sys.stageLoopNo

	// 11/5/2022
	gs.fight = sys.rollback.currentFight.Clone()
}

func (gs *GameState) savePalFX() {
	gs.allPalFX = sys.allPalFX
	gs.bgPalFX = sys.bgPalFX
}

func (gs *GameState) saveCharData() {
	for i := range sys.chars {
		gs.charData[i] = make([]Char, len(sys.chars[i]))
		gs.chars[i] = make([]*Char, len(sys.chars[i]))
		for j, c := range sys.chars[i] {
			//timeBefore := time.Now().UnixMilli()
			gs.charData[i][j] = c.Clone()
			gs.chars[i][j] = c
			//timeAfter := time.Now().UnixMilli()
			//fmt.Printf("Time to save character %s: %d ms\n", c.name, timeAfter-timeBefore)
			//gs.charMap[gs.charState[i][j].id] = gs.charState[i][j]
		}
	}
	if sys.workingChar != nil {
		gs.workingChar = sys.workingChar
	}

	gs.charList = CharList{}
	gs.charList.runOrder = make([]*Char, len(sys.charList.runOrder))
	copy(gs.charList.runOrder, sys.charList.runOrder)

	gs.charList.drawOrder = make([]*Char, len(sys.charList.drawOrder))
	copy(gs.charList.drawOrder, sys.charList.drawOrder)

	gs.charList.idMap = make(map[int32]*Char)
	for k, v := range sys.charList.idMap {
		gs.charList.idMap[k] = v
	}

}

func (gs *GameState) saveProjectileData() {
	for i := range sys.projs {
		gs.projs[i] = make([]Projectile, len(sys.projs[i]))
		for j := 0; j < len(sys.projs[i]); j++ {
			gs.projs[i][j] = sys.projs[i][j].Clone()
		}
	}
}

func (gs *GameState) saveSuperData() {
	gs.super = sys.super
	gs.supertime = sys.supertime
	gs.superpausebg = sys.superpausebg
	gs.superendcmdbuftime = sys.superendcmdbuftime
	gs.superplayer = sys.superplayer
	gs.superdarken = sys.superdarken
	if sys.superanim != nil {
		superanim := sys.superanim.Clone()
		gs.superanim = &superanim
	}
	gs.superpmap = sys.superpmap.Clone()
	gs.superpos = [2]float32{sys.superpos[0], sys.superpos[1]}
	gs.superfacing = sys.superfacing
	gs.superp2defmul = sys.superp2defmul
}

func (gs *GameState) savePauseData() {
	gs.pause = sys.pause // UIT
	gs.pausetime = sys.pausetime
	gs.pausebg = sys.pausebg
	gs.pauseendcmdbuftime = sys.pauseendcmdbuftime
	gs.pauseplayer = sys.pauseplayer
}

func (gs *GameState) saveExplodData() {
	for i := range sys.explods {
		gs.explods[i] = make([]Explod, len(sys.explods[i]))
		for j := 0; j < len(sys.explods[i]); j++ {
			gs.explods[i][j] = sys.explods[i][j].Clone()
		}
	}
	for i := range sys.explDrawlist {
		gs.explDrawlist[i] = make([]int, len(sys.explDrawlist[i]))
		copy(gs.explDrawlist[i], sys.explDrawlist[i])
	}

	for i := range sys.topexplDrawlist {
		gs.topexplDrawlist[i] = make([]int, len(sys.topexplDrawlist[i]))
		copy(gs.topexplDrawlist[i], sys.topexplDrawlist[i])
	}

	for i := range sys.underexplDrawlist {
		gs.underexplDrawlist[i] = make([]int, len(sys.underexplDrawlist[i]))
		copy(gs.underexplDrawlist[i], sys.underexplDrawlist[i])
	}
}

func (gs *GameState) loadPalFX() {
	sys.allPalFX = gs.allPalFX
	sys.bgPalFX = gs.bgPalFX
}

func (gs *GameState) loadCharData() {
	for i := 0; i < len(sys.chars); i++ {
		sys.chars[i] = make([]*Char, len(gs.chars[i]))
		copy(sys.chars[i], gs.chars[i])
	}
	for i := 0; i < len(sys.chars); i++ {
		for j := 0; j < len(sys.chars[i]); j++ {
			*sys.chars[i][j] = gs.charData[i][j]
		}
	}
	sys.workingChar = gs.workingChar

	sys.charList.runOrder = make([]*Char, len(gs.charList.runOrder))
	copy(sys.charList.runOrder, gs.charList.runOrder)

	sys.charList.drawOrder = make([]*Char, len(gs.charList.drawOrder))
	copy(sys.charList.drawOrder, gs.charList.drawOrder)

	sys.charList.idMap = make(map[int32]*Char)
	for k, v := range gs.charList.idMap {
		sys.charList.idMap[k] = v
	}

}

func (gs *GameState) loadSuperData() {
	sys.super = gs.super // UIT
	sys.supertime = gs.supertime
	sys.superpausebg = gs.superpausebg
	sys.superendcmdbuftime = gs.superendcmdbuftime
	sys.superplayer = gs.superplayer
	sys.superdarken = gs.superdarken
	if sys.superanim != nil {
		sys.superanim = gs.superanim
	}
	sys.superpmap = gs.superpmap
	sys.superpos = [2]float32{gs.superpos[0], gs.superpos[1]}
	sys.superfacing = gs.superfacing
	sys.superp2defmul = gs.superp2defmul
}

func (gs *GameState) loadPauseData() {
	sys.pause = gs.pause
	sys.pausetime = gs.pausetime
	sys.pausebg = gs.pausebg
	sys.pauseendcmdbuftime = gs.pauseendcmdbuftime
	sys.pauseplayer = gs.pauseplayer
}

func (gs *GameState) loadExplodData() {
	for i := range gs.explods {
		sys.explods[i] = make([]Explod, len(gs.explods[i]))
		copy(sys.explods[i], gs.explods[i])
	}

	for i := range gs.explDrawlist {
		sys.explDrawlist[i] = make([]int, len(gs.explDrawlist[i]))
		copy(sys.explDrawlist[i], gs.explDrawlist[i])
	}

	for i := range gs.topexplDrawlist {
		sys.topexplDrawlist[i] = make([]int, len(gs.topexplDrawlist[i]))
		copy(sys.topexplDrawlist[i], gs.topexplDrawlist[i])
	}

	for i := range gs.underexplDrawlist {
		sys.underexplDrawlist[i] = make([]int, len(gs.underexplDrawlist[i]))
		copy(sys.underexplDrawlist[i], gs.underexplDrawlist[i])
	}
}

func (gs *GameState) projectliesPersist() bool {
	for i := 0; i < len(sys.projs); i++ {
		if len(sys.projs[i]) != len(gs.projs[i]) {
			return false
		}
		for j := 0; j < len(sys.projs[i]); j++ {
			if sys.projs[i][j].id != gs.projs[i][j].id {
				return false
			}
		}
	}
	return true
}

func (gs *GameState) loadProjectileData() {
	if gs.projectliesPersist() {
		for i := range sys.projs {
			for j := range sys.projs[i] {
				sys.projs[i][j] = gs.projs[i][j]
			}
		}
	} else {
		for i := range gs.projs {
			sys.projs[i] = make([]Projectile, len(gs.projs[i]))
			for j := range gs.projs[i] {
				sys.projs[i][j] = gs.projs[i][j]
			}
		}

	}

}
