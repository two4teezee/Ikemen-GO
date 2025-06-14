package main

import (
	"bufio"
	"fmt"
	"image"
	"io"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/ikemen-engine/glfont"
	lua "github.com/yuin/gopher-lua"
)

const (
	MaxSimul        = 4
	MaxAttachedChar = 1
	MaxPlayerNo     = MaxSimul*2 + MaxAttachedChar
)

// sys
// The only instance of a System struct.
// Do not create more than 1.
var sys = System{
	randseed:          int32(time.Now().UnixNano()),
	scrrect:           [...]int32{0, 0, 320, 240},
	gameWidth:         320,
	gameHeight:        240,
	widthScale:        1,
	heightScale:       1,
	brightness:        256,
	roundTime:         -1,
	turnsRecoveryRate: 1.0 / 300,
	soundMixer:        &beep.Mixer{},
	bgm:               *newBgm(),
	soundChannels:     newSoundChannels(16),
	allPalFX:          *newPalFX(),
	bgPalFX:           *newPalFX(),
	ffx:               make(map[string]*FightFx),
	ffxRegexp:         "^(f)|^(s)|^(go)",
	sel:               *newSelect(),
	keyState:          make(map[Key]bool),
	match:             1,
	loader:            *newLoader(),
	numSimul:          [...]int32{2, 2}, numTurns: [...]int32{2, 2},
	ignoreMostErrors: true,
	superpmap:        *newPalFX(),
	stageList:        make(map[int32]*Stage),
	wincnt:           wincntMap(make(map[string][]int32)),
	wincntFileName:   "save/autolevel.save",
	oldNextAddTime:   1,
	commandLine:      make(chan string),
	cam:              *newCamera(),
	lifebarDisplay:   true,
	mainThreadTask:   make(chan func(), 65536),
	workpal:          make([]uint32, 256),
	errLog:           log.New(NewLogWriter(), "", log.LstdFlags),
	keyInput:         KeyUnknown,
	//FLAC_FrameWait:          -1,
	luaSpriteScale:       1,
	luaPortraitScale:     1,
	lifebarScale:         1,
	lifebarPortraitScale: 1,
}

type TeamMode int32

const (
	TM_Single TeamMode = iota
	TM_Simul
	TM_Turns
	TM_Tag
	TM_LAST = TM_Tag
)

// System struct, holds most of the data that is accessed globally through the program.
type System struct {
	randseed                int32
	scrrect                 [4]int32
	gameWidth, gameHeight   int32
	widthScale, heightScale float32
	window                  *Window
	gameEnd, frameSkip      bool
	redrawWait              struct{ nextTime, lastDraw time.Time }
	brightness              int32
	roundTime               int32
	turnsRecoveryRate       float32
	debugFont               *TextSprite
	debugDisplay            bool
	debugRef                [2]int // player number, helper index
	soundMixer              *beep.Mixer
	bgm                     Bgm
	soundChannels           *SoundChannels
	allPalFX, bgPalFX       PalFX
	lifebar                 Lifebar
	cfg                     Config
	ffx                     map[string]*FightFx
	ffxRegexp               string
	sel                     Select
	keyState                map[Key]bool
	netConnection           *NetConnection
	replayFile              *ReplayFile
	aiInput                 [MaxPlayerNo]AiInput
	keyConfig               []KeyConfig
	joystickConfig          []KeyConfig
	aiLevel                 [MaxPlayerNo]float32
	autolevel               bool
	home                    int
	gameTime                int32
	match                   int32
	inputRemap              [MaxPlayerNo]int
	round                   int32
	intro                   int32
	time                    int32
	lastHitter              [2]int
	winTeam                 int
	winType                 [2]WinType
	winTrigger              [2]WinType
	matchWins, wins         [2]int32
	roundsExisted           [2]int32
	draws                   int32
	loader                  Loader
	chars                   [MaxPlayerNo][]*Char
	charList                CharList
	cgi                     [MaxPlayerNo]CharGlobalInfo
	tmode                   [2]TeamMode
	numSimul, numTurns      [2]int32
	esc                     bool
	loadMutex               sync.Mutex
	ignoreMostErrors        bool
	stringPool              [MaxPlayerNo]StringPool
	bcStack, bcVarStack     BytecodeStack
	bcVar                   []BytecodeValue
	workingChar             *Char
	workingState            *StateBytecode
	specialFlag             GlobalSpecialFlag
	envShake                EnvShake
	pausetime               int32
	pausetimebuffer         int32
	pausebg                 bool
	pauseendcmdbuftime      int32
	pauseplayer             int
	supertime               int32
	supertimebuffer         int32
	superpausebg            bool
	superendcmdbuftime      int32
	superplayerno           int
	superdarken             bool
	superanim               *Animation
	superpmap               PalFX
	superpos                [2]float32
	superscale              [2]float32
	superp2defmul           float32
	envcol                  [3]int32
	envcol_time             int32
	envcol_under            bool
	stage                   *Stage
	stageList               map[int32]*Stage
	stageLoop               bool
	stageLoopNo             int
	wireframeDisplay        bool
	nextCharId              int32
	wincnt                  wincntMap
	wincntFileName          string
	tickCount               int
	oldTickCount            int
	tickCountF              float32
	lastTick                float32
	nextAddTime             float32
	oldNextAddTime          float32
	screenleft              float32
	screenright             float32
	xmin, xmax              float32
	zmin, zmax              float32
	winskipped              bool
	paused, step            bool
	roundResetFlg           bool
	reloadFlg               bool
	reloadStageFlg          bool
	reloadLifebarFlg        bool
	reloadCharSlot          [MaxPlayerNo]bool
	shortcutScripts         map[ShortcutKey]*ShortcutScript
	turbo                   float32
	commandLine             chan string
	drawScale               float32
	zoomlag                 float32
	zoomScale               float32
	zoomPosXLag             float32
	zoomPosYLag             float32
	enableZoomtime          int32
	zoomCameraBound         bool
	zoomStageBound          bool
	zoomPos                 [2]float32
	debugWC                 *Char
	cam                     Camera
	finishType              FinishType
	waitdown                int32
	slowtime                int32
	slowtimeTrigger         int32
	shuttertime             int32
	fadeintime              int32
	fadeouttime             int32
	wintime                 int32
	projs                   [MaxPlayerNo][]Projectile
	explods                 [MaxPlayerNo][]Explod
	explodsLayerN1          [MaxPlayerNo][]int
	explodsLayer0           [MaxPlayerNo][]int
	explodsLayer1           [MaxPlayerNo][]int
	changeStateNest         int32
	spritesLayerN1          DrawList
	spritesLayerU           DrawList
	spritesLayer0           DrawList
	spritesLayer1           DrawList
	shadows                 ShadowList
	debugc1hit              ClsnRect
	debugc1rev              ClsnRect
	debugc1not              ClsnRect
	debugc2                 ClsnRect
	debugc2hb               ClsnRect
	debugc2mtk              ClsnRect
	debugc2grd              ClsnRect
	debugc2stb              ClsnRect
	debugcsize              ClsnRect
	debugch                 ClsnRect
	accel                   float32
	clsnSpr                 Sprite
	clsnDisplay             bool
	lifebarDisplay          bool
	mainThreadTask          chan func()
	workpal                 []uint32
	errLog                  *log.Logger
	nomusic                 bool
	workBe                  []BytecodeExp
	keyInput                Key
	keyString               string
	timerCount              []int32
	cmdFlags                map[string]string
	//FLAC_FrameWait          int

	// Localcoord sceenpack
	luaLocalcoord    [2]int32
	luaSpriteScale   float32
	luaPortraitScale float32
	luaSpriteOffsetX float32

	// Localcoord lifebar
	lifebarScale         float32
	lifebarOffsetX       float32
	lifebarOffsetY       float32
	lifebarPortraitScale float32
	lifebarLocalcoord    [2]int32

	msaa              int32
	externalShaders   [][]string
	windowMainIcon    []image.Image
	gameMode          string
	frameCounter      int32
	preFightTime      int32
	motifDir          string
	captureNum        int
	decisiveRound     [2]bool
	timerStart        int32
	timerRounds       []int32
	scoreStart        [2]float32
	scoreRounds       [][2]float32
	matchData         *lua.LTable
	consecutiveWins   [2]int32
	consecutiveRounds bool
	firstAttack       [3]int
	teamLeader        [2]int
	gameSpeed         float32
	maxPowerMode      bool
	clsnText          []ClsnText
	consoleText       []string
	luaLState         *lua.LState
	statusLFunc       *lua.LFunction
	listLFunc         []*lua.LFunction
	introSkipped      bool
	endMatch          bool
	continueFlg       bool
	dialogueFlg       bool
	dialogueForce     int
	dialogueBarsFlg   bool
	noSoundFlg        bool
	postMatchFlg      bool
	continueScreenFlg bool
	victoryScreenFlg  bool
	winScreenFlg      bool
	playBgmFlg        bool
	brightnessOld     int32
	loopBreak         bool
	loopContinue      bool

	// for avg. FPS calculations
	gameFPS       float32
	prevTimestamp float64
	absTickCountF float32

	// screenshot deferral
	isTakingScreenshot bool
}

// Check if the application is running inside a macOS app bundle
func isRunningInsideAppBundle(exePath string) bool {
	// Check if we're on Darwin and the executable path contains .app (macOS application bundle)
	return runtime.GOOS == "darwin" && strings.Contains(exePath, ".app")
}

// Initialize stuff, this is called after the config int at main.go
func (s *System) init(w, h int32) *lua.LState {
	s.setWindowSize(w, h)
	var err error
	// Create a system window.
	s.window, err = s.newWindow(int(s.scrrect[2]), int(s.scrrect[3]))
	chk(err)

	// Correct the joystick mappings (macOS)
	if runtime.GOOS == "darwin" {
		for i := 0; i < len(sys.joystickConfig); i++ {
			jc := &sys.joystickConfig[i]
			joyS := jc.Joy

			if input.IsJoystickPresent(joyS) {
				guid := input.GetJoystickGUID(joyS)

				// Correct the inner config
				if sys.joystickConfig[joyS].GUID != guid && !sys.joystickConfig[i].isInitialized {

					// Swap those that don't match
					joyIndices := input.GetJoystickIndices(guid)
					if len(joyIndices) == 1 {
						for j := 0; j < len(sys.joystickConfig); j++ {
							if sys.joystickConfig[j].GUID == guid {
								sys.joystickConfig[i].swap(&sys.joystickConfig[j])
								break
							}
						}
					}
				}
			}
		}
	}

	// Loading of external shader data.
	// We need to do this before the render initialization at "gfx.Init()"
	if len(s.cfg.Video.ExternalShaders) > 0 {
		// First we initialize arrays.
		s.externalShaders = make([][]string, 2)
		s.externalShaders[0] = make([]string, len(s.cfg.Video.ExternalShaders))
		s.externalShaders[1] = make([]string, len(s.cfg.Video.ExternalShaders))

		// Then we load.
		for i, shaderLocation := range s.cfg.Video.ExternalShaders {
			// Create names.
			shaderLocation = strings.Replace(shaderLocation, "\\", "/", -1)

			// Load vert shaders.
			content, err := os.ReadFile(shaderLocation + ".vert")
			if err != nil {
				chk(err)
			}
			s.externalShaders[0][i] = string(content) + "\x00"

			// Load frag shaders.
			content, err = os.ReadFile(shaderLocation + ".frag")
			if err != nil {
				chk(err)
			}
			s.externalShaders[1][i] = string(content) + "\x00"
		}
	}
	// PS: The "\x00" is what is know as Null Terminator.

	// Now we proceed to init the render.
	if s.cfg.Video.RenderMode == "OpenGL 2.1" {
		gfx = &Renderer_GL21{}
		gfxFont = &glfont.FontRenderer_GL21{}
	} else {
		gfx = &Renderer_GL32{}
		gfxFont = &glfont.FontRenderer_GL32{}
	}
	gfx.Init()
	gfx.BeginFrame(false)
	// And the audio.
	speaker.Init(beep.SampleRate(sys.cfg.Sound.SampleRate), audioOutLen)
	speaker.Play(NewNormalizer(s.soundMixer))
	l := lua.NewState()
	l.Options.IncludeGoStackTrace = true
	l.OpenLibs()
	for i := range s.inputRemap {
		s.inputRemap[i] = i
	}
	for i := range s.stringPool {
		s.stringPool[i] = *NewStringPool()
	}
	s.clsnSpr = *newSprite()
	s.clsnSpr.Size, s.clsnSpr.Pal = [...]uint16{1, 1}, make([]uint32, 256)
	s.clsnSpr.SetPxl([]byte{0})
	systemScriptInit(l)
	s.shortcutScripts = make(map[ShortcutKey]*ShortcutScript)
	// So now that we have a window we add an icon.
	if len(s.cfg.Config.WindowIcon) > 0 {
		// First we initialize arrays.
		var f = make([]io.ReadCloser, len(s.cfg.Config.WindowIcon))
		s.windowMainIcon = make([]image.Image, len(s.cfg.Config.WindowIcon))
		// And then we load them.
		for i, iconLocation := range s.cfg.Config.WindowIcon {
			exePath, err := os.Executable()
			if err != nil {
				fmt.Println("Error getting executable path:", err)
			} else {
				// Change the context for Darwin if we're in an app bundle
				if isRunningInsideAppBundle(exePath) {
					os.Chdir(path.Dir(exePath))
					os.Chdir("../../../")
				}
			}
			f[i], err = os.Open(iconLocation)
			if err != nil {
				var dErr = "Icon file can not be found.\nPanic: " + err.Error()
				ShowErrorDialog(dErr)
				panic(Error(dErr))
			}
			s.windowMainIcon[i], _, err = image.Decode(f[i])
		}
		s.window.SetIcon(s.windowMainIcon)
		chk(err)
	}
	// [Icon add end]

	// Error print?
	go func() {
		stdin := bufio.NewScanner(os.Stdin)
		for stdin.Scan() {
			if err := stdin.Err(); err != nil {
				s.errLog.Println(err.Error())
				return
			}
			s.commandLine <- stdin.Text()
		}
	}()
	return l
}

func (s *System) shutdown() {
	if !sys.gameEnd {
		sys.gameEnd = true
	}
	gfx.Close()
	s.window.Close()
	speaker.Close()
}

func (s *System) setWindowSize(w, h int32) {
	s.scrrect[2], s.scrrect[3] = w, h
	if s.scrrect[2]*3 > s.scrrect[3]*4 {
		s.gameWidth, s.gameHeight = s.scrrect[2]*3*320/(s.scrrect[3]*4), 240
	} else {
		s.gameWidth, s.gameHeight = 320, s.scrrect[3]*4*240/(s.scrrect[2]*3)
	}
	s.widthScale = float32(s.scrrect[2]) / float32(s.gameWidth)
	s.heightScale = float32(s.scrrect[3]) / float32(s.gameHeight)
}

func (s *System) eventUpdate() bool {
	s.esc = false
	for _, v := range s.shortcutScripts {
		v.Activate = false
	}
	s.window.pollEvents()
	s.gameEnd = s.window.shouldClose()
	return !s.gameEnd
}

func (s *System) runMainThreadTask() {
	for {
		select {
		case f := <-s.mainThreadTask:
			f()
		default:
			return
		}
	}
}

func (s *System) await(fps int) bool {
	if !s.frameSkip {
		// Render the finished frame
		gfx.EndFrame()
		s.window.SwapBuffers()
		if s.isTakingScreenshot {
			defer captureScreen()
			s.isTakingScreenshot = false
		}
		// Begin the next frame after events have been processed. Do not clear
		// the screen if network input is present.
		defer gfx.BeginFrame(sys.netConnection == nil)
	}
	s.runMainThreadTask()
	now := time.Now()
	diff := s.redrawWait.nextTime.Sub(now)
	wait := time.Second / time.Duration(fps)
	s.redrawWait.nextTime = s.redrawWait.nextTime.Add(wait)
	switch {
	case diff >= 0 && diff < wait+2*time.Millisecond:
		time.Sleep(diff)
		fallthrough
	case now.Sub(s.redrawWait.lastDraw) > 250*time.Millisecond:
		fallthrough
	case diff >= -17*time.Millisecond:
		s.redrawWait.lastDraw = now
		s.frameSkip = false
	default:
		if diff < -150*time.Millisecond {
			s.redrawWait.nextTime = now.Add(wait)
		}
		s.frameSkip = true
	}
	s.eventUpdate()

	return !s.gameEnd
}

func (s *System) update() bool {
	s.frameCounter++
	if s.gameTime == 0 {
		s.preFightTime = s.frameCounter
	}
	if s.replayFile != nil {
		if s.anyHardButton() {
			s.await(s.cfg.Config.Framerate * 4)
		} else {
			s.await(s.cfg.Config.Framerate)
		}
		return s.replayFile.Update()
	}
	if s.netConnection != nil {
		s.await(s.cfg.Config.Framerate)
		return s.netConnection.Update()
	}
	return s.await(s.cfg.Config.Framerate)
}

func (s *System) tickSound() {
	s.soundChannels.Tick()
	if !s.noSoundFlg {
		for _, ch := range s.chars {
			for _, c := range ch {
				c.soundChannels.Tick()
			}
		}
	}

	// Always pause if noMusic flag set or pause master volume is 0.
	s.bgm.SetPaused(s.nomusic || (s.paused && s.cfg.Sound.PauseMasterVolume == 0))

	// Set BGM volume if paused
	if s.paused && s.bgm.volRestore == 0 {
		s.bgm.volRestore = s.bgm.bgmVolume
		s.bgm.bgmVolume = int(s.cfg.Sound.PauseMasterVolume * s.bgm.bgmVolume / 100.0)
		s.bgm.UpdateVolume()
		s.softenAllSound()
	} else if !s.paused && s.bgm.volRestore > 0 {
		// Restore all volume
		s.bgm.bgmVolume = s.bgm.volRestore
		s.bgm.volRestore = 0
		s.bgm.UpdateVolume()
		s.restoreAllVolume()
	}
}

func (s *System) resetRemapInput() {
	for i := range s.inputRemap {
		s.inputRemap[i] = i
	}
}

func (s *System) loaderReset() {
	s.round, s.wins, s.roundsExisted, s.decisiveRound = 1, [2]int32{}, [2]int32{}, [2]bool{}
	s.loader.reset()
}

func (s *System) loadStart() {
	s.loaderReset()
	s.loader.runTread()
}

func (s *System) synchronize() error {
	if s.replayFile != nil {
		s.replayFile.Synchronize()
	} else if s.netConnection != nil {
		return s.netConnection.Synchronize()
	}
	return nil
}

func (s *System) anyHardButton() bool {
	for _, kc := range s.keyConfig {
		if kc.a() || kc.b() || kc.c() || kc.x() || kc.y() || kc.z() {
			return true
		}
	}
	for _, kc := range s.joystickConfig {
		if kc.a() || kc.b() || kc.c() || kc.x() || kc.y() || kc.z() {
			return true
		}
	}
	return false
}

func (s *System) anyButton() bool {
	if s.replayFile != nil {
		return s.replayFile.AnyButton()
	}
	if s.netConnection != nil {
		return s.netConnection.AnyButton()
	}
	return s.anyHardButton()
}

func (s *System) playerID(id int32) *Char {
	return s.charList.get(id)
}

func (s *System) playerIndex(id int32) *Char {
	return s.charList.getIndex(id)
}

// We must check if wins are greater than 0 because modes like Training may have "0 rounds to win"
func (s *System) matchOver() bool {
	return s.wins[0] > 0 && s.wins[0] >= s.matchWins[0] ||
		s.wins[1] > 0 && s.wins[1] >= s.matchWins[1]
}

func (s *System) playerIDExist(id BytecodeValue) BytecodeValue {
	if id.IsSF() {
		return BytecodeSF()
	}
	return BytecodeBool(s.playerID(id.ToI()) != nil)
}

func (s *System) playerIndexExist(idx BytecodeValue) BytecodeValue {
	if idx.IsSF() {
		return BytecodeSF()
	}
	return BytecodeBool(s.playerIndex(idx.ToI()) != nil)
}

func (s *System) playerNoExist(no BytecodeValue) BytecodeValue {
	if no.IsSF() {
		return BytecodeSF()
	}
	exist := false
	number := int(no.ToI() - 1)
	if number >= 0 && number < len(sys.chars) {
		exist = len(sys.chars[number]) > 0
	}
	return BytecodeBool(exist)
}

func (s *System) playercount() int32 {
	return int32(len(s.charList.runOrder))
}

func (s *System) palfxvar(x int32, y int32) int32 {
	n := int32(0)
	if x >= 4 {
		n = 256
	}
	pfx := s.bgPalFX
	if y == 2 {
		pfx = s.allPalFX
	}
	if pfx.enable {
		switch x {
		case -2:
			n = pfx.eInvertblend
		case -1:
			n = Btoi(pfx.eInvertall)
		case 0:
			n = pfx.time
		case 1:
			n = pfx.eAdd[0]
		case 2:
			n = pfx.eAdd[1]
		case 3:
			n = pfx.eAdd[2]
		case 4:
			n = pfx.eMul[0]
		case 5:
			n = pfx.eMul[1]
		case 6:
			n = pfx.eMul[2]
		default:
			n = 0
		}
	}
	return n
}

func (s *System) palfxvar2(x int32, y int32) float32 {
	n := float32(1)
	if x > 1 {
		n = 0
	}
	pfx := s.bgPalFX
	if y == 2 {
		pfx = s.allPalFX
	}
	if pfx.enable {
		switch x {
		case 1:
			n = pfx.eColor
		case 2:
			n = pfx.eHue
		default:
			n = 0
		}
	}
	return n * 256
}

func (s *System) screenHeight() float32 {
	return 240
}

func (s *System) screenWidth() float32 {
	return float32(s.gameWidth)
}

func (s *System) roundEnd() bool {
	return s.intro < -s.lifebar.ro.over_hittime
}

// Characters cannot hurt each other between lifebar timers over.hittime and over.waittime
func (s *System) roundNoDamage() bool {
	return sys.intro < 0 && sys.intro <= -sys.lifebar.ro.over_hittime && sys.intro >= -sys.lifebar.ro.over_waittime
}

func (s *System) roundState() int32 {
	switch {
	case sys.intro > sys.lifebar.ro.ctrl_time+1 || sys.postMatchFlg:
		return 0
	case sys.lifebar.ro.current == 0:
		return 1
	case sys.intro >= 0 || sys.finishType == FT_NotYet:
		return 2
	case sys.intro < -sys.lifebar.ro.over_waittime:
		return 4
	default:
		return 3
	}
}

func (s *System) introState() int32 {
	switch {
	case s.intro > s.lifebar.ro.ctrl_time+1:
		// Pre-intro [RoundState = 0]
		return 1
	case (s.dialogueFlg && s.dialogueForce == 0) || s.intro == s.lifebar.ro.ctrl_time+1:
		// Player intros [RoundState = 1]
		return 2
	case s.intro == s.lifebar.ro.ctrl_time:
		// Dialogue detection (s.dialogueFlg is detectable 1 frame later)
		if s.dialogueForce == 0 {
			for _, p := range sys.chars {
				if len(p) > 0 && len(p[0].dialogue) > 0 {
					return 2
				}
			}
		}
		// Round announcement
		return 3
	case s.lifebar.ro.waitTimer[1] == -1 || (s.intro > 0 && s.intro < s.lifebar.ro.ctrl_time):
		// Fight called
		return 4
	default:
		// Not applicable
		return 0
	}
}

func (s *System) outroState() int32 {
	switch {
	case s.intro >= 0:
		// Not applicable
		return 0
	case s.roundOver():
		// Round over
		return 5
	case s.roundWinStates():
		// Player win states
		return 4
	case sys.intro < -sys.lifebar.ro.over_waittime || sys.lifebar.ro.over_waittime == 1:
		// Players lose control, but the round has not yet entered win states
		return 3
	case s.intro < -s.lifebar.ro.over_hittime || sys.lifebar.ro.over_hittime == 1:
		// Players still have control, but the match outcome can no longer be changed
		return 2
	case s.intro < 0:
		// Payers can still act, allowing a possible double KO
		return 1
	default:
		// Fallback case, shouldn't be reached
		return 0
	}
}

func (s *System) roundWinStates() bool {
	return s.waitdown <= 0 || s.roundWinTime()
}

func (s *System) roundWinTime() bool {
	return s.wintime < 0
}

func (s *System) roundOver() bool {
	return s.intro < -(s.lifebar.ro.over_waittime + s.lifebar.ro.over_time)
}

func (s *System) gsf(gsf GlobalSpecialFlag) bool {
	return s.specialFlag&gsf != 0
}

func (s *System) setGSF(gsf GlobalSpecialFlag) {
	s.specialFlag |= gsf
}

func (s *System) unsetGSF(gsf GlobalSpecialFlag) {
	s.specialFlag &^= gsf
}

func (s *System) appendToConsole(str string) {
	s.consoleText = append(s.consoleText, str)
	if len(s.consoleText) > s.cfg.Debug.ConsoleRows {
		s.consoleText = s.consoleText[len(s.consoleText)-s.cfg.Debug.ConsoleRows:]
	}
}

func (s *System) printToConsole(pn, sn int, a ...interface{}) {
	spl := s.stringPool[pn].List
	if sn >= 0 && sn < len(spl) {
		for _, str := range strings.Split(OldSprintf(spl[sn], a...), "\n") {
			fmt.Printf("%s\n", str)
			s.appendToConsole(str)
		}
	}
}

// Print an error directly from bytecode.go
// Printing from char.go is preferable, but not always possible
func (s *System) printBytecodeError(str string) {
	if s.loader.state == LS_Complete && s.workingChar != nil {
		// Print during matches
		s.appendToConsole(sys.workingChar.warn() + str)
	} else if !sys.ignoreMostErrors {
		// Print outside matches (compiling)
		sys.errLog.Println(str)
	}
}

func (s *System) loadTime(start time.Time, str string, shell, console bool) {
	elapsed := time.Since(start)
	str = fmt.Sprintf("%v; Load time: %v", str, elapsed)
	if shell {
		fmt.Printf("%s\n", str)
	}
	if console {
		s.appendToConsole(str)
	}
}

// Update Z scale
// TODO: See if this still works correctly with Winmugen stages that scaled chars with Z
func (s *System) updateZScale(pos, localscale float32) float32 {
	topz := sys.stage.stageCamera.topz / localscale
	botz := sys.stage.stageCamera.botz / localscale
	scale := float32(1)
	if topz != botz {
		ztopscale, zbotscale := sys.stage.stageCamera.ztopscale, sys.stage.stageCamera.zbotscale
		d := (pos - topz) / (botz - topz)
		scale = ztopscale + d*(zbotscale-ztopscale)
		if scale <= 0 {
			scale = 0
		}
	}
	return scale
}

func (s *System) zEnabled() bool {
	return s.zmin != s.zmax
}

// Convert X and Y drawing position to Z perspective
func (s *System) drawposXYfromZ(inpos [2]float32, localscl, zpos, zscale float32) (outpos [2]float32) {
	outpos[0] = (inpos[0]-s.cam.Pos[0])*zscale + s.cam.Pos[0]
	outpos[1] = inpos[1] * zscale
	outpos[1] += s.posZtoYoffset(zpos, localscl) // "Z" position
	return
}

// Convert Z logic position to Y drawing offset
// This is separate from the above because shadows only need this part
func (s *System) posZtoYoffset(zpos, localscl float32) float32 {
	return zpos * localscl * s.stage.stageCamera.depthtoscreen
}

// Z axis check
// Changed to no longer check z enable constant, depends on stage now
func (s *System) zAxisOverlap(posz1, top1, bot1, localscl1, posz2, top2, bot2, localscl2 float32) bool {
	if s.zEnabled() {
		if (posz1+bot1)*localscl1 < (posz2-top2)*localscl2 ||
			(posz1-top1)*localscl1 > (posz2+bot2)*localscl2 {
			return false
		}
	}
	return true
}

func (s *System) clsnOverlap(clsn1 [][4]float32, scl1, pos1 [2]float32, facing1 float32, angle1 float32,
	clsn2 [][4]float32, scl2, pos2 [2]float32, facing2 float32, angle2 float32) bool {

	// Skip function if any boxes are missing
	if clsn1 == nil || clsn2 == nil {
		return false
	}
	anface1 := facing1
	anface2 := facing2

	// Flip boxes if scale < 0
	if scl1[0] < 0 {
		facing1 *= -1
		scl1[0] *= -1
	}
	if scl2[0] < 0 {
		facing2 *= -1
		scl2[0] *= -1
	}

	// Loop through first set of boxes
	for i := 0; i < len(clsn1); i++ {
		// Calculate positions
		l1 := clsn1[i][0]
		r1 := clsn1[i][2]
		if facing1 < 0 {
			l1, r1 = -r1, -l1
		}
		left1 := l1 * scl1[0]
		right1 := r1 * scl1[0]
		top1 := clsn1[i][1] * scl1[1]
		bottom1 := clsn1[i][3] * scl1[1]

		// Loop through second set of boxes
		for j := 0; j < len(clsn2); j++ {
			// Calculate positions
			l2 := clsn2[j][0]
			r2 := clsn2[j][2]
			if facing2 < 0 {
				l2, r2 = -r2, -l2
			}
			left2 := l2 * scl2[0]
			right2 := r2 * scl2[0]
			top2 := clsn2[j][1] * scl2[1]
			bottom2 := clsn2[j][3] * scl2[1]

			// Check for overlap
			if angle1 != 0 || angle2 != 0 {
				if RectIntersect(left1+pos1[0], top1+pos1[1], right1-left1, bottom1-top1,
					left2+pos2[0], top2+pos2[1], right2-left2, bottom2-top2, pos1[0], pos1[1], pos2[0], pos2[1],
					-Rad(angle1*anface1), -Rad(angle2*anface2)) {
					return true
				}
			} else {
				if left1+pos1[0] <= right2+pos2[0] &&
					left2+pos2[0] <= right1+pos1[0] &&
					top1+pos1[1] <= bottom2+pos2[1] &&
					top2+pos2[1] <= bottom1+pos1[1] {
					return true
				}
			}
		}
	}

	return false
}

func (s *System) newCharId() int32 {
	// Check if the next ID is already being used by a helper with "preserve"
	// We specifically check for preserved helpers because otherwise this also detects the players from the previous match that are being replaced
	newid := s.nextCharId
	taken := true
	for taken {
		taken = false
		for _, p := range s.chars {
			for _, c := range p {
				if c.id == newid && c.preserve != 0 && !c.csf(CSF_destroy) {
					taken = true
					newid++
					break
				}
			}
			if taken {
				break // Skip outer loop
			}
		}
	}
	s.nextCharId = newid + 1
	return newid
}

func (s *System) resetGblEffect() {
	s.allPalFX.clear()
	s.bgPalFX.clear()
	s.envShake.clear()
	s.pausetime, s.pausetimebuffer = 0, 0
	s.supertime, s.supertimebuffer = 0, 0
	s.superanim = nil
	s.envcol_time = 0
	s.specialFlag = 0
}

func (s *System) stopAllSound() {
	for _, p := range s.chars {
		for _, c := range p {
			c.soundChannels.SetSize(0)
		}
	}
}

func (s *System) softenAllSound() {
	for _, p := range s.chars {
		for _, c := range p {
			for i := 0; i < int(c.soundChannels.count()); i++ {
				// Temporarily store the volume so it can be recalled later.
				if c.soundChannels.channels[i].sfx != nil && c.soundChannels.channels[i].ctrl != nil {
					c.soundChannels.volResume[i] = c.soundChannels.channels[i].sfx.volume
					c.soundChannels.channels[i].SetVolume(float32(c.gi().data.volume * int32(s.cfg.Sound.PauseMasterVolume) / 100))

					// Pause if pause master volume is 0
					if s.cfg.Sound.PauseMasterVolume == 0 {
						c.soundChannels.channels[i].SetPaused(true)
					}
				}
			}
		}
	}
	// Don't pause motif sounds
}

func (s *System) restoreAllVolume() {
	for _, p := range s.chars {
		for _, c := range p {
			for i := 0; i < int(c.soundChannels.count()); i++ {
				// Restore the volume we had.
				if c.soundChannels.channels[i].sfx != nil && c.soundChannels.channels[i].ctrl != nil {
					c.soundChannels.channels[i].SetVolume(c.soundChannels.volResume[i])

					// Unpause
					if c.soundChannels.channels[i].ctrl.Paused {
						c.soundChannels.channels[i].SetPaused(false)
					}
				}
			}
		}
	}
}

func (s *System) clearAllSound() {
	s.soundChannels.StopAll()
	s.stopAllSound()
}

// Remove the player's explods, projectiles and (optionally) helpers as well as stopping their sounds
func (s *System) clearPlayerAssets(pn int, destroy bool) {
	if len(s.chars[pn]) > 0 {
		p := s.chars[pn][0]
		for _, h := range s.chars[pn][1:] {
			if destroy || h.preserve == 0 || (s.roundResetFlg && h.preserve == s.round) {
				h.destroy()
			}
			h.soundChannels.SetSize(0)
		}
		if destroy {
			p.children = p.children[:0]
		} else {
			for i, ch := range p.children {
				if ch != nil {
					if ch.preserve == 0 || (s.roundResetFlg && ch.preserve == s.round) {
						p.children[i] = nil
					}
				}
			}
		}
		p.targets = p.targets[:0]
		p.soundChannels.SetSize(0)
	}
	s.projs[pn] = s.projs[pn][:0]
	s.explods[pn] = s.explods[pn][:0]
	s.explodsLayerN1[pn] = s.explodsLayerN1[pn][:0]
	s.explodsLayer0[pn] = s.explodsLayer0[pn][:0]
	s.explodsLayer1[pn] = s.explodsLayer1[pn][:0]
}

func (s *System) nextRound() {
	s.resetGblEffect()
	s.lifebar.reset()
	s.firstAttack = [3]int{-1, -1, 0}
	s.finishType = FT_NotYet
	s.winTeam = -1
	s.winType = [...]WinType{WT_Normal, WT_Normal}
	s.winTrigger = [...]WinType{WT_Normal, WT_Normal}
	s.lastHitter = [2]int{-1, -1}
	s.waitdown = s.lifebar.ro.over_waittime + 900
	s.slowtime = s.lifebar.ro.slow_time
	s.shuttertime = 0
	s.fadeintime = s.lifebar.ro.fadein_time
	s.fadeouttime = s.lifebar.ro.fadeout_time
	s.wintime = s.lifebar.ro.over_wintime
	s.winskipped = false
	s.intro = s.lifebar.ro.start_waittime + s.lifebar.ro.ctrl_time + 1
	s.time = s.roundTime
	s.nextCharId = s.cfg.Config.HelperMax
	if (s.tmode[0] == TM_Turns && s.wins[1] >= s.numTurns[0]-1) ||
		(s.tmode[0] != TM_Turns && s.wins[1] >= s.lifebar.ro.match_wins[0]-1) {
		s.decisiveRound[0] = true
	}
	if (s.tmode[1] == TM_Turns && s.wins[0] >= s.numTurns[1]-1) ||
		(s.tmode[1] != TM_Turns && s.wins[0] >= s.lifebar.ro.match_wins[1]-1) {
		s.decisiveRound[1] = true
	}
	var roundRef int32
	if s.round == 1 {
		s.stageLoopNo = 0
	} else {
		roundRef = s.round
	}
	if s.stageLoop && !s.roundResetFlg {
		var keys []int
		for k := range s.stageList {
			keys = append(keys, int(k))
		}
		sort.Ints(keys)
		roundRef = int32(keys[s.stageLoopNo])
		s.stageLoopNo++
		if s.stageLoopNo >= len(s.stageList) {
			s.stageLoopNo = 0
		}
	}
	var swap bool
	if _, ok := s.stageList[roundRef]; ok {
		s.stage = s.stageList[roundRef]
		if s.round > 1 && !s.roundResetFlg {
			swap = true
		}
		if s.stage.model != nil {
			sys.mainThreadTask <- func() {
				gfx.SetModelVertexData(0, s.stage.model.vertexBuffer)
				gfx.SetModelIndexData(0, s.stage.model.elementBuffer...)
			}
		}
	}
	s.cam.stageCamera = s.stage.stageCamera
	s.cam.Init()
	s.screenleft = float32(s.stage.screenleft) * s.stage.localscl
	s.screenright = float32(s.stage.screenright) * s.stage.localscl
	if s.stage.resetbg || swap {
		s.stage.reset()
	}
	s.cam.ResetZoomdelay()
	for i, p := range s.chars {
		if len(p) > 0 {
			s.nextCharId = Max(s.nextCharId, p[0].id+1) // nextCharId can't be this char's ID
			s.clearPlayerAssets(i, false)
			p[0].posReset()
			p[0].setCtrl(false)
			p[0].clearState()
			p[0].prepareNextRound()
			p[0].varRangeSet(0, s.cgi[i].data.intpersistindex-1, 0)
			p[0].fvarRangeSet(0, s.cgi[i].data.floatpersistindex-1, 0)
			for j := range p[0].cmd {
				p[0].cmd[j].BufReset()
			}
			if s.roundsExisted[i&1] == 0 {
				s.cgi[i].palettedata.palList.ResetRemap()
				if s.cgi[i].sff.header.Ver0 == 1 {
					p[0].remapPal(p[0].getPalfx(),
						[...]int32{1, 1}, [...]int32{1, s.cgi[i].palno})
				}
			}
			s.cgi[i].clearPCTime()
		}
	}
	for _, p := range s.chars {
		if len(p) > 0 {
			zeroDeclared := p[0].gi().anim[0] != nil

			if zeroDeclared {
				p[0].selfState(5900, 0, -1, 0, "")
			} else {
				// Default to first anim in .AIR
				var firstAnim int32
				for k := range p[0].gi().anim {
					firstAnim = k
					break
				}
				p[0].selfState(5900, firstAnim, -1, 0, "")
			}
		}
	}
}

func (s *System) debugPaused() bool {
	return s.paused && !s.step && s.oldTickCount < s.tickCount
}

// "Tick frames" are the frames where most of the game logic happens
func (s *System) tickFrame() bool {
	return (!s.paused || s.step) && s.oldTickCount < s.tickCount
}

// "Tick next frame" is right after the "tick frame"
// Where for instance the collision detections happen
func (s *System) tickNextFrame() bool {
	return int(s.tickCountF+s.nextAddTime) > s.tickCount &&
		(!s.paused || s.step || s.oldTickCount >= s.tickCount)
}

// This divides a frame into fractions for the purpose of drawing position interpolation
func (s *System) tickInterpolation() float32 {
	if s.tickNextFrame() {
		return 1
	} else {
		return s.tickCountF - s.lastTick + s.nextAddTime
	}
}

func (s *System) addFrameTime(t float32) bool {
	if s.debugPaused() {
		s.oldNextAddTime = 0
		return true
	}
	s.oldTickCount = s.tickCount
	if int(s.tickCountF) > s.tickCount {
		s.tickCount++
		return false
	}
	s.tickCountF += s.nextAddTime
	if int(s.tickCountF) > s.tickCount {
		s.tickCount++
		s.lastTick = s.tickCountF
	}
	s.oldNextAddTime = s.nextAddTime
	s.nextAddTime = t
	return true
}

func (s *System) resetFrameTime() {
	s.tickCount, s.oldTickCount, s.tickCountF, s.lastTick, s.absTickCountF = 0, -1, 0, 0, 0
	s.nextAddTime, s.oldNextAddTime = 1, 1
}

func (s *System) charUpdate() {
	s.charList.update()
	// Because sys.projs has actual elements rather than pointers like sys.chars does, it's important to not copy its contents with range
	// https://github.com/ikemen-engine/Ikemen-GO/discussions/1707
	// for i, pr := range s.projs {
	for i := range s.projs {
		for j := range s.projs[i] {
			if s.projs[i][j].id >= 0 {
				s.projs[i][j].playerno = i // Safeguard
				s.projs[i][j].update()
			}
		}
	}
	// Set global First Attack flag if either team got it
	if s.firstAttack[0] >= 0 || s.firstAttack[1] >= 0 {
		s.firstAttack[2] = 1
	}
}

// Run collision detection for chars and projectiles
func (s *System) globalCollision() {
	for i := range s.projs {
		for j := range s.projs[i] {
			if s.projs[i][j].id >= 0 {
				s.projs[i][j].tradeDetection(i, j)
			}
		}
	}
	s.charList.collisionDetection()
	for i := range s.projs {
		for j := range s.projs[i] {
			if s.projs[i][j].id != IErr {
				s.projs[i][j].tick()
			}
		}
	}
}

func (s *System) posReset() {
	for _, p := range s.chars {
		if len(p) > 0 {
			p[0].posReset()
		}
	}
}

func (s *System) action() {
	// Clear sprite data
	s.spritesLayerN1 = s.spritesLayerN1[:0]
	s.spritesLayerU = s.spritesLayerU[:0]
	s.spritesLayer0 = s.spritesLayer0[:0]
	s.spritesLayer1 = s.spritesLayer1[:0]
	s.shadows = s.shadows[:0]
	s.debugc1hit = s.debugc1hit[:0]
	s.debugc1rev = s.debugc1rev[:0]
	s.debugc1not = s.debugc1not[:0]
	s.debugc2 = s.debugc2[:0]
	s.debugc2hb = s.debugc2hb[:0]
	s.debugc2mtk = s.debugc2mtk[:0]
	s.debugc2grd = s.debugc2grd[:0]
	s.debugc2stb = s.debugc2stb[:0]
	s.debugcsize = s.debugcsize[:0]
	s.debugch = s.debugch[:0]
	s.clsnText = nil

	var x, y, scl float32 = s.cam.Pos[0], s.cam.Pos[1], s.cam.Scale / s.cam.BaseScale()
	s.cam.ResetTracking()

	// Run fight screen
	if s.lifebar.ro.act() {
		if s.intro > s.lifebar.ro.ctrl_time {
			s.intro--
			if s.gsf(GSF_intro) && s.intro <= s.lifebar.ro.ctrl_time {
				s.intro = s.lifebar.ro.ctrl_time + 1
			}
		} else if s.intro > 0 {
			if s.intro == s.lifebar.ro.ctrl_time {
				for _, p := range s.chars {
					if len(p) > 0 {
						if !p[0].asf(ASF_nointroreset) {
							p[0].posReset()
						}
					}
				}
			}
			s.intro--
			if s.intro == 0 {
				for _, p := range s.chars {
					if len(p) > 0 {
						if p[0].alive() {
							p[0].unsetSCF(SCF_over_alive)
							if !p[0].scf(SCF_standby) || p[0].teamside == -1 {
								p[0].setCtrl(true)
								if p[0].ss.no != 0 && !p[0].asf(ASF_nointroreset) {
									p[0].selfState(0, -1, -1, 1, "")
								}
							}
						}
					}
				}
			}
		}
		if s.intro == 0 && s.time > 0 && !s.gsf(GSF_timerfreeze) &&
			(s.supertime <= 0 || !s.superpausebg) && (s.pausetime <= 0 || !s.pausebg) {
			s.time--
		}

		// Check if round ended by KO or time over and set win types
		fin := func() bool {
			checkPerfect := func(team int) bool {
				for i := team; i < MaxSimul*2; i += 2 {
					if len(s.chars[i]) > 0 &&
						s.chars[i][0].life < s.chars[i][0].lifeMax {
						return false
					}
				}
				return true
			}
			if s.intro > 0 {
				return false
			}
			// KO
			ko := [...]bool{true, true}
			for loser := range ko {
				// Check if all players or leader on one side are KO
				for i := loser; i < MaxSimul*2; i += 2 {
					if len(s.chars[i]) > 0 && s.chars[i][0].teamside != -1 {
						if s.chars[i][0].alive() {
							ko[loser] = false
						} else if (s.tmode[i&1] == TM_Simul && s.cfg.Options.Simul.LoseOnKO && s.aiLevel[i] == 0) ||
							(s.tmode[i&1] == TM_Tag && s.cfg.Options.Tag.LoseOnKO) {
							ko[loser] = true
							break
						}
					}
				}
				if ko[loser] {
					if checkPerfect(loser ^ 1) {
						s.winType[loser^1].SetPerfect()
					}
				}
			}
			// Time over
			ft := s.finishType
			if s.time == 0 {
				s.winType[0], s.winType[1] = WT_Time, WT_Time
				l := [2]float32{}
				for i := 0; i < 2; i++ { // Check life percentage of each team
					for j := i; j < MaxSimul*2; j += 2 {
						if len(s.chars[j]) > 0 {
							if s.tmode[i] == TM_Simul || s.tmode[i] == TM_Tag {
								l[i] += (float32(s.chars[j][0].life) / float32(s.numSimul[i])) / float32(s.chars[j][0].lifeMax)
							} else {
								l[i] += float32(s.chars[j][0].life) / float32(s.chars[j][0].lifeMax)
							}
						}
					}
				}
				// Some other methods were considered to make the winner decision more fair, like a minimum % difference
				// But ultimately a direct comparison seems to be the fairest method
				if math.Round(float64(l[0]*1000)) != math.Round(float64(l[1]*1000)) || // Convert back to 1000 life points scale then round it to reduce calculation errors
					((l[0] >= float32(1.0)) != (l[1] >= float32(1.0))) { // But make sure the rounding doesn't turn a perfect into a draw game
					winner := 0
					if l[0] < l[1] {
						winner = 1
					}
					if checkPerfect(winner) {
						s.winType[winner].SetPerfect()
					}
					s.finishType = FT_TO
					s.winTeam = winner
				} else { // Draw game
					s.finishType = FT_TODraw
					s.winTeam = -1
				}
			}
			if s.intro >= -1 && (ko[0] || ko[1]) {
				if ko[0] && ko[1] {
					s.finishType = FT_DKO
					s.winTeam = -1
				} else {
					s.finishType = FT_KO
					s.winTeam = int(Btoi(ko[0]))
				}
			}
			// Update win triggers if finish type was changed
			if ft != s.finishType {
				for i, p := range s.chars {
					if len(p) > 0 && ko[^i&1] {
						for _, h := range p {
							for _, tid := range h.targets {
								if t := s.playerID(tid); t != nil {
									if t.ghv.attr&int32(AT_AH) != 0 {
										s.winTrigger[i&1] = WT_Hyper
									} else if t.ghv.attr&int32(AT_AS) != 0 && s.winTrigger[i&1] == WT_Normal {
										s.winTrigger[i&1] = WT_Special
									}
								}
							}
						}
					}
				}
			}
			return ko[0] || ko[1] || s.time == 0
		}

		// Post round
		if s.roundEnd() || fin() {
			rs4t := -s.lifebar.ro.over_waittime
			s.intro--
			if s.intro == -s.lifebar.ro.over_hittime && s.finishType != FT_NotYet {
				// Consecutive wins counter
				winner := [...]bool{!s.chars[1][0].win(), !s.chars[0][0].win()}
				if !winner[0] || !winner[1] ||
					s.tmode[0] == TM_Turns || s.tmode[1] == TM_Turns ||
					s.draws >= s.lifebar.ro.match_maxdrawgames[0] ||
					s.draws >= s.lifebar.ro.match_maxdrawgames[1] {
					for i, win := range winner {
						if win {
							s.wins[i]++
							if s.matchOver() && s.wins[^i&1] == 0 {
								s.consecutiveWins[i]++
							}
							s.consecutiveWins[^i&1] = 0
						}
					}
				}
			}
			// Check if player skipped win pose time
			if s.roundWinTime() && (s.anyButton() && !s.gsf(GSF_roundnotskip)) {
				s.intro = Min(s.intro, rs4t-2-s.lifebar.ro.over_time+s.lifebar.ro.fadeout_time)
				s.winskipped = true
			}
			if s.winskipped || !s.roundWinTime() {
				// Check if game can proceed into roundstate 4
				if s.waitdown > 0 {
					if s.intro == rs4t-1 {
						for _, p := range s.chars {
							if len(p) > 0 {
								// Check if this player is ready to proceed to roundstate 4
								// TODO: The game should normally only wait for players that are active in the fight // || p[0].teamside == -1 || p[0].scf(SCF_standby)
								// TODO: This could be manageable from the char's side with an AssertSpecial or such
								if p[0].scf(SCF_over_alive) || p[0].scf(SCF_over_ko) ||
									(p[0].scf(SCF_ctrl) && p[0].ss.moveType == MT_I && p[0].ss.stateType != ST_A && p[0].ss.stateType != ST_L) {
									continue
								}
								// Freeze timer if any player is not ready to proceed yet
								s.intro = rs4t
								break
							}
						}
					}
				}
				// Disable ctrl (once) at the first frame of roundstate 4
				if s.intro == rs4t-1 {
					for _, p := range s.chars {
						if len(p) > 0 {
							p[0].setCtrl(false)
						}
					}
				}
				// Start running wintime counter only after getting into roundstate 4
				if s.intro < rs4t && !s.roundWinTime() {
					s.wintime--
				}
				// Set characters into win/lose poses, update win counters
				if s.roundWinStates() {
					if s.waitdown >= 0 {
						winner := [...]bool{!s.chars[1][0].win(), !s.chars[0][0].win()}
						if !winner[0] || !winner[1] ||
							s.tmode[0] == TM_Turns || s.tmode[1] == TM_Turns ||
							s.draws >= s.lifebar.ro.match_maxdrawgames[0] ||
							s.draws >= s.lifebar.ro.match_maxdrawgames[1] {
							for i, win := range winner {
								if win {
									s.lifebar.wi[i].add(s.winType[i])
									if s.matchOver() {
										// In a draw game both players go back to 0 wins
										if winner[0] == winner[1] { // sys.winTeam < 0
											s.lifebar.wc[0].wins = 0
											s.lifebar.wc[1].wins = 0
										} else {
											if s.wins[i] >= s.matchWins[i] {
												s.lifebar.wc[i].wins += 1
											}
										}
									}
								}
							}
						} else {
							s.draws++
						}
					}
					for _, p := range s.chars {
						if len(p) > 0 {
							// Default life recovery. Used only if externalized Lua implementation is disabled
							if len(sys.cfg.Common.Lua) == 0 && s.waitdown >= 0 && s.time > 0 && p[0].win() &&
								p[0].alive() && !s.matchOver() &&
								(s.tmode[0] == TM_Turns || s.tmode[1] == TM_Turns) {
								p[0].life += int32((float32(p[0].lifeMax) *
									float32(s.time) / 60) * s.turnsRecoveryRate)
								if p[0].life > p[0].lifeMax {
									p[0].life = p[0].lifeMax
								}
							}
							// TODO: These changestates ought to be unhardcoded
							if !p[0].scf(SCF_over_alive) && !p[0].hitPause() && p[0].alive() && p[0].animNo != 5 {
								p[0].setSCF(SCF_over_alive)
								if p[0].win() {
									p[0].selfState(180, -1, -1, -1, "")
								} else if p[0].lose() {
									p[0].selfState(170, -1, -1, -1, "")
								} else {
									p[0].selfState(175, -1, -1, -1, "")
								}
							}
						}
					}
					s.waitdown = 0
				}
				s.waitdown--
			}
			// If the game can't proceed to the fadeout screen, we turn back the counter 1 tick
			if !s.winskipped && s.gsf(GSF_roundnotover) &&
				s.intro == rs4t-2-s.lifebar.ro.over_time+s.lifebar.ro.fadeout_time {
				s.intro++
			}
		} else if s.intro < 0 {
			s.intro = 0
		}
	}

	// Run "tick frame"
	if s.tickFrame() {
		// X axis player limits
		s.xmin = s.cam.ScreenPos[0] + s.cam.Offset[0] + s.screenleft
		s.xmax = s.cam.ScreenPos[0] + s.cam.Offset[0] + float32(s.gameWidth)/s.cam.Scale - s.screenright
		if s.xmin > s.xmax {
			s.xmin = (s.xmin + s.xmax) / 2
			s.xmax = s.xmin
		}
		if AbsF(s.cam.maxRight-s.xmax) < 0.0001 {
			s.xmax = s.cam.maxRight
		}
		if AbsF(s.cam.minLeft-s.xmin) < 0.0001 {
			s.xmin = s.cam.minLeft
		}
		// Z axis player limits
		s.zmin = s.stage.topbound * s.stage.localscl
		s.zmax = s.stage.botbound * s.stage.localscl
		s.allPalFX.step()
		//s.bgPalFX.step()
		s.envShake.next()
		if s.envcol_time > 0 {
			s.envcol_time--
		}
		if s.enableZoomtime > 0 {
			s.enableZoomtime--
		} else {
			s.zoomCameraBound = true
			s.zoomStageBound = true
		}
		if s.supertime > 0 {
			s.supertime--
		} else if s.pausetime > 0 {
			s.pausetime--
		}
		if s.supertimebuffer < 0 {
			s.supertimebuffer = ^s.supertimebuffer
			s.supertime = s.supertimebuffer
		}
		if s.pausetimebuffer < 0 {
			s.pausetimebuffer = ^s.pausetimebuffer
			s.pausetime = s.pausetimebuffer
		}
		// In Mugen 1.1, few global AssertSpecial flags persist during pauses. Seemingly only TimerFreeze
		if s.supertime <= 0 && s.pausetime <= 0 {
			s.specialFlag = 0
		} else {
			// These flags persist even during pauses
			// "Intro" seems to have been deliberately added. Does not persist in Mugen 1.1
			// "NoKOSlow" added to facilitate custom slowdown. In Mugen that flag only needs to be asserted in first frame of KO slowdown
			s.specialFlag = (s.specialFlag&GSF_intro | s.specialFlag&GSF_nokoslow | s.specialFlag&GSF_timerfreeze)
		}
		if s.superanim != nil {
			s.superanim.Action()
		}
		s.charList.action()
		s.nomusic = s.gsf(GSF_nomusic) && !sys.postMatchFlg
	}

	// This function runs every tick
	// It should be placed between "tick frame" and "tick next frame"
	s.charUpdate()

	// Update lifebars
	// This must happen before hit detection for accurate display
	// Allows a combo to still end if a character is hit in the same frame where it exits movetype H
	s.lifebar.step()
	if s.tickNextFrame() {
		s.globalCollision() // This could perhaps happen during "tick frame" instead? Would need more testing
		s.charList.tick()
	}

	// Run camera
	x, y, scl = s.cam.action(x, y, scl, s.supertime > 0 || s.pausetime > 0)

	// Skip character intros on button press and play the shutter effect
	if s.tickNextFrame() {
		if s.lifebar.ro.current < 1 && !s.introSkipped {
			if s.shuttertime > 0 ||
				// Checking the intro flag prevents skipping intros when they don't exist
				s.anyButton() && s.gsf(GSF_intro) && !s.gsf(GSF_roundnotskip) && s.intro > s.lifebar.ro.ctrl_time {
				s.shuttertime++
				// Do the actual skipping in the frame when the "shutter" effect is closed
				if s.shuttertime == s.lifebar.ro.shutter_time {
					// SkipRoundDisplay and SkipFightDisplay flags must be preserved during intro skip frame
					skipround := (s.specialFlag&GSF_skiprounddisplay | s.specialFlag&GSF_skipfightdisplay)
					s.resetGblEffect()
					s.specialFlag = skipround
					s.fadeintime = 0
					s.intro = s.lifebar.ro.ctrl_time
					for i, p := range s.chars {
						if len(p) > 0 {
							s.clearPlayerAssets(i, false)
							p[0].posReset()
							p[0].selfState(0, -1, -1, 0, "")
						}
					}
					s.introSkipped = true
				}
			}
		} else {
			if s.shuttertime > 0 {
				s.shuttertime--
			}
		}
	}

	if !s.cam.ZoomEnable {
		// Lower the precision to prevent errors in Pos X.
		x = float32(math.Ceil(float64(x)*4-0.5) / 4)
	}
	s.cam.Update(scl, x, y)
	s.xmin = s.cam.ScreenPos[0] + s.cam.Offset[0] + s.screenleft
	s.xmax = s.cam.ScreenPos[0] + s.cam.Offset[0] +
		float32(s.gameWidth)/s.cam.Scale - s.screenright
	if s.xmin > s.xmax {
		s.xmin = (s.xmin + s.xmax) / 2
		s.xmax = s.xmin
	}
	if AbsF(s.cam.maxRight-s.xmax) < 0.0001 {
		s.xmax = s.cam.maxRight
	}
	if AbsF(s.cam.minLeft-s.xmin) < 0.0001 {
		s.xmin = s.cam.minLeft
	}
	s.charList.xScreenBound()
	// Superpause effect
	if s.superanim != nil {
		s.spritesLayer1.add(&SprData{
			anim:         s.superanim,
			fx:           &s.superpmap,
			pos:          s.superpos,
			scl:          s.superscale,
			alpha:        [2]int32{-1},
			priority:     5,
			rot:          Rotation{},
			screen:       false,
			undarken:     true,
			oldVer:       s.cgi[s.superplayerno].mugenver[0] != 1,
			facing:       1,
			airOffsetFix: [2]float32{1, 1},
			projection:   0,
			fLength:      0,
			window:       [4]float32{0, 0, 0, 0},
		})
		if s.superanim.loopend {
			s.superanim = nil // Not allowed to loop
		}
	}
	for i := range s.projs {
		for j := range s.projs[i] {
			if s.projs[i][j].id >= 0 {
				s.projs[i][j].cueDraw(s.cgi[i].mugenver[0] != 1)
			}
		}
	}
	s.charList.cueDraw()
	explUpdate := func(edl *[len(s.chars)][]int, drop bool) {
		for i, el := range *edl {
			for j := len(el) - 1; j >= 0; j-- {
				if el[j] >= 0 {
					s.explods[i][el[j]].update(s.cgi[i].mugenver[0] != 1, i)
					if s.explods[i][el[j]].id == IErr {
						if drop {
							el = append(el[:j], el[j+1:]...)
							(*edl)[i] = el
						} else {
							el[j] = -1
						}
					}
				}
			}
		}
	}
	explUpdate(&s.explodsLayerN1, true)
	explUpdate(&s.explodsLayer0, true)
	explUpdate(&s.explodsLayer1, false)
	// Adjust game speed
	if s.tickNextFrame() {
		spd := (60 + s.cfg.Options.GameSpeed*5) / float32(s.cfg.Config.Framerate) * s.accel
		// KO slowdown
		s.slowtimeTrigger = 0
		if s.intro < 0 && s.time != 0 && s.slowtime > 0 {
			if !s.gsf(GSF_nokoslow) {
				spd *= s.lifebar.ro.slow_speed
				if s.slowtime < s.lifebar.ro.slow_fadetime {
					spd += (float32(1) - s.lifebar.ro.slow_speed) * float32(s.lifebar.ro.slow_fadetime-s.slowtime) / float32(s.lifebar.ro.slow_fadetime)
				}
			}
			s.slowtimeTrigger = s.slowtime
			s.slowtime--
		}
		// Outside match or while frame stepping
		if s.postMatchFlg || s.step {
			spd = 1
		}
		s.turbo = spd
	}
	s.tickSound()
	return
}

func (s *System) draw(x, y, scl float32) {
	ecol := uint32(s.envcol[2]&0xff | s.envcol[1]&0xff<<8 |
		s.envcol[0]&0xff<<16)
	s.brightnessOld = s.brightness
	s.brightness = 0x100 >> uint(Btoi(s.supertime > 0 && s.superdarken))
	bgx, bgy := x/s.stage.localscl, y/s.stage.localscl
	//fade := func(rect [4]int32, color uint32, alpha int32) {
	//	FillRect(rect, color, alpha>>uint(Btoi(s.clsnDisplay))+Btoi(s.clsnDisplay)*128)
	//}
	if s.envcol_time == 0 {
		c := uint32(0)

		// Draw stage background fill if stage is disabled
		if s.gsf(GSF_nobg) {
			if s.allPalFX.enable {
				var rgb [3]int32
				if s.allPalFX.eInvertall {
					rgb = [...]int32{0xff, 0xff, 0xff}
				}
				for i, v := range rgb {
					rgb[i] = Clamp((v+s.allPalFX.eAdd[i])*s.allPalFX.eMul[i]>>8, 0, 0xff)
				}
				c = uint32(rgb[2] | rgb[1]<<8 | rgb[0]<<16)
			}
			FillRect(s.scrrect, c, 0xff)
		}

		// Draw normal stage background fill and elements with layerNo == -1
		if !s.gsf(GSF_nobg) {
			if s.stage.debugbg {
				FillRect(s.scrrect, 0xff00ff, 0xff)
			} else {
				c = uint32(s.stage.bgclearcolor[2]&0xff | s.stage.bgclearcolor[1]&0xff<<8 | s.stage.bgclearcolor[0]&0xff<<16)
				FillRect(s.scrrect, c, 0xff)
			}
			if s.stage.ikemenver[0] != 0 || s.stage.ikemenver[1] != 0 { // This layer did not render in Mugen
				s.stage.draw(-1, bgx, bgy, scl)
			}
		}

		// Draw reflections on layer -1
		if !s.gsf(GSF_globalnoshadow) {
			if s.stage.reflection.intensity > 0 && s.stage.reflectionlayerno < 0 {
				s.shadows.drawReflection(x, y, scl*s.cam.BaseScale())
			}
		}

		// Draw character sprites with layerNo == -1
		s.spritesLayerN1.draw(x, y, scl*s.cam.BaseScale())

		// Draw stage elements with layerNo == 0
		if !s.gsf(GSF_nobg) {
			s.stage.draw(0, bgx, bgy, scl)
		}

		// Draw character sprites with special under flag
		s.spritesLayerU.draw(x, y, scl*s.cam.BaseScale())

		// Draw shadows
		// Draw reflections on layer 0
		// TODO: Make shadows render in same layers as their sources?
		if !s.gsf(GSF_globalnoshadow) {
			if s.stage.reflection.intensity > 0 && s.stage.reflectionlayerno >= 0 {
				s.shadows.drawReflection(x, y, scl*s.cam.BaseScale())
			}
			s.shadows.draw(x, y, scl*s.cam.BaseScale())
		}

		//off := s.envShake.getOffset()
		//yofs, yofs2 := float32(s.gameHeight), float32(0)
		//if scl > 1 && s.cam.verticalfollow > 0 {
		//	yofs = s.cam.screenZoff + float32(s.gameHeight-240)
		//	yofs2 = (240 - s.cam.screenZoff) * (1 - 1/scl)
		//}
		//yofs *= 1/scl - 1
		//rect := s.scrrect
		//if off < (yofs-y+s.cam.boundH)*scl {
		//	rect[3] = (int32(math.Ceil(float64(((yofs-y+s.cam.boundH)*scl-off)*
		//		float32(s.scrrect[3])))) + s.gameHeight - 1) / s.gameHeight
		//	fade(rect, 0, 255)
		//}
		//if off > (-y+yofs2)*scl {
		//	rect[3] = (int32(math.Ceil(float64(((y-yofs2)*scl+off)*
		//		float32(s.scrrect[3])))) + s.gameHeight - 1) / s.gameHeight
		//	rect[1] = s.scrrect[3] - rect[3]
		//	fade(rect, 0, 255)
		//}
		//bl, br := MinF(x, s.cam.boundL), MaxF(x, s.cam.boundR)
		//xofs := float32(s.gameWidth) * (1/scl - 1) / 2
		//rect = s.scrrect
		//if x-xofs < bl {
		//	rect[2] = (int32(math.Ceil(float64((bl-(x-xofs))*scl*
		//		float32(s.scrrect[2])))) + s.gameWidth - 1) / s.gameWidth
		//	fade(rect, 0, 255)
		//}
		//if x+xofs > br {
		//	rect[2] = (int32(math.Ceil(float64(((x+xofs)-br)*scl*
		//		float32(s.scrrect[2])))) + s.gameWidth - 1) / s.gameWidth
		//	rect[0] = s.scrrect[2] - rect[2]
		//	fade(rect, 0, 255)
		//}

		// Draw lifebar layers -1 and 0
		s.lifebar.draw(-1)
		s.lifebar.draw(0)
	}
	// Draw EnvColor effect
	if s.envcol_time != 0 {
		FillRect(s.scrrect, ecol, 255)
	}

	// Draw character sprites in layer 0
	if s.envcol_time == 0 || s.envcol_under {
		s.spritesLayer0.draw(x, y, scl*s.cam.BaseScale())
		if s.envcol_time == 0 && !s.gsf(GSF_nofg) {
			s.stage.draw(1, bgx, bgy, scl)
		}
	}

	// Draw lifebar layer 1
	s.lifebar.draw(1)

	// Draw character sprites in layer 1 (old "ontop")
	s.spritesLayer1.draw(x, y, scl*s.cam.BaseScale())

	// Draw lifebar layer 2
	s.lifebar.draw(2)
}

func (s *System) drawTop() {
	BlendReset()
	fade := func(rect [4]int32, color uint32, alpha int32) {
		FillRect(rect, color, alpha>>uint(Btoi(s.clsnDisplay))+Btoi(s.clsnDisplay)*128)
	}
	fadeout := s.intro + s.lifebar.ro.over_waittime + s.lifebar.ro.over_time
	if fadeout == s.lifebar.ro.fadeout_time-1 && len(s.cfg.Common.Lua) > 0 && s.matchOver() && !s.dialogueFlg {
		for _, p := range s.chars {
			if len(p) > 0 && len(p[0].dialogue) > 0 {
				s.lifebar.ro.current = 3
				s.dialogueFlg = true
				break
			}
		}
	}
	if s.fadeintime > 0 {
		fade(s.scrrect, s.lifebar.ro.fadein_col, 256*s.fadeintime/s.lifebar.ro.fadein_time)
		if s.tickFrame() {
			s.fadeintime--
		}
	} else if s.fadeouttime > 0 && fadeout < s.lifebar.ro.fadeout_time-1 && !s.dialogueFlg {
		fade(s.scrrect, s.lifebar.ro.fadeout_col, 256*(s.lifebar.ro.fadeout_time-s.fadeouttime)/s.lifebar.ro.fadeout_time)
		if s.tickFrame() {
			s.fadeouttime--
		}
	} else if s.clsnDisplay && s.cfg.Debug.ClsnDarken {
		fade(s.scrrect, 0, 0)
	}
	if s.shuttertime > 0 {
		rect := s.scrrect
		rect[3] = s.shuttertime * ((s.scrrect[3] + 1) >> 1) / s.lifebar.ro.shutter_time
		fade(rect, s.lifebar.ro.shutter_col, 255)
		rect[1] = s.scrrect[3] - rect[3]
		fade(rect, s.lifebar.ro.shutter_col, 255)
	}
	s.brightness = s.brightnessOld
	// Draw Clsn boxes
	if s.clsnDisplay {
		s.clsnSpr.Pal[0] = 0xff0000ff
		s.debugc1hit.draw(0x3feff)
		s.clsnSpr.Pal[0] = 0xff0040c0
		s.debugc1rev.draw(0x3feff)
		s.clsnSpr.Pal[0] = 0xff000080
		s.debugc1not.draw(0x3feff)
		s.clsnSpr.Pal[0] = 0xffff0000
		s.debugc2.draw(0x3feff)
		s.clsnSpr.Pal[0] = 0xff808000
		s.debugc2hb.draw(0x3feff)
		s.clsnSpr.Pal[0] = 0xff004000
		s.debugc2mtk.draw(0x3feff)
		s.clsnSpr.Pal[0] = 0xffc00040
		s.debugc2grd.draw(0x3feff)
		s.clsnSpr.Pal[0] = 0xff404040
		s.debugc2stb.draw(0x3feff)
		s.clsnSpr.Pal[0] = 0xff303030
		s.debugcsize.draw(0x3feff)
		s.clsnSpr.Pal[0] = 0xffffffff
		s.debugch.draw(0x3feff)
	}
}

func (s *System) drawDebugText() {
	put := func(x, y *float32, txt string) {
		for txt != "" {
			w, drawTxt := int32(0), ""
			for i, r := range txt {
				w += s.debugFont.fnt.CharWidth(r, 0) + s.debugFont.fnt.Spacing[0]
				if w > s.scrrect[2] {
					drawTxt, txt = txt[:i], txt[i:]
					break
				}
			}
			if drawTxt == "" {
				drawTxt, txt = txt, ""
			}
			*y += float32(s.debugFont.fnt.Size[1]) * s.debugFont.yscl / s.heightScale
			s.debugFont.fnt.Print(drawTxt, *x, *y, s.debugFont.xscl/s.widthScale,
				s.debugFont.yscl/s.heightScale, 0, 0, 1, &s.scrrect,
				s.debugFont.palfx, s.debugFont.frgba)
		}
	}
	if s.debugDisplay {
		// Player Info on top of screen
		x := (320-float32(s.gameWidth))/2 + 1
		y := 240 - float32(s.gameHeight)
		if s.statusLFunc != nil {
			s.debugFont.SetColor(255, 255, 255)
			for i, p := range s.chars {
				if len(p) > 0 {
					top := s.luaLState.GetTop()
					if s.luaLState.CallByParam(lua.P{Fn: s.statusLFunc, NRet: 1,
						Protect: true}, lua.LNumber(i+1)) == nil {
						l, ok := s.luaLState.Get(-1).(lua.LString)
						if ok && len(l) > 0 {
							put(&x, &y, string(l))
						}
					}
					s.luaLState.SetTop(top)
				}
			}
		}
		// Console
		y = MaxF(y, 48+240-float32(s.gameHeight))
		s.debugFont.SetColor(255, 255, 255)
		for _, s := range s.consoleText {
			put(&x, &y, s)
		}
		// Data
		y = float32(s.gameHeight) - float32(s.debugFont.fnt.Size[1])*sys.debugFont.yscl/s.heightScale*
			(float32(len(s.listLFunc))+float32(s.cfg.Debug.ClipboardRows)) - 1*s.heightScale
		pn := s.debugRef[0]
		hn := s.debugRef[1]
		if pn >= len(s.chars) || hn >= len(s.chars[pn]) {
			s.debugRef[0] = 0
			s.debugRef[1] = 0
		}
		s.debugWC = s.chars[s.debugRef[0]][s.debugRef[1]]
		for i, f := range s.listLFunc {
			if f != nil {
				if i == 1 {
					s.debugFont.SetColor(199, 199, 219)
				} else if (i == 2 && s.debugWC.animPN != s.debugWC.playerNo) ||
					(i == 3 && s.debugWC.ss.sb.playerNo != s.debugWC.playerNo) {
					s.debugFont.SetColor(255, 255, 127)
				} else {
					s.debugFont.SetColor(255, 255, 255)
				}
				top := s.luaLState.GetTop()
				if s.luaLState.CallByParam(lua.P{Fn: f, NRet: 1,
					Protect: true}) == nil {
					s, ok := s.luaLState.Get(-1).(lua.LString)
					if ok && len(s) > 0 {
						if i == 1 && (sys.debugWC == nil || sys.debugWC.csf(CSF_destroy)) {
							put(&x, &y, string(s)+" disabled")
							break
						}
						put(&x, &y, string(s))
					}
				}
				s.luaLState.SetTop(top)
			}
		}
		// Clipboard
		s.debugFont.SetColor(255, 255, 255)
		for _, s := range s.debugWC.clipboardText {
			put(&x, &y, s)
		}
	}
	// Draw Clsn text
	// Unlike Mugen, this is drawn separately from the Clsn boxes themselves, making debug more flexible
	//if s.clsnDisplay {
	for _, t := range s.clsnText {
		s.debugFont.SetColor(t.r, t.g, t.b)
		s.debugFont.fnt.Print(t.text, t.x, t.y, s.debugFont.xscl/s.widthScale,
			s.debugFont.yscl/s.heightScale, 0, 0, 0, &s.scrrect,
			s.debugFont.palfx, s.debugFont.frgba)
	}
	//}
}

// Starts and runs gameplay
// Called to start each match, on hard reset with shift+F4, and
// at the start of any round where a new character tags in for turns mode
func (s *System) fight() (reload bool) {
	// Reset variables
	s.gameTime, s.paused, s.accel = 0, false, 1
	s.aiInput = [len(s.aiInput)]AiInput{}
	if sys.netConnection != nil {
		s.clsnDisplay = false
		s.debugDisplay = false
		s.lifebarDisplay = true
	}

	// Defer resetting variables on return
	defer func() {
		s.oldNextAddTime = 1
		s.nomusic = false
		s.allPalFX.clear()
		s.allPalFX.enable = false
		for i, p := range s.chars {
			if len(p) > 0 {
				s.clearPlayerAssets(i, s.matchOver() || (s.tmode[i&1] == TM_Turns && p[0].life <= 0))
			}
		}
		s.wincnt.update()
	}()

	var oldStageVars Stage
	oldStageVars.copyStageVars(s.stage) // NOTE: This save and restore of stage variables makes ModifyStageVar not persist. Maybe that should not be the case?

	// Vars to use in copyVar backup
	var life, lifeMax, power, powerMax [len(s.chars)]int32
	var guardPoints, guardPointsMax, dizzyPoints, dizzyPointsMax, redLife [len(s.chars)]int32
	var teamside [len(s.chars)]int
	var cnsvar [len(s.chars)]map[int32]int32
	var cnsfvar [len(s.chars)]map[int32]float32
	var mapArray [len(s.chars)]map[string]float32
	var dialogue [len(s.chars)][]string
	var remapSpr [len(s.chars)]RemapPreset

	// Anonymous function to assign initial character values
	// ModifyPlayer parameters should ideally also be reset here
	copyVar := func(pn int) {
		life[pn] = s.chars[pn][0].life
		lifeMax[pn] = s.chars[pn][0].lifeMax
		power[pn] = s.chars[pn][0].power
		powerMax[pn] = s.chars[pn][0].powerMax
		guardPoints[pn] = s.chars[pn][0].guardPoints
		guardPointsMax[pn] = s.chars[pn][0].guardPointsMax
		dizzyPoints[pn] = s.chars[pn][0].dizzyPoints
		dizzyPointsMax[pn] = s.chars[pn][0].dizzyPointsMax
		redLife[pn] = s.chars[pn][0].redLife
		teamside[pn] = s.chars[pn][0].teamside
		cnsvar[pn] = make(map[int32]int32)
		for k, v := range s.chars[pn][0].cnsvar {
			cnsvar[pn][k] = v
		}
		cnsfvar[pn] = make(map[int32]float32)
		for k, v := range s.chars[pn][0].cnsfvar {
			cnsfvar[pn][k] = v
		}
		mapArray[pn] = make(map[string]float32)
		for k, v := range s.chars[pn][0].mapArray {
			mapArray[pn][k] = v
		}
		copy(dialogue[pn], s.chars[pn][0].dialogue[:])
		remapSpr[pn] = make(RemapPreset)
		for k, v := range s.chars[pn][0].remapSpr {
			remapSpr[pn][k] = v
		}
	}

	s.debugWC = sys.chars[0][0]
	debugInput := func() {
		select {
		case cl := <-s.commandLine:
			if err := s.luaLState.DoString(cl); err != nil {
				s.errLog.Println(err.Error())
			}
		default:
		}
	}

	// Synchronize with external inputs (netplay, replays, etc)
	if err := s.synchronize(); err != nil {
		s.errLog.Println(err.Error())
		s.esc = true
	}
	if s.netConnection != nil {
		defer s.netConnection.Stop()
	}
	s.wincnt.init()

	// Initialize super meter values, and max power for teams sharing meter
	var level [len(s.chars)]int32
	for i, p := range s.chars {
		if len(p) > 0 && p[0].teamside != -1 {
			p[0].prepareNextRound()
			level[i] = s.wincnt.getLevel(i)
			if s.cfg.Options.Team.PowerShare {
				pmax := Max(s.cgi[i&1].data.power, s.cgi[i].data.power)
				for j := i & 1; j < MaxSimul*2; j += 2 {
					if len(s.chars[j]) > 0 {
						s.chars[j][0].powerMax = pmax
					}
				}
			}
		}
	}
	minlv, maxlv := level[0], level[0]
	for i, lv := range level[1:] {
		if len(s.chars[i+1]) > 0 {
			minlv = Min(minlv, lv)
			maxlv = Max(maxlv, lv)
		}
	}
	if minlv > 0 {
		for i := range level {
			level[i] -= minlv
		}
	} else if maxlv < 0 {
		for i := range level {
			level[i] -= maxlv
		}
	}

	// Initialize each character
	lvmul := math.Pow(2, 1.0/12)
	for i, p := range s.chars {
		if len(p) > 0 {
			// Get max life, and adjust based on team mode
			var lm float32
			if p[0].ocd().lifeMax != -1 {
				lm = float32(p[0].ocd().lifeMax) * p[0].ocd().lifeRatio * s.cfg.Options.Life / 100
			} else {
				lm = float32(p[0].gi().data.life) * p[0].ocd().lifeRatio * s.cfg.Options.Life / 100
			}
			if p[0].teamside != -1 {
				switch s.tmode[i&1] {
				case TM_Single:
					switch s.tmode[(i+1)&1] {
					case TM_Simul, TM_Tag:
						lm *= s.cfg.Options.Team.SingleVsTeamLife / 100
					case TM_Turns:
						if s.numTurns[(i+1)&1] < s.matchWins[(i+1)&1] && s.cfg.Options.Team.LifeShare {
							lm = lm * float32(s.numTurns[(i+1)&1]) /
								float32(s.matchWins[(i+1)&1])
						}
					}
				case TM_Simul, TM_Tag:
					switch s.tmode[(i+1)&1] {
					case TM_Simul, TM_Tag:
						if s.numSimul[(i+1)&1] < s.numSimul[i&1] && s.cfg.Options.Team.LifeShare {
							lm = lm * float32(s.numSimul[(i+1)&1]) / float32(s.numSimul[i&1])
						}
					case TM_Turns:
						if s.numTurns[(i+1)&1] < s.numSimul[i&1]*s.matchWins[(i+1)&1] && s.cfg.Options.Team.LifeShare {
							lm = lm * float32(s.numTurns[(i+1)&1]) /
								float32(s.numSimul[i&1]*s.matchWins[(i+1)&1])
						}
					default:
						if s.cfg.Options.Team.LifeShare {
							lm /= float32(s.numSimul[i&1])
						}
					}
				case TM_Turns:
					switch s.tmode[(i+1)&1] {
					case TM_Single:
						if s.matchWins[i&1] < s.numTurns[i&1] && s.cfg.Options.Team.LifeShare {
							lm = lm * float32(s.matchWins[i&1]) / float32(s.numTurns[i&1])
						}
					case TM_Simul, TM_Tag:
						if s.numSimul[(i+1)&1]*s.matchWins[i&1] < s.numTurns[i&1] && s.cfg.Options.Team.LifeShare {
							lm = lm * s.cfg.Options.Team.SingleVsTeamLife / 100 *
								float32(s.numSimul[(i+1)&1]*s.matchWins[i&1]) /
								float32(s.numTurns[i&1])
						}
					case TM_Turns:
						if s.numTurns[(i+1)&1] < s.numTurns[i&1] && s.cfg.Options.Team.LifeShare {
							lm = lm * float32(s.numTurns[(i+1)&1]) / float32(s.numTurns[i&1])
						}
					}
				}
			}
			foo := math.Pow(lvmul, float64(-level[i]))
			p[0].lifeMax = Max(1, int32(math.Floor(foo*float64(lm))))

			if p[0].roundsExisted() > 0 {
				// If character already existed for a round, presumably because of turns mode, just update life
				p[0].life = Min(p[0].lifeMax, int32(math.Ceil(foo*float64(p[0].life))))
			} else if s.round == 1 || s.tmode[i&1] == TM_Turns {
				// If round 1 or a new character in Turns mode, initialize values
				if p[0].ocd().life != -1 {
					p[0].life = Clamp(p[0].ocd().life, 0, p[0].lifeMax)
					p[0].redLife = p[0].life
				} else {
					p[0].life = p[0].lifeMax
					p[0].redLife = p[0].lifeMax
				}
				if s.round == 1 {
					if s.maxPowerMode {
						p[0].power = p[0].powerMax
					} else if p[0].ocd().power != -1 {
						p[0].power = Clamp(p[0].ocd().power, 0, p[0].powerMax)
					} else if !sys.consecutiveRounds || sys.consecutiveWins[0] == 0 {
						p[0].power = 0
					}
				}
				p[0].power = Clamp(p[0].power, 0, p[0].powerMax) // Because of previous partner in Turns mode
				p[0].mapArray = make(map[string]float32)
				for k, v := range p[0].mapDefault {
					p[0].mapArray[k] = v
				}
				p[0].dialogue = []string{}
				p[0].remapSpr = make(RemapPreset)
			}

			if p[0].ocd().guardPoints != -1 {
				p[0].guardPoints = Clamp(p[0].ocd().guardPoints, 0, p[0].guardPointsMax)
			} else {
				p[0].guardPoints = p[0].guardPointsMax
			}
			if p[0].ocd().dizzyPoints != -1 {
				p[0].dizzyPoints = Clamp(p[0].ocd().dizzyPoints, 0, p[0].dizzyPointsMax)
			} else {
				p[0].dizzyPoints = p[0].dizzyPointsMax
			}
			copyVar(i)
		}
	}

	oldWins, oldDraws := s.wins, s.draws
	oldTeamLeader := s.teamLeader

	// Anonymous function to reset values, called at the start of each round
	reset := func() {
		s.wins, s.draws = oldWins, oldDraws
		s.teamLeader = oldTeamLeader
		for i, p := range s.chars {
			if len(p) > 0 {
				p[0].life = life[i]
				p[0].lifeMax = lifeMax[i]
				p[0].power = power[i]
				p[0].powerMax = powerMax[i]
				p[0].guardPoints = guardPoints[i]
				p[0].guardPointsMax = guardPointsMax[i]
				p[0].dizzyPoints = dizzyPoints[i]
				p[0].dizzyPointsMax = dizzyPointsMax[i]
				p[0].redLife = redLife[i]
				p[0].teamside = teamside[i]
				p[0].cnsvar = make(map[int32]int32)
				for k, v := range cnsvar[i] {
					p[0].cnsvar[k] = v
				}
				p[0].cnsfvar = make(map[int32]float32)
				for k, v := range cnsfvar[i] {
					p[0].cnsfvar[k] = v
				}
				p[0].cnssysvar = make(map[int32]int32) // SysVars never persist
				p[0].cnssysfvar = make(map[int32]float32)
				p[0].mapArray = make(map[string]float32)
				for k, v := range mapArray[i] {
					p[0].mapArray[k] = v
				}
				copy(p[0].dialogue[:], dialogue[i])
				p[0].remapSpr = make(RemapPreset)
				for k, v := range remapSpr[i] {
					p[0].remapSpr[k] = v
				}
			}
		}
		s.stage.copyStageVars(&oldStageVars)
		s.resetFrameTime()
		s.nextRound()
		s.roundResetFlg, s.introSkipped = false, false
		s.reloadFlg, s.reloadStageFlg, s.reloadLifebarFlg = false, false, false
		s.runMainThreadTask()
		gfx.Await()
	}
	reset()

	// Loop until end of match
	fin := false
	didTryLoadBGM := false
	for !s.endMatch {
		// default bgm playback, used only in Quick VS or if externalized Lua implementaion is disabled
		if s.round == 1 && (s.gameMode == "" || len(sys.cfg.Common.Lua) == 0) && sys.stage.stageTime > 0 && !didTryLoadBGM {
			// Need to search first
			LoadFile(&s.stage.bgmusic, []string{s.stage.def, "", "sound/"}, func(path string) error {
				s.bgm.Open(path, 1, int(s.stage.bgmvolume), int(s.stage.bgmloopstart), int(s.stage.bgmloopend), int(s.stage.bgmstartposition), s.stage.bgmfreqmul, -1)
				didTryLoadBGM = true
				return nil
			})
		}
		s.step = false
		for _, v := range s.shortcutScripts {
			if v.Activate {
				if err := s.luaLState.DoString(v.Script); err != nil {
					s.errLog.Println(err.Error())
				}
			}
		}

		// If next round
		if s.roundOver() && !fin {
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
					tmp.RawSetString("aiLevel", lua.LNumber(p[0].getAILevel()))
					tmp.RawSetString("palno", lua.LNumber(p[0].gi().palno))
					tmp.RawSetString("ratiolevel", lua.LNumber(p[0].ocd().ratioLevel))
					tmp.RawSetString("win", lua.LBool(p[0].win()))
					tmp.RawSetString("winKO", lua.LBool(p[0].winKO()))
					tmp.RawSetString("winTime", lua.LBool(p[0].winTime()))
					tmp.RawSetString("winPerfect", lua.LBool(p[0].winPerfect()))
					tmp.RawSetString("winSpecial", lua.LBool(p[0].winType(WT_Special)))
					tmp.RawSetString("winHyper", lua.LBool(p[0].winType(WT_Hyper)))
					tmp.RawSetString("drawgame", lua.LBool(p[0].drawgame()))
					tmp.RawSetString("ko", lua.LBool(p[0].scf(SCF_ko)))
					tmp.RawSetString("over_ko", lua.LBool(p[0].scf(SCF_over_ko)))
					tbl_roundNo.RawSetInt(p[0].playerNo+1, tmp)
				}
			}
			s.matchData.RawSetInt(int(s.round-1), tbl_roundNo)
			s.scoreRounds = append(s.scoreRounds, [2]float32{s.lifebar.sc[0].scorePoints, s.lifebar.sc[1].scorePoints})
			oldTeamLeader = s.teamLeader

			if !s.matchOver() && (s.tmode[0] != TM_Turns || s.chars[0][0].win()) &&
				(s.tmode[1] != TM_Turns || s.chars[1][0].win()) {
				// Prepare for the next round
				for i, p := range s.chars {
					if len(p) > 0 {
						if s.tmode[i&1] != TM_Turns || !p[0].win() {
							p[0].life = p[0].lifeMax
						} else if p[0].life <= 0 {
							p[0].life = 1
						}
						p[0].redLife = p[0].life // TODO: This doesn't truly need to be hardcoded
						copyVar(i)
					}
				}
				oldWins, oldDraws = s.wins, s.draws
				oldStageVars.copyStageVars(s.stage)
				reset()
			} else {
				// End match, or prepare for a new character in turns mode
				for i, tm := range s.tmode {
					if s.chars[i][0].win() || !s.chars[i][0].lose() && tm != TM_Turns {
						for j := i; j < len(s.chars); j += 2 {
							if len(s.chars[j]) > 0 {
								if s.chars[j][0].win() {
									s.chars[j][0].life = Max(1, int32(math.Ceil(math.Pow(lvmul,
										float64(level[i]))*float64(s.chars[j][0].life))))
								} else {
									s.chars[j][0].life = Max(1, s.cgi[j].data.life)
								}
							}
						}
					}
				}
				// If match isn't over, presumably this is turns mode,
				// so break to restart fight for the next character
				if !s.matchOver() {
					break
				}

				// Otherwise match is over
				s.postMatchFlg = true
				fin = true
			}
		}

		s.bgPalFX.step()
		s.stage.action()

		// Update game state
		s.action()

		debugInput()
		if !s.addFrameTime(s.turbo) {
			if !s.eventUpdate() {
				return false
			}
			continue
		}

		// F4 pressed to restart round
		if s.roundResetFlg && !s.postMatchFlg {
			sys.paused = false
			reset()
		}
		// Shift+F4 pressed to restart match
		if s.reloadFlg {
			return true
		}

		// Render frame
		if !s.frameSkip {
			x, y, scl := s.cam.Pos[0], s.cam.Pos[1], s.cam.Scale/s.cam.BaseScale()
			dx, dy, dscl := x, y, scl
			if s.enableZoomtime > 0 {
				if !s.debugPaused() {
					s.zoomPosXLag += ((s.zoomPos[0] - s.zoomPosXLag) * (1 - s.zoomlag))
					s.zoomPosYLag += ((s.zoomPos[1] - s.zoomPosYLag) * (1 - s.zoomlag))
					s.drawScale = s.drawScale / (s.drawScale + (s.zoomScale*scl-s.drawScale)*s.zoomlag) * s.zoomScale * scl
				}
				if s.zoomStageBound {
					dscl = MaxF(s.cam.MinScale, s.drawScale/s.cam.BaseScale())
					if s.zoomCameraBound {
						dx = x + ClampF(s.zoomPosXLag/scl, -s.cam.halfWidth/scl*2*(1-1/s.zoomScale), s.cam.halfWidth/scl*2*(1-1/s.zoomScale))
					} else {
						dx = x + s.zoomPosXLag/scl
					}
					dx = s.cam.XBound(dscl, dx)
				} else {
					dscl = s.drawScale / s.cam.BaseScale()
					dx = x + s.zoomPosXLag/scl
				}
				dy = y + s.zoomPosYLag/scl
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
		// Render top elements such as fade effects
		if !s.frameSkip {
			s.drawTop()
		}
		// Lua code is executed after drawing the fade effects, so that the menus are on top of them
		for _, key := range SortedKeys(sys.cfg.Common.Lua) {
			for _, v := range sys.cfg.Common.Lua[key] {
				if err := s.luaLState.DoString(v); err != nil {
					s.luaLState.RaiseError(err.Error())
				}
			}
		}
		// Render debug elements
		if !s.frameSkip && s.debugDisplay {
			s.drawDebugText()
		}
		// Break if finished
		if fin && (!s.postMatchFlg || len(sys.cfg.Common.Lua) == 0) {
			break
		}

		// Update system; break if update returns false (game ended)
		if !s.update() {
			break
		}

		// If end match selected from menu/end of attract mode match/etc
		if s.endMatch {
			s.esc = true
		} else if s.esc {
			s.endMatch = s.netConnection != nil || len(sys.cfg.Common.Lua) == 0
		}
	}

	return false
}

// Code responsible for updating the 'autolevel.save' file.
// This file stores win/loss data for each character per palette, which is used by 'randomtest.lua'.
// The 'randomtest.lua' script reads this data to generate AI ranks and adjust the difficulty of opponents in random battles.

type wincntMap map[string][]int32 // Map of character definitions to their win counts per palette

// Initializes the win count map by reading from 'autolevel.save' file
func (wm *wincntMap) init() {
	if sys.autolevel {
		b, err := os.ReadFile(sys.wincntFileName) // Read the autolevel.save file
		if err != nil {
			return
		}
		str := string(b)
		if len(str) < 3 {
			return
		}
		if str[:3] == "\ufeff" { // Remove Byte Order Mark if present
			str = str[3:]
		}
		// Converts array of strings to array of int32
		toint := func(strAry []string) (intAry []int32) {
			for _, s := range strAry {
				i, _ := strconv.ParseInt(s, 10, 32)
				intAry = append(intAry, int32(i))
			}
			return
		}
		// Parse each line in the autolevel.save file
		for _, l := range strings.Split(str, "\n") {
			tmp := strings.Split(l, ",")
			if len(tmp) >= 2 {
				item := toint(strings.Split(strings.TrimSpace(tmp[1]), " ")) // Get win counts per palette
				if len(item) < MaxPalNo {
					item = append(item, make([]int32, MaxPalNo-len(item))...) // Ensure item has MaxPalNo elements
				}
				(*wm)[tmp[0]] = item // Map character definition to win counts
			}
		}
	}
}

// Updates the win count map after a match and writes to 'autolevel.save' file
func (wm *wincntMap) update() {
	// Calculates win points based on team modes and number of simul characters
	winPoint := func(i int) int32 {
		if sys.tmode[(i+1)&1] == TM_Simul || sys.tmode[(i+1)&1] == TM_Tag {
			if sys.tmode[i&1] != TM_Simul && sys.tmode[i&1] != TM_Tag {
				return sys.numSimul[(i+1)&1]
			} else if sys.numSimul[(i+1)&1] > sys.numSimul[i&1] {
				return sys.numSimul[(i+1)&1] / sys.numSimul[i&1]
			}
		}
		return 1
	}
	// Updates win counts for winning characters
	win := func(i int) {
		item := wm.getItem(sys.cgi[i].def)
		item[sys.cgi[i].palno-1] += winPoint(i)
		wm.setItem(i, item)
	}
	// Updates win counts for losing characters
	lose := func(i int) {
		item := wm.getItem(sys.cgi[i].def)
		item[sys.cgi[i].palno-1] -= winPoint(i)
		wm.setItem(i, item)
	}
	if sys.autolevel && sys.matchOver() {
		// Iterate over all characters in the match
		for i, p := range sys.chars {
			if len(p) > 0 {
				if p[0].win() {
					win(i) // Update win counts for winners
				} else if p[0].lose() {
					lose(i) // Update win counts for losers
				}
			}
		}
		// Write updated win counts back to 'autolevel.save' file
		var str string
		for k, v := range *wm {
			str += k + ","
			for _, w := range v {
				str += fmt.Sprintf(" %v", w)
			}
			str += "\r\n"
		}
		f, err := os.Create(sys.wincntFileName)
		if err == nil {
			f.Write([]byte(str))
			chk(f.Close())
		}
	}
}

// Retrieves win counts for a character, ensuring the slice has MaxPalNo elements
func (wm wincntMap) getItem(def string) []int32 {
	lv := wm[def]
	if len(lv) < MaxPalNo {
		lv = append(lv, make([]int32, MaxPalNo-len(lv))...)
	}
	return lv
}

// Sets win counts for a character, averaging values for non-selectable palettes
func (wm wincntMap) setItem(pn int, item []int32) {
	var ave, palcnt int32 = 0, 0
	for i, v := range item {
		if sys.cgi[pn].palSelectable[i] {
			ave += v
			palcnt++
		}
	}
	if palcnt > 0 {
		ave /= palcnt
	}
	for i := range item {
		if !sys.cgi[pn].palSelectable[i] {
			item[i] = ave // Set non-selectable palettes to average value
		}
	}
	wm[sys.cgi[pn].def] = item
}

// Gets the win count (level) for a character's specific palette
func (wm wincntMap) getLevel(p int) int32 {
	return wm.getItem(sys.cgi[p].def)[sys.cgi[p].palno-1]
}

type SelectChar struct {
	def            string
	name           string
	lifebarname    string
	author         string
	sound          string
	intro          string
	ending         string
	arcadepath     string
	ratiopath      string
	movelist       string
	pal            []int32
	pal_defaults   []int32
	pal_keymap     []int32
	localcoord     int32
	portrait_scale float32
	cns_scale      [2]float32
	anims          PreloadedAnims
	sff            *Sff
	fnt            [10]*Fnt
}

func newSelectChar() *SelectChar {
	return &SelectChar{
		localcoord:     320,
		portrait_scale: 1,
		cns_scale:      [...]float32{1, 1},
		anims:          NewPreloadedAnims(),
	}
}

type SelectStage struct {
	def             string
	name            string
	attachedchardef string
	stagebgm        IniSection
	portrait_scale  float32
	anims           PreloadedAnims
	sff             *Sff
}

func newSelectStage() *SelectStage {
	return &SelectStage{portrait_scale: 1, anims: NewPreloadedAnims()}
}

type OverrideCharData struct {
	life        int32
	lifeMax     int32
	power       int32
	dizzyPoints int32
	guardPoints int32
	ratioLevel  int32
	lifeRatio   float32
	attackRatio float32
	existed     bool
}

func newOverrideCharData() *OverrideCharData {
	return &OverrideCharData{life: -1, lifeMax: -1, power: -1, dizzyPoints: -1,
		guardPoints: -1, ratioLevel: 0, lifeRatio: 1, attackRatio: 1}
}

type Select struct {
	charlist           []SelectChar
	stagelist          []SelectStage
	selected           [2][][2]int
	selectedStageNo    int
	charAnimPreload    []int32
	stageAnimPreload   []int32
	charSpritePreload  map[[2]int16]bool
	stageSpritePreload map[[2]int16]bool
	cdefOverwrite      map[int]string
	sdefOverwrite      string
	ocd                [3][]OverrideCharData
}

func newSelect() *Select {
	return &Select{selectedStageNo: -1,
		charSpritePreload: map[[2]int16]bool{[...]int16{9000, 0}: true,
			[...]int16{9000, 1}: true}, stageSpritePreload: make(map[[2]int16]bool),
		cdefOverwrite: make(map[int]string)}
}

func (s *Select) GetCharNo(i int) int {
	n := i
	if len(s.charlist) > 0 {
		n %= len(s.charlist)
		if n < 0 {
			n += len(s.charlist)
		}
	}
	return n
}

func (s *Select) GetChar(i int) *SelectChar {
	if len(s.charlist) == 0 {
		return nil
	}
	n := s.GetCharNo(i)
	return &s.charlist[n]
}

func (s *Select) SelectStage(n int) { s.selectedStageNo = n }

func (s *Select) GetStage(n int) *SelectStage {
	if len(s.stagelist) == 0 {
		return nil
	}
	n %= len(s.stagelist) + 1
	if n < 0 {
		n += len(s.stagelist) + 1
	}
	return &s.stagelist[n-1]
}

func (s *Select) addChar(def string) {
	var tstr string
	tnow := time.Now()
	defer func() {
		sys.loadTime(tnow, tstr, false, false)
	}()
	s.charlist = append(s.charlist, *newSelectChar())
	sc := &s.charlist[len(s.charlist)-1]
	def = strings.Replace(strings.TrimSpace(strings.Split(def, ",")[0]),
		"\\", "/", -1)
	tstr = fmt.Sprintf("Char added: %v", def)
	if strings.ToLower(def) == "dummyslot" {
		sc.name = "dummyslot"
		return
	}
	if strings.ToLower(def) == "randomselect" {
		sc.def, sc.name = "randomselect", "Random"
		return
	}
	idx := strings.Index(def, "/")
	if len(def) >= 4 && strings.ToLower(def[len(def)-4:]) == ".def" {
		if idx < 0 {
			sc.name = "dummyslot"
			return
		}
	} else if idx < 0 {
		def += "/" + def + ".def"
	} else {
		def += ".def"
	}
	if chk := FileExist(def); len(chk) != 0 {
		def = chk
	} else {
		if strings.ToLower(def[0:6]) != "chars/" && strings.ToLower(def[1:3]) != ":/" && (def[0] != '/' || idx > 0 && !strings.Contains(def[:idx], ":")) {
			def = "chars/" + def
		}
		if def = FileExist(def); len(def) == 0 {
			sc.name = "dummyslot"
			return
		}
	}
	str, err := LoadText(def)
	if err != nil {
		sc.name = "dummyslot"
		return
	}
	sc.def = def
	lines, i, info, files, keymap, arcade, lanInfo, lanFiles, lanKeymap, lanArcade := SplitAndTrim(str, "\n"), 0, true, true, true, true, true, true, true, true
	var cns, sprite, anim, movelist string
	var fnt [10][2]string
	for i < len(lines) {
		is, name, subname := ReadIniSection(lines, &i)
		switch name {
		case "info":
			if info {
				info = false
				var ok bool
				if sc.name, ok, _ = is.getText("displayname"); !ok {
					sc.name, _, _ = is.getText("name")
				}
				if sc.lifebarname, ok, _ = is.getText("lifebarname"); !ok {
					sc.lifebarname = sc.name
				}
				sc.author, _, _ = is.getText("author")
				sc.pal_defaults = is.readI32CsvForStage("pal.defaults")
				is.ReadI32("localcoord", &sc.localcoord)
				if ok = is.ReadF32("portraitscale", &sc.portrait_scale); !ok {
					sc.portrait_scale = 320 / float32(sc.localcoord)
				}
			}
		case fmt.Sprintf("%v.info", sys.cfg.Config.Language):
			if lanInfo {
				info = false
				lanInfo = false
				var ok bool
				if sc.name, ok, _ = is.getText("displayname"); !ok {
					sc.name, _, _ = is.getText("name")
				}
				if sc.lifebarname, ok, _ = is.getText("lifebarname"); !ok {
					sc.lifebarname = sc.name
				}
				sc.author, _, _ = is.getText("author")
				sc.pal_defaults = is.readI32CsvForStage("pal.defaults")
				is.ReadI32("localcoord", &sc.localcoord)
				if ok = is.ReadF32("portraitscale", &sc.portrait_scale); !ok {
					sc.portrait_scale = 320 / float32(sc.localcoord)
				}
			}
		case "files":
			if files {
				files = false
				cns = is["cns"]
				sprite = is["sprite"]
				anim = is["anim"]
				sc.sound = is["sound"]
				for i := 1; i <= MaxPalNo; i++ {
					if is[fmt.Sprintf("pal%v", i)] != "" {
						sc.pal = append(sc.pal, int32(i))
					}
				}
				movelist = is["movelist"]
				for i := range fnt {
					fnt[i][0] = is[fmt.Sprintf("font%v", i)]
					fnt[i][1] = is[fmt.Sprintf("fnt_height%v", i)]
				}
			}
		case fmt.Sprintf("%v.files", sys.cfg.Config.Language):
			if lanFiles {
				files = false
				lanFiles = false
				cns = is["cns"]
				sprite = is["sprite"]
				anim = is["anim"]
				sc.sound = is["sound"]
				for i := 1; i <= MaxPalNo; i++ {
					if is[fmt.Sprintf("pal%v", i)] != "" {
						sc.pal = append(sc.pal, int32(i))
					}
				}
				movelist = is["movelist"]
				for i := range fnt {
					fnt[i][0] = is[fmt.Sprintf("font%v", i)]
					fnt[i][1] = is[fmt.Sprintf("fnt_height%v", i)]
				}
			}
		case "palette ":
			if keymap &&
				len(subname) >= 6 && strings.ToLower(subname[:6]) == "keymap" {
				keymap = false
				for _, v := range [12]string{"a", "b", "c", "x", "y", "z",
					"a2", "b2", "c2", "x2", "y2", "z2"} {
					var i32 int32
					if is.ReadI32(v, &i32) {
						sc.pal_keymap = append(sc.pal_keymap, i32)
					}
				}
			}
		case fmt.Sprintf("%v.palette ", sys.cfg.Config.Language):
			if lanKeymap &&
				len(subname) >= 6 && strings.ToLower(subname[:6]) == "keymap" {
				keymap = false
				for _, v := range [12]string{"a", "b", "c", "x", "y", "z",
					"a2", "b2", "c2", "x2", "y2", "z2"} {
					var i32 int32
					if is.ReadI32(v, &i32) {
						sc.pal_keymap = append(sc.pal_keymap, i32)
					}
				}
			}
		case "arcade":
			if arcade {
				arcade = false
				sc.intro, _, _ = is.getText("intro.storyboard")
				sc.ending, _, _ = is.getText("ending.storyboard")
				sc.arcadepath, _, _ = is.getText("arcadepath")
				sc.ratiopath, _, _ = is.getText("ratiopath")
			}
		case fmt.Sprintf("%v.arcade", sys.cfg.Config.Language):
			if lanArcade {
				arcade = false
				lanArcade = false
				sc.intro, _, _ = is.getText("intro.storyboard")
				sc.ending, _, _ = is.getText("ending.storyboard")
				sc.arcadepath, _, _ = is.getText("arcadepath")
				sc.ratiopath, _, _ = is.getText("ratiopath")
			}
		}
	}
	listSpr := make(map[[2]int16]bool)
	for k := range s.charSpritePreload {
		listSpr[[...]int16{k[0], k[1]}] = true
	}
	sff := newSff()
	// read size values
	LoadFile(&cns, []string{def, "", "data/"}, func(filename string) error {
		str, err := LoadText(filename)
		if err != nil {
			return err
		}
		lines, i := SplitAndTrim(str, "\n"), 0
		for i < len(lines) {
			is, name, _ := ReadIniSection(lines, &i)
			switch name {
			case "size":
				if ok := is.ReadF32("xscale", &sc.cns_scale[0]); !ok {
					sc.cns_scale[0] = 320 / float32(sc.localcoord)
				}
				if ok := is.ReadF32("yscale", &sc.cns_scale[1]); !ok {
					sc.cns_scale[1] = 320 / float32(sc.localcoord)
				}
				return nil
			}
		}
		return nil
	})
	// preload animations
	LoadFile(&anim, []string{def, "", "data/"}, func(filename string) error {
		str, err := LoadText(filename)
		if err != nil {
			return err
		}
		lines, i := SplitAndTrim(str, "\n"), 0
		at := ReadAnimationTable(sff, &sff.palList, lines, &i)
		for _, v := range s.charAnimPreload {
			if anim := at.get(v); anim != nil {
				sc.anims.addAnim(anim, v)
				for _, fr := range anim.frames {
					listSpr[[...]int16{fr.Group, fr.Number}] = true
				}
			}
		}
		return nil
	})
	// preload portion of sff file
	fp := fmt.Sprintf("%v_preload.sff", strings.TrimSuffix(def, filepath.Ext(def)))
	if fp = FileExist(fp); len(fp) == 0 {
		fp = sprite
	}
	if len(fp) > 0 {
		LoadFile(&fp, []string{def, "", "data/"}, func(file string) error {
			var selPal []int32
			var err error
			sc.sff, selPal, err = preloadSff(file, true, listSpr)
			if err != nil {
				panic(fmt.Errorf("failed to load %v: %v\nerror preloading %v", file, err, def))
			}
			sc.anims.updateSff(sc.sff)
			for k := range s.charSpritePreload {
				sc.anims.addSprite(sc.sff, k[0], k[1])
			}
			if len(sc.pal) == 0 {
				sc.pal = selPal
			}
			return nil
		})
	} else {
		sc.sff = newSff()
		sc.anims.updateSff(sc.sff)
		for k := range s.charSpritePreload {
			sc.anims.addSprite(sc.sff, k[0], k[1])
		}
	}
	// read movelist
	if len(movelist) > 0 {
		LoadFile(&movelist, []string{def, "", "data/"}, func(file string) error {
			sc.movelist, _ = LoadText(file)
			return nil
		})
	}
	// preload fonts
	for i, f := range fnt {
		if len(f[0]) > 0 {
			LoadFile(&f[0], []string{def, sys.motifDir, "", "data/", "font/"}, func(filename string) error {
				var err error
				var height int32 = -1
				if len(f[1]) > 0 {
					height = Atoi(f[1])
				}
				if sc.fnt[i], err = loadFnt(filename, height); err != nil {
					sys.errLog.Printf("failed to load %v (char font): %v", filename, err)
				}
				return nil
			})
		}
	}
}

func (s *Select) AddStage(def string) error {
	var tstr string
	tnow := time.Now()
	defer func() {
		sys.loadTime(tnow, tstr, false, false)
	}()
	var lines []string
	if err := LoadFile(&def, []string{"", "data/"}, func(file string) error {
		str, err := LoadText(file)
		if err != nil {
			return err
		}
		lines = SplitAndTrim(str, "\n")
		return nil
	}); err != nil {
		sys.errLog.Printf("Failed to add stage, file not found: %v\n", def)
		return err
	}
	tstr = fmt.Sprintf("Stage added: %v", def)
	i, info, music, bgdef, stageinfo, lanInfo, lanMusic, lanBgdef, lanStageinfo := 0, true, true, true, true, true, true, true, true
	var spr string
	s.stagelist = append(s.stagelist, *newSelectStage())
	ss := &s.stagelist[len(s.stagelist)-1]
	ss.def = def
	for i < len(lines) {
		is, name, _ := ReadIniSection(lines, &i)
		switch name {
		case "info":
			if info {
				info = false
				var ok bool
				if ss.name, ok, _ = is.getText("displayname"); !ok {
					if ss.name, ok, _ = is.getText("name"); !ok {
						ss.name = def
					}
				}
				if err := is.LoadFile("attachedchar", []string{def, "", sys.motifDir, "data/"}, func(filename string) error {
					ss.attachedchardef = filename
					return nil
				}); err != nil {
					return nil
				}
			}
		case fmt.Sprintf("%v.info", sys.cfg.Config.Language):
			if lanInfo {
				info = false
				lanInfo = false
				var ok bool
				if ss.name, ok, _ = is.getText("displayname"); !ok {
					if ss.name, ok, _ = is.getText("name"); !ok {
						ss.name = def
					}
				}
				if err := is.LoadFile("attachedchar", []string{def, "", sys.motifDir, "data/"}, func(filename string) error {
					ss.attachedchardef = filename
					return nil
				}); err != nil {
					return nil
				}
			}
		case "music":
			if music {
				music = false
				ss.stagebgm = is
			}
		case fmt.Sprintf("%v.music", sys.cfg.Config.Language):
			if lanMusic {
				music = false
				lanMusic = false
				ss.stagebgm = is
			}
		case "bgdef":
			if bgdef {
				bgdef = false
				spr = is["spr"]
			}
		case fmt.Sprintf("%v.bgdef", sys.cfg.Config.Language):
			if lanBgdef {
				bgdef = false
				lanBgdef = false
				spr = is["spr"]
			}
		case "stageinfo":
			if stageinfo {
				stageinfo = false
				if ok := is.ReadF32("portraitscale", &ss.portrait_scale); !ok {
					localcoord := float32(320)
					is.ReadF32("localcoord", &localcoord)
					ss.portrait_scale = 320 / localcoord
				}
			}
		case fmt.Sprintf("%v.stageinfo", sys.cfg.Config.Language):
			if lanStageinfo {
				stageinfo = false
				lanStageinfo = false
				if ok := is.ReadF32("portraitscale", &ss.portrait_scale); !ok {
					localcoord := float32(320)
					is.ReadF32("localcoord", &localcoord)
					ss.portrait_scale = 320 / localcoord
				}
			}
		}
	}
	if len(s.stageSpritePreload) > 0 || len(s.stageAnimPreload) > 0 {
		listSpr := make(map[[2]int16]bool)
		for k := range s.stageSpritePreload {
			listSpr[[...]int16{k[0], k[1]}] = true
		}
		sff := newSff()
		// preload animations
		i = 0
		at := ReadAnimationTable(sff, &sff.palList, lines, &i)
		for _, v := range s.stageAnimPreload {
			if anim := at.get(v); anim != nil {
				ss.anims.addAnim(anim, v)
				for _, fr := range anim.frames {
					listSpr[[...]int16{fr.Group, fr.Number}] = true
				}
			}
		}
		// preload portion of sff file
		LoadFile(&spr, []string{def, "", "data/"}, func(file string) error {
			var err error
			ss.sff, _, err = preloadSff(file, false, listSpr)
			if err != nil {
				panic(fmt.Errorf("failed to load %v: %v\nerror preloading %v", file, err, def))
			}
			ss.anims.updateSff(ss.sff)
			for k := range s.stageSpritePreload {
				ss.anims.addSprite(ss.sff, k[0], k[1])
			}
			return nil
		})
	}
	return nil
}

func (s *Select) AddSelectedChar(tn, cn, pl int) bool {
	m, n := 0, s.GetCharNo(cn)
	if len(s.charlist) == 0 || len(s.charlist[n].def) == 0 {
		return false
	}
	for s.charlist[n].def == "randomselect" || len(s.charlist[n].def) == 0 {
		m++
		if m > 100000 {
			return false
		}
		n = int(Rand(0, int32(len(s.charlist))-1))
		pl = int(Rand(1, MaxPalNo))
	}
	sys.loadMutex.Lock()
	s.selected[tn] = append(s.selected[tn], [...]int{n, pl})
	s.ocd[tn] = append(s.ocd[tn], *newOverrideCharData())
	sys.loadMutex.Unlock()
	return true
}

func (s *Select) ClearSelected() {
	sys.loadMutex.Lock()
	s.selected = [2][][2]int{}
	s.ocd = [3][]OverrideCharData{}
	sys.loadMutex.Unlock()
	s.selectedStageNo = -1
}

type LoaderState int32

const (
	LS_NotYet LoaderState = iota
	LS_Loading
	LS_Complete
	LS_Error
	LS_Cancel
)

type Loader struct {
	state    LoaderState
	loadExit chan LoaderState
	err      error
}

func newLoader() *Loader {
	return &Loader{state: LS_NotYet, loadExit: make(chan LoaderState, 1)}
}

func (l *Loader) loadPlayerChar(pn int) int {
	return l.loadCharacter(pn, false)
}

func (l *Loader) loadAttachedChar(pn int) int {
	return l.loadCharacter(pn, true)
}

func (l *Loader) loadCharacter(pn int, attached bool) int {
	if !attached && sys.roundsExisted[pn&1] > 0 {
		return 1
	}
	if attached && sys.round != 1 {
		return 1
	}

	sys.loadMutex.Lock()
	defer sys.loadMutex.Unlock()

	// Get number of selected characters in team
	memberNo := pn >> 1
	nsel := len(sys.sel.selected[pn&1])

	// Check if player number is acceptable for selected team mode
	if !attached {
		if sys.tmode[pn&1] == TM_Simul || sys.tmode[pn&1] == TM_Tag {
			if memberNo >= int(sys.numSimul[pn&1]) {
				sys.cgi[pn].states = nil
				sys.chars[pn] = nil
				return 1
			}
		} else if pn >= 2 {
			return 0
		}

		if sys.tmode[pn&1] == TM_Turns && nsel < int(sys.numTurns[pn&1]) {
			return 0
		}

		if sys.tmode[pn&1] == TM_Turns {
			memberNo = int(sys.wins[^pn&1])
		}

		if nsel <= memberNo {
			return 0
		}
	}

	idx := make([]int, nsel)
	for i := range idx {
		idx[i] = sys.sel.selected[pn&1][i][0]
	}

	// Prepare loading time clipboard message
	var tstr string
	tnow := time.Now()
	defer func() {
		sys.loadTime(tnow, tstr, false, true)
		// Mugen compatibility mode indicator
		if sys.cgi[pn].ikemenver[0] == 0 && sys.cgi[pn].ikemenver[1] == 0 {
			if sys.cgi[pn].mugenver[0] == 1 && sys.cgi[pn].mugenver[1] == 1 {
				sys.appendToConsole("Using Mugen 1.1 compatibility mode.")
			} else if sys.cgi[pn].mugenver[0] == 1 && sys.cgi[pn].mugenver[1] == 0 {
				sys.appendToConsole("Using Mugen 1.0 compatibility mode.")
			} else if sys.cgi[pn].mugenver[0] != 1 {
				sys.appendToConsole("Using WinMugen compatibility mode.")
			} else {
				sys.appendToConsole("Character with unknown engine version.")
			}
		}
	}()

	var cdef string
	var cdefOWnumber int

	if attached {
		atcpn := pn - MaxSimul*2
		cdef = sys.stageList[0].attachedchardef[atcpn]
	} else {
		if sys.tmode[pn&1] == TM_Turns {
			cdefOWnumber = memberNo*2 + pn&1
		} else {
			cdefOWnumber = pn
		}
		if sys.sel.cdefOverwrite[cdefOWnumber] != "" {
			cdef = sys.sel.cdefOverwrite[cdefOWnumber]
		} else {
			cdef = sys.sel.charlist[idx[memberNo]].def
		}
	}

	var p *Char
	sys.workingChar = p // This should help compiler and bytecode stay consistent

	// Reuse existing character or create a new one
	if len(sys.chars[pn]) > 0 && cdef == sys.cgi[pn].def {
		p = sys.chars[pn][0]
		if !attached {
			p.controller = pn
			if sys.aiLevel[pn] != 0 {
				p.controller ^= -1
			}
		}
		p.clearCachedData()
	} else {
		p = newChar(pn, 0)
		sys.cgi[pn].sff = nil
		sys.cgi[pn].palettedata = nil
		if len(sys.chars[pn]) > 0 {
			p.power = sys.chars[pn][0].power
			p.guardPoints = sys.chars[pn][0].guardPoints
			p.dizzyPoints = sys.chars[pn][0].dizzyPoints
		}
	}

	// Set new character parameters
	if attached {
		atcpn := pn - MaxSimul*2
		p.memberNo = -atcpn
		p.selectNo = -atcpn
		p.teamside = -1
		sys.aiLevel[pn] = float32(sys.cfg.Options.Difficulty)
	} else {
		p.memberNo = memberNo
		p.selectNo = sys.sel.selected[pn&1][memberNo][0]
		p.teamside = p.playerNo & 1
	}

	if !p.ocd().existed {
		p.initCnsVar()
		p.ocd().existed = true
	}

	sys.chars[pn] = make([]*Char, 1)
	sys.chars[pn][0] = p

	// Load new SFF if previous one was not cached
	if sys.cgi[pn].sff == nil {
		if l.err = p.load(cdef); l.err != nil {
			sys.chars[pn] = nil
			if attached {
				tstr = fmt.Sprintf("WARNING: Failed to load new attached char: %v", cdef)
			} else {
				tstr = fmt.Sprintf("WARNING: Failed to load new char: %v", cdef)
			}
			return -1
		}
		if sys.cgi[pn].states, l.err = newCompiler().Compile(p.playerNo, cdef, p.gi().constants); l.err != nil {
			sys.chars[pn] = nil
			if attached {
				tstr = fmt.Sprintf("WARNING: Failed to compile new attached char states: %v", cdef)
			} else {
				tstr = fmt.Sprintf("WARNING: Failed to compile new char states: %v", cdef)
			}
			return -1
		}
		if attached {
			tstr = fmt.Sprintf("New attached char loaded: %v", cdef)
		} else {
			tstr = fmt.Sprintf("New char loaded: %v", cdef)
		}
	} else {
		if attached {
			tstr = fmt.Sprintf("Cached attached char loaded: %v", cdef)
		} else {
			tstr = fmt.Sprintf("Cached char loaded: %v", cdef)
		}
	}

	if attached {
		sys.cgi[pn].palno = 1
	} else {
		// Get palette number from select screen choice
		sys.cgi[pn].palno = int32(sys.sel.selected[pn&1][memberNo][1])
		// Prepare lifebar portraits
		if pn < len(sys.lifebar.fa[sys.tmode[pn&1]]) && sys.tmode[pn&1] == TM_Turns && sys.round == 1 {
			fa := sys.lifebar.fa[sys.tmode[pn&1]][pn]
			fa.numko = 0
			fa.teammate_face = make([]*Sprite, nsel)
			fa.teammate_scale = make([]float32, nsel)
			sys.lifebar.nm[sys.tmode[pn&1]][pn].numko = 0
			for i, ci := range idx {
				fa.teammate_scale[i] = sys.sel.charlist[ci].portrait_scale
				fa.teammate_face[i] = sys.sel.charlist[ci].sff.GetSprite(int16(fa.teammate_face_spr[0]), int16(fa.teammate_face_spr[1]))
			}
		}
	}
	return 1
}

func (l *Loader) loadStage() bool {
	if sys.round == 1 {
		var tstr string
		tnow := time.Now()
		defer func() {
			if sys.stage != nil {
				sys.loadTime(tnow, tstr, false, true)
				// Mugen compatibility mode indicator
				if sys.stage.ikemenver[0] == 0 && sys.stage.ikemenver[1] == 0 {
					if sys.stage.mugenver[0] == 1 && sys.stage.mugenver[1] == 1 {
						sys.appendToConsole("Using Mugen 1.1 compatibility mode.")
					} else if sys.stage.mugenver[0] == 1 && sys.stage.mugenver[1] == 0 {
						sys.appendToConsole("Using Mugen 1.0 compatibility mode.")
					} else if sys.stage.mugenver[0] != 1 {
						sys.appendToConsole("Using WinMugen compatibility mode.")
					} else {
						sys.appendToConsole("Stage with unknown engine version.")
					}
				}
				// Warn when camera boundaries are smaller than player boundaries
				if int32(sys.stage.leftbound) > sys.stage.stageCamera.boundleft || int32(sys.stage.rightbound) < sys.stage.stageCamera.boundright {
					sys.appendToConsole("Warning: Stage player boundaries defined incorrectly")
				}
			}
		}()
		var def string
		if sys.sel.selectedStageNo == 0 {
			randomstageno := Rand(0, int32(len(sys.sel.stagelist))-1)
			def = sys.sel.stagelist[randomstageno].def
		} else {
			def = sys.sel.stagelist[sys.sel.selectedStageNo-1].def
		}
		if sys.sel.sdefOverwrite != "" {
			def = sys.sel.sdefOverwrite
		}
		if sys.stage != nil && sys.stage.def == def && sys.stage.mainstage && !sys.stage.reload {
			tstr = fmt.Sprintf("Cached stage loaded: %v", def)
			return true
		}
		sys.stageList = make(map[int32]*Stage)
		sys.stageLoop = false
		sys.stageList[0], l.err = loadStage(def, true)
		sys.stage = sys.stageList[0]
		tstr = fmt.Sprintf("New stage loaded: %v", def)
	}
	return l.err == nil
}

func (l *Loader) load() {
	defer func() {
		l.loadExit <- l.state
	}()

	charDone, stageDone := make([]bool, len(sys.chars)), false

	// Check if all chars are loaded
	allCharDone := func() bool {
		for _, b := range charDone {
			if !b {
				return false
			}
		}
		return true
	}

	for !stageDone || !allCharDone() {
		// Load stage
		if !stageDone && sys.sel.selectedStageNo >= 0 {
			if !l.loadStage() {
				l.state = LS_Error
				return
			}
			stageDone = true
		}
		// Load characters that aren't already loaded
		for i, b := range charDone {
			if !b {
				result := -1
				if i < len(sys.chars)-MaxAttachedChar ||
					len(sys.stageList[0].attachedchardef) <= i-MaxSimul*2 {
					result = l.loadCharacter(i, false)
				} else {
					result = l.loadCharacter(i, true)
				}
				if result > 0 {
					charDone[i] = true
				} else if result < 0 {
					l.state = LS_Error
					return
				}
			}
		}
		for i := 0; i < 2; i++ {
			if !charDone[i+2] && len(sys.sel.selected[i]) > 0 &&
				sys.tmode[i] != TM_Simul && sys.tmode[i] != TM_Tag {
				for j := i + 2; j < len(sys.chars); j += 2 {
					if !charDone[j] {
						sys.chars[j] = nil
						sys.cgi[j].states = nil
						sys.cgi[j].hitPauseToggleFlagCount = 0
						charDone[j] = true
					}
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
		if sys.gameEnd {
			l.state = LS_Cancel
		}
		if l.state == LS_Cancel {
			return
		}
	}

	// Flag loading state as complete
	l.state = LS_Complete
}

func (l *Loader) reset() {
	if l.state != LS_NotYet {
		l.state = LS_Cancel
		<-l.loadExit
		l.state = LS_NotYet
	}
	l.err = nil
	for i := range sys.cgi {
		if sys.roundsExisted[i&1] == 0 {
			sys.cgi[i].palno = -1
		}
	}
}

func (l *Loader) runTread() bool {
	if l.state != LS_NotYet {
		return false
	}
	l.state = LS_Loading
	go l.load()
	return true
}

type EnvShake struct {
	time  int32
	freq  float32
	ampl  float32
	phase float32
	mul   float32
	dir   float32 // rad, for ampl=-4:  0: down first, 90: left first, 180: up first, 270: right first
}

func (es *EnvShake) clear() {
	*es = EnvShake{
		freq:  float32(math.Pi / 3),
		ampl:  -4.0,
		phase: float32(math.NaN()),
		mul:   1.0,
		dir:   0.0}
}

func (es *EnvShake) setDefaultPhase() {
	if math.IsNaN(float64(es.phase)) {
		if es.freq >= math.Pi/2 {
			es.phase = math.Pi / 2
		} else {
			es.phase = 0
		}
	}
}

func (es *EnvShake) next() {
	if es.time > 0 {
		es.time--
		es.phase += es.freq
		if es.phase > math.Pi*2 {
			es.ampl *= es.mul
			es.phase -= math.Pi * 2
		}
	} else {
		es.ampl = 0
	}
}

func (es *EnvShake) getOffset() [2]float32 {
	if es.time > 0 {
		offset := -(es.ampl * float32(math.Sin(float64(es.phase))))
		return [2]float32{offset * float32(math.Sin(float64(-es.dir))),
			offset * float32(math.Cos(float64(-es.dir)))}
	}
	return [2]float32{0, 0}
}
