package main

import (
	"arena"
	"fmt"
	"hash/fnv"
	"strconv"
	"sync"
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

const MaxSaveStates = 8

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
	enableZoomtime          int32
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
	fight        Fight
	introSkipped bool
	preFightTime int32
	debugWC      *Char

	commandLists []*CommandList
	luaTables    []*lua.LTable
}

func NewGameState() *GameState {
	return &GameState{
		id: int(time.Now().UnixMilli()),
	}
}

func (gs *GameState) LoadState(stateID int) {
	sys.arenaLoadMap[stateID] = arena.NewArena()
	a := sys.arenaLoadMap[stateID]
	gsp := &sys.loadPool

	sys.randseed = gs.randseed
	sys.time = gs.Time // UIT
	sys.gameTime = gs.GameTime
	gs.loadCharData(a, gsp)
	gs.loadExplodData(a)
	sys.cam = gs.cam
	gs.loadPauseData()
	gs.loadSuperData(a)
	gs.loadPalFX(a)
	gs.loadProjectileData(a)
	sys.com = gs.com
	sys.envShake = gs.envShake
	sys.envcol_time = gs.envcol_time
	sys.specialFlag = gs.specialFlag
	sys.envcol = gs.envcol

	sys.bcStack = arena.MakeSlice[BytecodeValue](a, len(gs.bcStack), len(gs.bcStack))
	copy(sys.bcStack, gs.bcStack)
	sys.bcVarStack = arena.MakeSlice[BytecodeValue](a, len(gs.bcVarStack), len(gs.bcVarStack))
	copy(sys.bcVarStack, gs.bcVarStack)
	sys.bcVar = arena.MakeSlice[BytecodeValue](a, len(gs.bcVar), len(gs.bcVar))
	copy(sys.bcVar, gs.bcVar)

	sys.stage = gs.stage.Clone(a, gsp)

	sys.aiInput = gs.aiInput
	sys.inputRemap = gs.inputRemap
	sys.autoguard = gs.autoguard

	sys.workBe = arena.MakeSlice[BytecodeExp](a, len(gs.workBe), len(gs.workBe))
	for i := 0; i < len(gs.workBe); i++ {
		sys.workBe[i] = arena.MakeSlice[OpCode](a, len(gs.workBe[i]), len(gs.workBe[i]))
		copy(sys.workBe[i], gs.workBe[i])
	}

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

	sys.drawc1 = arena.MakeSlice[[4]float32](a, len(gs.drawc1), len(gs.drawc1))
	copy(sys.drawc1, gs.drawc1)
	sys.drawc2 = arena.MakeSlice[[4]float32](a, len(gs.drawc2), len(gs.drawc2))
	copy(sys.drawc2, gs.drawc2)
	sys.drawc2sp = arena.MakeSlice[[4]float32](a, len(gs.drawc2sp), len(gs.drawc2sp))
	copy(sys.drawc2sp, gs.drawc2sp)
	sys.drawc2mtk = arena.MakeSlice[[4]float32](a, len(gs.drawc2mtk), len(gs.drawc2mtk))
	copy(sys.drawc2mtk, gs.drawc2mtk)
	sys.drawwh = arena.MakeSlice[[4]float32](a, len(gs.drawwh), len(gs.drawwh))
	copy(sys.drawwh, gs.drawwh)

	sys.accel = gs.accel
	sys.clsnDraw = gs.clsnDraw
	//sys.statusDraw = gs.statusDraw
	sys.explodMax = gs.explodMax

	// Things that directly or indirectly get put into CGO can't go into arenas
	sys.workpal = make([]uint32, len(gs.workpal)) //arena.MakeSlice[uint32](a, len(gs.workpal), len(gs.workpal))
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
	sys.enableZoomtime = gs.enableZoomtime
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
		ws := gs.workingStateState.Clone(a)
		sys.workingState = &ws
		// }
	}

	sys.lifebar = gs.lifebar.Clone(a)

	sys.cgi = gs.cgi

	sys.timerStart = gs.timerStart

	sys.timerRounds = arena.MakeSlice[int32](a, len(gs.timerRounds), len(gs.timerRounds))
	copy(sys.timerRounds, gs.timerRounds)

	sys.teamLeader = gs.teamLeader
	sys.postMatchFlg = gs.postMatchFlg
	sys.scoreStart = gs.scoreStart

	sys.scoreRounds = arena.MakeSlice[[2]float32](a, len(gs.scoreRounds), len(gs.scoreRounds))
	copy(sys.scoreRounds, gs.scoreRounds)

	sys.roundType = gs.roundType

	sys.sel = gs.sel.Clone(a)
	for i := 0; i < len(sys.stringPool); i++ {
		sys.stringPool[i] = gs.stringPool[i].Clone(a, gsp)
	}

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

	sys.timerCount = arena.MakeSlice[int32](a, len(gs.timerCount), len(gs.timerCount))
	copy(sys.timerCount, gs.timerCount)
	sys.commonLua = arena.MakeSlice[string](a, len(gs.commonLua), len(gs.commonLua))
	copy(sys.commonLua, gs.commonLua)
	sys.commonStates = arena.MakeSlice[string](a, len(gs.commonStates), len(gs.commonStates))
	copy(sys.commonStates, gs.commonStates)

	sys.endMatch = gs.endMatch

	// theoretically this shouldn't do anything.
	sys.matchData = gs.cloneLuaTable(gs.matchData)

	sys.noSoundFlg = gs.noSoundFlg
	sys.loseSimul = gs.loseSimul
	sys.loseTag = gs.loseTag
	sys.continueFlg = gs.continueFlg
	sys.stageLoopNo = gs.stageLoopNo

	// 11/5/22

	wc := gs.debugWC.Clone(a, gsp)
	sys.debugWC = &wc

	// gotta keep these pointers around because they are userdata
	for i := 0; i < len(sys.commandLists); i++ {
		gs.commandLists[i].CopyTo(sys.commandLists[i], a)
	}

	// sys.luaTables = gs.luaTables

	// This won't be around if we aren't in a proper rollback session.
	if sys.rollback.session != nil {
		sys.rollback.currentFight = gs.fight.Clone(a, gsp)
	}

	sys.introSkipped = gs.introSkipped

	sys.preFightTime = gs.preFightTime
}

func (gs *GameState) SaveState(stateID int) {
	sys.arenaSaveMap[stateID] = arena.NewArena()
	a := sys.arenaSaveMap[stateID]
	gsp := &sys.savePool

	gs.cgi = sys.cgi
	gs.saved = true
	gs.frame = sys.frameCounter
	gs.randseed = sys.randseed
	gs.Time = sys.time
	gs.GameTime = sys.gameTime

	gs.saveCharData(a, gsp)
	gs.saveExplodData(a)
	gs.cam = sys.cam
	gs.savePauseData()
	gs.saveSuperData(a)
	gs.savePalFX(a)
	gs.saveProjectileData(a)

	gs.com = sys.com
	gs.envShake = sys.envShake
	gs.envcol_time = sys.envcol_time
	gs.specialFlag = sys.specialFlag
	gs.envcol = sys.envcol

	gs.bcStack = arena.MakeSlice[BytecodeValue](a, len(sys.bcStack), len(sys.bcStack))
	copy(gs.bcStack, sys.bcStack)
	gs.bcVarStack = arena.MakeSlice[BytecodeValue](a, len(sys.bcVarStack), len(sys.bcVarStack))
	copy(gs.bcVarStack, sys.bcVarStack)
	gs.bcVar = arena.MakeSlice[BytecodeValue](a, len(sys.bcVar), len(sys.bcVar))
	copy(gs.bcVar, sys.bcVar)

	gs.stage = sys.stage.Clone(a, gsp)

	gs.aiInput = sys.aiInput
	gs.inputRemap = sys.inputRemap
	gs.autoguard = sys.autoguard
	gs.workBe = arena.MakeSlice[BytecodeExp](a, len(sys.workBe), len(sys.workBe))
	for i := 0; i < len(sys.workBe); i++ {
		gs.workBe[i] = arena.MakeSlice[OpCode](a, len(sys.workBe[i]), len(sys.workBe[i]))
		copy(gs.workBe[i], sys.workBe[i])
	}

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

	gs.drawc1 = arena.MakeSlice[[4]float32](a, len(sys.drawc1), len(sys.drawc1))
	copy(gs.drawc1, sys.drawc1)
	gs.drawc2 = arena.MakeSlice[[4]float32](a, len(sys.drawc2), len(sys.drawc2))
	copy(gs.drawc2, sys.drawc2)
	gs.drawc2sp = arena.MakeSlice[[4]float32](a, len(sys.drawc2sp), len(sys.drawc2sp))
	copy(gs.drawc2sp, sys.drawc2sp)
	gs.drawc2mtk = arena.MakeSlice[[4]float32](a, len(sys.drawc2mtk), len(sys.drawc2mtk))
	copy(gs.drawc2mtk, sys.drawc2mtk)
	gs.drawwh = arena.MakeSlice[[4]float32](a, len(sys.drawwh), len(sys.drawwh))
	copy(gs.drawwh, sys.drawwh)

	gs.accel = sys.accel
	gs.clsnDraw = sys.clsnDraw
	gs.statusDraw = sys.statusDraw
	gs.explodMax = sys.explodMax

	// Things that directly or indirectly get put into CGO can't go into arenas
	gs.workpal = make([]uint32, len(sys.workpal)) //arena.MakeSlice[uint32](a, len(sys.workpal), len(sys.workpal))
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
	gs.enableZoomtime = sys.enableZoomtime
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
		gs.workingStateState = sys.workingState.Clone(a)
	}

	gs.lifebar = sys.lifebar.Clone(a)

	gs.timerStart = sys.timerStart
	gs.timerRounds = arena.MakeSlice[int32](a, len(sys.timerRounds), len(sys.timerRounds))
	copy(gs.timerRounds, sys.timerRounds)
	gs.teamLeader = sys.teamLeader
	gs.postMatchFlg = sys.postMatchFlg
	gs.scoreStart = sys.scoreStart
	gs.scoreRounds = arena.MakeSlice[[2]float32](a, len(sys.scoreRounds), len(sys.scoreRounds))
	copy(gs.scoreRounds, sys.scoreRounds)
	gs.roundType = sys.roundType
	gs.sel = sys.sel.Clone(a)
	for i := 0; i < len(sys.stringPool); i++ {
		gs.stringPool[i] = sys.stringPool[i].Clone(a, gsp)
	}

	gs.dialogueFlg = sys.dialogueFlg
	gs.gameMode = sys.gameMode
	gs.consecutiveWins = sys.consecutiveWins

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

	gs.timerCount = arena.MakeSlice[int32](a, len(sys.timerCount), len(sys.timerCount))
	copy(gs.timerCount, sys.timerCount)
	gs.commonLua = arena.MakeSlice[string](a, len(sys.commonLua), len(sys.commonLua))
	copy(gs.commonLua, sys.commonLua)
	gs.commonStates = arena.MakeSlice[string](a, len(sys.commonStates), len(sys.commonStates))
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

	debugWC := sys.debugWC.Clone(a, gsp)
	gs.debugWC = &debugWC
	gs.commandLists = arena.MakeSlice[*CommandList](a, len(sys.commandLists), len(sys.commandLists))
	for i := 0; i < len(sys.commandLists); i++ {
		cl := sys.commandLists[i].Clone(a)
		gs.commandLists[i] = &cl
	}
	gs.luaTables = arena.MakeSlice[*lua.LTable](a, len(sys.luaTables), len(sys.luaTables))
	for i := 0; i < len(sys.luaTables); i++ {
		gs.luaTables[i] = gs.cloneLuaTable(sys.luaTables[i])
	}

	// This won't be around if we aren't in a proper rollback session.
	if sys.rollback.session != nil {
		gs.fight = sys.rollback.currentFight.Clone(a, gsp)
	}

	gs.introSkipped = sys.introSkipped
	gs.preFightTime = sys.preFightTime
}

func (gs *GameState) cloneLuaTable(s *lua.LTable) *lua.LTable {
	tbl := sys.luaLState.NewTable()
	s.ForEach(func(key lua.LValue, value lua.LValue) {
		switch value.Type() {
		case lua.LTTable:
			innerTbl := value.(*lua.LTable)
			tbl.RawSet(key, gs.cloneLuaTable(innerTbl))
		default:
			tbl.RawSet(key, value)
		}
	})
	return tbl
}

func (src *CommandList) CopyTo(dst *CommandList, a *arena.Arena) {
	clone := src.Clone(a)
	*dst = clone
}

func (gs *GameState) savePalFX(a *arena.Arena) {
	gs.allPalFX = sys.allPalFX.Clone(a)
	gs.bgPalFX = sys.bgPalFX.Clone(a)
}

func (gs *GameState) saveCharData(a *arena.Arena, gsp *GameStatePool) {
	for i := range sys.chars {
		gs.charData[i] = arena.MakeSlice[Char](a, len(sys.chars[i]), len(sys.chars[i]))
		gs.chars[i] = arena.MakeSlice[*Char](a, len(sys.chars[i]), len(sys.chars[i]))
		for j, c := range sys.chars[i] {
			gs.charData[i][j] = c.Clone(a, gsp)
			gs.chars[i][j] = c
		}
	}

	for i := range gs.chars {
		for _, c := range gs.chars[i] {
			if !c.keyctrl[0] {
				c.cmd = gs.chars[c.playerNo][0].cmd
			}
		}
	}

	if sys.workingChar != nil {
		c := sys.workingChar.Clone(a, gsp)
		gs.workingChar = &c
	} else {
		gs.workingChar = sys.workingChar
	}

	gs.charList = sys.charList.Clone(a, gsp)

}

func (gs *GameState) saveProjectileData(a *arena.Arena) {
	for i := range sys.projs {
		gs.projs[i] = arena.MakeSlice[Projectile](a, len(sys.projs[i]), len(sys.projs[i]))
		for j := 0; j < len(sys.projs[i]); j++ {
			gs.projs[i][j] = sys.projs[i][j].clone(a)
		}
	}
}

func (gs *GameState) saveSuperData(a *arena.Arena) {
	gs.super = sys.super
	gs.supertime = sys.supertime
	gs.superpausebg = sys.superpausebg
	gs.superendcmdbuftime = sys.superendcmdbuftime
	gs.superplayer = sys.superplayer
	gs.superdarken = sys.superdarken
	if sys.superanim != nil {
		gs.superanim = sys.superanim.Clone(a)
	} else {
		gs.superanim = sys.superanim
	}
	gs.superpmap = sys.superpmap.Clone(a)
	gs.superpos = [2]float32{sys.superpos[0], sys.superpos[1]}
	gs.superfacing = sys.superfacing
	gs.superp2defmul = sys.superp2defmul
}

func (gs *GameState) savePauseData() {
	gs.pause = sys.pause
	gs.pausetime = sys.pausetime
	gs.pausebg = sys.pausebg
	gs.pauseendcmdbuftime = sys.pauseendcmdbuftime
	gs.pauseplayer = sys.pauseplayer
}

func (gs *GameState) saveExplodData(a *arena.Arena) {
	for i := range sys.explods {
		gs.explods[i] = arena.MakeSlice[Explod](a, len(sys.explods[i]), len(sys.explods[i]))
		for j := 0; j < len(sys.explods[i]); j++ {
			gs.explods[i][j] = *sys.explods[i][j].Clone(a)
		}
	}
	for i := range sys.explDrawlist {
		gs.explDrawlist[i] = arena.MakeSlice[int](a, len(sys.explDrawlist[i]), len(sys.explDrawlist[i]))
		copy(gs.explDrawlist[i], sys.explDrawlist[i])
	}

	for i := range sys.topexplDrawlist {
		gs.topexplDrawlist[i] = arena.MakeSlice[int](a, len(sys.topexplDrawlist[i]), len(sys.topexplDrawlist[i]))
		copy(gs.topexplDrawlist[i], sys.topexplDrawlist[i])
	}

	for i := range sys.underexplDrawlist {
		gs.underexplDrawlist[i] = arena.MakeSlice[int](a, len(sys.underexplDrawlist[i]), len(sys.underexplDrawlist[i]))
		copy(gs.underexplDrawlist[i], sys.underexplDrawlist[i])
	}
}

func (gs *GameState) loadPalFX(a *arena.Arena) {
	sys.allPalFX = gs.allPalFX.Clone(a)
	sys.bgPalFX = gs.bgPalFX.Clone(a)
}
func (gs *GameState) loadCharData(a *arena.Arena, gsp *GameStatePool) {
	for i := 0; i < len(sys.chars); i++ {
		sys.chars[i] = arena.MakeSlice[*Char](a, len(gs.chars[i]), len(gs.chars[i]))
		copy(sys.chars[i], gs.chars[i])
	}

	for i := 0; i < len(sys.chars); i++ {
		for j := 0; j < len(sys.chars[i]); j++ {
			*sys.chars[i][j] = gs.charData[i][j].Clone(a, gsp)
		}
	}

	for i := range sys.chars {
		for _, c := range sys.chars[i] {
			if !c.keyctrl[0] {
				c.cmd = sys.chars[c.playerNo][0].cmd
			}
		}
	}

	if gs.workingChar != nil {
		wc := gs.workingChar.Clone(a, gsp)
		sys.workingChar = &wc
	} else {
		sys.workingChar = gs.workingChar
	}

	sys.charList = gs.charList.Clone(a, gsp)
}

func (gs *GameState) loadSuperData(a *arena.Arena) {
	sys.super = gs.super
	sys.supertime = gs.supertime
	sys.superpausebg = gs.superpausebg
	sys.superendcmdbuftime = gs.superendcmdbuftime
	sys.superplayer = gs.superplayer
	sys.superdarken = gs.superdarken
	if gs.superanim != nil {
		sys.superanim = gs.superanim.Clone(a)
	} else {
		sys.superanim = gs.superanim
	}
	sys.superpmap = gs.superpmap.Clone(a)
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

func (gs *GameState) loadExplodData(a *arena.Arena) {
	for i := range gs.explods {
		sys.explods[i] = arena.MakeSlice[Explod](a, len(gs.explods[i]), len(gs.explods[i]))
		for j := 0; j < len(gs.explods[i]); j++ {
			sys.explods[i][j] = *gs.explods[i][j].Clone(a)
		}
	}

	for i := range gs.explDrawlist {
		sys.explDrawlist[i] = arena.MakeSlice[int](a, len(gs.explDrawlist[i]), len(gs.explDrawlist[i]))
		copy(sys.explDrawlist[i], gs.explDrawlist[i])
	}

	for i := range gs.topexplDrawlist {
		sys.topexplDrawlist[i] = arena.MakeSlice[int](a, len(gs.topexplDrawlist[i]), len(gs.topexplDrawlist[i]))
		copy(sys.topexplDrawlist[i], gs.topexplDrawlist[i])
	}

	for i := range gs.underexplDrawlist {
		sys.underexplDrawlist[i] = arena.MakeSlice[int](a, len(gs.underexplDrawlist[i]), len(gs.underexplDrawlist[i]))
		copy(sys.underexplDrawlist[i], gs.underexplDrawlist[i])
	}
}

func (gs *GameState) loadProjectileData(a *arena.Arena) {
	for i := range gs.projs {
		sys.projs[i] = arena.MakeSlice[Projectile](a, len(gs.projs[i]), len(gs.projs[i]))
		for j := range gs.projs[i] {
			sys.projs[i][j] = gs.projs[i][j].clone(a)
		}
	}
}

func (gsp *GameStatePool) Get(item interface{}) (result interface{}) {
	objs, ok := gsp.poolObjs[gsp.curStateID]
	if !ok {
		gsp.poolObjs[gsp.curStateID] = make([]interface{}, 0, 50)
		objs = gsp.poolObjs[gsp.curStateID]
	}

	switch item.(type) {
	case (map[int32][3]*HitScale):
		objs = append(objs, gsp.hitscaleMapPool.Get())
		return objs[len(objs)-1]
	case (map[string]float32):
		objs = append(objs, gsp.stringFloat32MapPool.Get())
		return objs[len(objs)-1]
	case (map[string]int):
		objs = append(objs, gsp.stringIntMapPool.Get())
		return objs[len(objs)-1]
	case (AnimationTable):
		objs = append(objs, gsp.animationTablePool.Get())
		return objs[len(objs)-1]
	case (map[int32]*Char):
		objs = append(objs, gsp.int32CharPointerMapPool.Get())
		return objs[len(objs)-1]
	default:
		return nil
	}
}

func (gsp *GameStatePool) Put(item interface{}) {
	switch item.(type) {
	case (*map[int32][3]*HitScale):
		gsp.hitscaleMapPool.Put(item)
	case (*map[string]float32):
		gsp.stringFloat32MapPool.Put(item)
	case (*map[string]int):
		gsp.stringIntMapPool.Put(item)
	case (*AnimationTable):
		gsp.animationTablePool.Put(item)
	case (*map[int32]*Char):
		gsp.int32CharPointerMapPool.Put(item)
	default:
	}
}

func (gsp *GameStatePool) Free(stateID int) {
	objs, ok := gsp.poolObjs[stateID]
	if ok {
		for i := 0; i < len(objs); i++ {
			gsp.Put(objs[i])
		}
	}
	delete(gsp.poolObjs, stateID)
}

func NewGameStatePool() GameStatePool {
	return GameStatePool{
		gameStatePool: sync.Pool{
			New: func() interface{} {
				return NewGameState()
			},
		},
		stringIntMapPool: sync.Pool{
			New: func() interface{} {
				si := make(map[string]int)
				return &si
			},
		},
		hitscaleMapPool: sync.Pool{
			New: func() interface{} {
				hs := make(map[int32][3]*HitScale)
				return &hs
			},
		},
		stringFloat32MapPool: sync.Pool{
			New: func() interface{} {
				sf := make(map[string]float32)
				return &sf
			},
		},
		animationTablePool: sync.Pool{
			New: func() interface{} {
				at := make(AnimationTable)
				return &at
			},
		},
		int32CharPointerMapPool: sync.Pool{
			New: func() interface{} {
				ic := make(map[int32]*Char)
				return &ic
			},
		},
		poolObjs: make(map[int][]interface{}),
	}
}

func PreAllocHitScale() [3]*HitScale {
	h := [3]*HitScale{}
	for i := 0; i < len(h); i++ {
		h[i] = &HitScale{}
	}
	return h
}

type GameStatePool struct {
	gameStatePool           sync.Pool
	stringIntMapPool        sync.Pool
	hitscaleMapPool         sync.Pool
	stringFloat32MapPool    sync.Pool
	animationTablePool      sync.Pool
	mapArraySlicePool       sync.Pool
	int32CharPointerMapPool sync.Pool
	poolObjs                map[int][]interface{}
	curStateID              int
}
