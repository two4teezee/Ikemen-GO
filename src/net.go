package main

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"log"
	"math"
	"os"
	"strings"
	"time"
	"unsafe"

	ggpo "github.com/assemblaj/ggpo"
)

type RollbackLogger struct {
	filename   string
	currentLog strings.Builder
	logs       [(MaxSaveStates + 3) * 3]string
}

func NewRollbackLogger(timestamp string) RollbackLogger {
	gl := RollbackLogger{filename: "save/logs/Rollback-State-" + timestamp + ".log"}
	return gl
}

func (g *RollbackLogger) logState(action string, stateID int, state string, checksum int) {
	fmt.Fprintf(&g.currentLog, "State Logger: \n")
	fmt.Fprintf(&g.currentLog, "%s state for stateID: %d\n", action, stateID)
	fmt.Fprintf(&g.currentLog, state+"\n")
	fmt.Fprintf(&g.currentLog, "Checksum: %d\n", checksum)
}

func (g *RollbackLogger) Write(p []byte) (n int, err error) {
	g.currentLog.WriteString(string(p))
	return len(p), nil
}

func (g *RollbackLogger) updateLogs() {
	tmp := g.logs
	for i := 0; i < len(g.logs)-1; i++ {
		g.logs[i] = tmp[i+1]
	}
	g.logs[len(g.logs)-1] = g.currentLog.String()
	g.currentLog.Reset()
}

func (g *RollbackLogger) saveLogs() {
	var fullLog string
	for i := 0; i < len(g.logs); i++ {
		fullLog += g.logs[i]
	}
	err := os.WriteFile(g.filename, []byte(fullLog), 0666)
	if err != nil {
		fmt.Println(err)
	}
}

type RollbackSession struct {
	backend             ggpo.Backend
	saveStates          map[int]*GameState
	now                 int64
	next                int64
	players             []ggpo.Player
	handles             []ggpo.PlayerHandle
	rep                 *os.File
	connected           bool
	host                string
	playerNo            int
	syncProgress        int
	synchronized        bool
	syncTest            bool
	run                 int
	remoteIp            string
	currentPlayer       int
	currentPlayerHandle ggpo.PlayerHandle
	remotePlayerHandle  ggpo.PlayerHandle
	loopTimer           LoopTimer
	inputs              [][MaxSimul*2 + MaxAttachedChar]InputBits
	config              RollbackConfig
	log                 RollbackLogger
	timestamp           string
	netTime             int32
}

func (rs *RollbackSession) SetInput(time int32, player int, input InputBits) {
	for len(rs.inputs) <= int(time) {
		rs.inputs = append(rs.inputs, [MaxSimul*2 + MaxAttachedChar]InputBits{})
	}
	rs.inputs[time][player] = input
}

func (rs *RollbackSession) SaveReplay() {
	if rs.rep != nil {
		size := len(rs.inputs) * (MaxSimul*2 + MaxAttachedChar) * 4
		buf := make([]byte, size)
		offset := 0
		for i := range rs.inputs {
			inputBuf := rs.inputToBytes(i)
			copy(buf[offset:offset+len(inputBuf)], inputBuf)
			offset += len(inputBuf)
		}
		rs.rep.Write(buf)
	}
}

func (rs *RollbackSession) inputToBytes(time int) []byte {
	buf := []byte{}
	for i := 0; i < MaxSimul*2+MaxAttachedChar; i++ {
		buf = append(buf, writeI32(int32(rs.inputs[time][i]))...)
	}
	return buf
}

type LoopTimer struct {
	lastAdvantage      float32
	usPergameLoop      time.Duration
	usExtraToWait      int
	framesToSpreadWait int
	waitCount          time.Duration
	timeWait           time.Duration
	waitTotal          time.Duration
}

func NewLoopTimer(fps uint32, framesToSpread uint32) LoopTimer {
	return LoopTimer{
		framesToSpreadWait: int(framesToSpread),
		usPergameLoop:      time.Second / time.Duration(fps),
	}
}

func (lt *LoopTimer) OnGGPOTimeSyncEvent(framesAhead float32) {
	lt.waitTotal = time.Duration(float32(time.Second/60) * framesAhead)
	lt.lastAdvantage = float32(time.Second/60) * framesAhead
	lt.lastAdvantage = lt.lastAdvantage / 4
	lt.timeWait = time.Duration(lt.lastAdvantage) / time.Duration(lt.framesToSpreadWait)
	lt.waitCount = time.Duration(lt.framesToSpreadWait)
}

func (lt *LoopTimer) usToWaitThisLoop() time.Duration {
	if lt.waitCount > 0 {
		lt.waitCount--
		return lt.usPergameLoop + lt.timeWait
	}
	return lt.usPergameLoop
}

func (r *RollbackSession) Close() {
	if r.backend != nil {
		r.backend.Close()
	}
}

func (r *RollbackSession) IsConnected() bool {
	return r.connected
}

func (r *RollbackSession) SaveGameState(stateID int) int {
	sys.savePool.curStateID = stateID
	sys.rollbackStateID = stateID
	oldest := stateID + 1%(MaxSaveStates+2)
	if _, ok := sys.arenaSaveMap[oldest]; ok {
		sys.arenaSaveMap[oldest].Free()
		sys.arenaSaveMap[oldest] = nil
		delete(sys.arenaSaveMap, oldest)
	}
	sys.savePool.Free(oldest)
	if _, ok := sys.arenaSaveMap[stateID]; ok {
		sys.arenaSaveMap[stateID].Free()
		sys.arenaSaveMap[stateID] = nil
		delete(sys.arenaSaveMap, stateID)
	}
	sys.savePool.Free(stateID)

	r.saveStates[stateID] = sys.statePool.gameStatePool.Get().(*GameState)
	r.saveStates[stateID].SaveState(stateID)

	if r.config.DesyncTest {
		checksum := r.saveStates[stateID].Checksum()
		if r.config.LogsEnabled {
			r.log.logState("Saving", stateID, r.saveStates[stateID].String(), checksum)
			r.log.updateLogs()
		}
		return checksum
	} else {
		if r.config.LogsEnabled {
			checksum := r.saveStates[stateID].Checksum()
			r.log.logState("Saving", stateID, r.saveStates[stateID].String(), checksum)
			r.log.updateLogs()
		}
		return ggpo.DefaultChecksum
	}
}

var lastLoadedFrame int = -1

func (r *RollbackSession) LoadGameState(stateID int) {
	sys.loadPool.curStateID = stateID
	if _, ok := sys.arenaLoadMap[stateID]; ok {
		sys.arenaLoadMap[stateID].Free()
		sys.arenaLoadMap[stateID] = nil
		delete(sys.arenaLoadMap, stateID)
	}
	for sid := range sys.arenaLoadMap {
		if sid != lastLoadedFrame {
			sys.arenaLoadMap[sid].Free()
			sys.arenaLoadMap[sid] = nil
			delete(sys.arenaLoadMap, sid)
		}
	}
	sys.loadPool.Free(stateID)

	r.saveStates[stateID].LoadState(stateID)

	if r.config.DesyncTest && r.config.LogsEnabled {
		checksum := r.saveStates[stateID].Checksum()
		r.log.logState("Loaded", stateID, r.saveStates[stateID].String(), checksum)
	}

	sys.statePool.gameStatePool.Put(r.saveStates[stateID])

	lastLoadedFrame = stateID
}

func (r *RollbackSession) AdvanceFrame(flags int) {
	var discconectFlags int
	//Lua code executed before drawing fade, clsns and debug
	for _, str := range sys.commonLua {
		if err := sys.luaLState.DoString(str); err != nil {
			sys.luaLState.RaiseError(err.Error())
		}
	}

	// Make sure we fetch the inputs from GGPO and use these to update
	// the game state instead of reading from the keyboard.
	inputs, result := r.backend.SyncInput(&discconectFlags)
	if result == nil {
		input := decodeInputs(inputs)
		if r.rep != nil {
			r.SetInput(sys.gameTime, 0, input[0])
			r.SetInput(sys.gameTime, 1, input[1])
		}

		sys.step = false
		//sys.rollback.runShortcutScripts(&sys)

		// If next round
		if !sys.rollback.runNextRound(&sys) {
			return
		}

		sys.bgPalFX.step()
		sys.stage.action()

		//sys.rollback.updateStage(&sys)

		// update lua
		for i := 0; i < len(inputs) && i < len(sys.commandLists); i++ {
			sys.commandLists[i].Buffer.InputBits(input[i], 1)
			sys.commandLists[i].Step(1, false, false, 0)
		}
		sys.rollback.action(&sys, input)

		// if sys.rollback.handleFlags(&sys) {
		// 	return
		// }

		if !sys.rollback.updateEvents(&sys) {
			return
		}

		if sys.rollback.currentFight.fin && (!sys.postMatchFlg || len(sys.commonLua) == 0) {
			return
		}

		if sys.endMatch {
			sys.esc = true
			return
		} else if sys.esc {
			sys.endMatch = sys.netInput != nil || len(sys.commonLua) == 0
			return
		}

		//sys.rollback.updateCamera(&sys)
		defer func() {
			if re := recover(); re != nil {
				if r.config.DesyncTest {
					r.log.updateLogs()
					r.log.saveLogs()
					panic("RaiseDesyncError")
				}
			}
		}()

		err := r.backend.AdvanceFrame(r.LiveChecksum(&sys))
		if err != nil {
			panic(err)
		}
	}
}

func (r *RollbackSession) OnEvent(info *ggpo.Event) {
	switch info.Code {
	case ggpo.EventCodeConnectedToPeer:
		r.connected = true
	case ggpo.EventCodeSynchronizingWithPeer:
		r.syncProgress = 100 * (info.Count / info.Total)
	case ggpo.EventCodeSynchronizedWithPeer:
		r.syncProgress = 100
		r.synchronized = true
	case ggpo.EventCodeRunning:
		fmt.Println("EventCodeRunning")
	case ggpo.EventCodeDisconnectedFromPeer:
		if sys.postMatchFlg {
			return
		}
		if r.config.LogsEnabled {
			r.log.saveLogs()
		}
		fmt.Println("EventCodeDisconnectedFromPeer")
		sys.rollback.currentFight.fin = true
		sys.endMatch = true
		disconnectMessage := fmt.Sprintf("Player %d disconnected.", info.Player)
		r.SaveReplay()
		ShowInfoDialog(disconnectMessage, "Disconnection")
	case ggpo.EventCodeTimeSync:
		fmt.Printf("EventCodeTimeSync: FramesAhead %f TimeSyncPeriodInFrames: %d\n", info.FramesAhead, info.TimeSyncPeriodInFrames)
		r.loopTimer.OnGGPOTimeSyncEvent(info.FramesAhead)
	case ggpo.EventCodeDesync:
		if r.config.LogsEnabled {
			r.log.saveLogs()
		}
		fmt.Println("EventCodeDesync")
		sys.rollback.currentFight.fin = true
		sys.endMatch = true
		r.SaveReplay()
		ShowInfoDialog("Desync Error. Please report your issue to the rollback alpha issues page at https://github.com/assemblaj/Ikemen-GO/issues. Thanks for your patience", "Desync Error")
	case ggpo.EventCodeConnectionInterrupted:
		fmt.Println("EventCodeconnectionInterrupted")
	case ggpo.EventCodeConnectionResumed:
		fmt.Println("EventCodeconnectionInterrupted")
	}
}

func NewRollbackSesesion(config RollbackConfig) RollbackSession {
	r := RollbackSession{}
	r.saveStates = make(map[int]*GameState)
	r.players = make([]ggpo.Player, 9)
	r.handles = make([]ggpo.PlayerHandle, 9)
	r.config = config
	r.loopTimer = NewLoopTimer(60, 100)
	r.timestamp = time.Now().Format("2006-01-02_03-04PM-05s")
	r.log = NewRollbackLogger(r.timestamp)
	return r

}
func encodeInputs(inputs InputBits) []byte {
	return writeI32(int32(inputs))
}

type CharChecksum struct {
	life    int32
	redLife int32
	juggle  int32
	animNo  int32
	pos     [3]float32
}

func (cc *CharChecksum) ToBytes() []byte {
	buf := make([]byte, 0, unsafe.Sizeof(*cc))
	buf = binary.BigEndian.AppendUint32(buf, uint32(cc.life))
	buf = binary.BigEndian.AppendUint32(buf, uint32(cc.redLife))
	buf = binary.BigEndian.AppendUint32(buf, uint32(cc.juggle))
	buf = binary.BigEndian.AppendUint32(buf, uint32(cc.animNo))
	buf = binary.BigEndian.AppendUint32(buf, math.Float32bits(cc.pos[0]))
	buf = binary.BigEndian.AppendUint32(buf, math.Float32bits(cc.pos[1]))
	buf = binary.BigEndian.AppendUint32(buf, math.Float32bits(cc.pos[2]))
	return buf
}

func (c *Char) LiveChecksum() []byte {
	cc := CharChecksum{
		life:    c.life,
		redLife: c.redLife,
		juggle:  c.juggle,
		animNo:  c.animNo,
		pos:     c.pos,
	}
	return cc.ToBytes()
}

func (rs *RollbackSession) LiveChecksum(s *System) uint32 {
	// This (the full checksum) is unstable in live gameplay, do not use. Looking for replacements.
	// if rs.config.LogsEnabled {
	// 	return uint32(rs.saveStates[sys.rollbackStateID].Checksum())
	// }
	buf := writeI32(s.randseed)
	buf = append(buf, writeI32(s.gameTime)...)
	for i := 0; i < len(s.chars); i++ {
		if len(s.chars[i]) > 0 {
			buf = append(buf, s.chars[i][0].LiveChecksum()...)
		}
	}
	return crc32.ChecksumIEEE(buf)
}
func (rs *RollbackSession) Input(time int32, player int) (input InputBits) {
	inputs := rs.inputs[time]
	input = inputs[player]
	return
}
func (rs *RollbackSession) AnyButton() bool {
	for i := 0; i < len(rs.inputs[len(rs.inputs)-1]); i++ {
		if rs.Input(sys.gameTime, i)&IB_anybutton != 0 {
			return true
		}
	}
	return false
}

func (rs *RollbackSession) AnyButtonIB(ib []InputBits) bool {
	for i := 0; i < len(ib); i++ {
		if ib[i]&IB_anybutton != 0 {
			return true
		}
	}
	return false
}

func (rs *RollbackSession) InitP1(numPlayers int, localPort int, remotePort int, remoteIp string) {
	if rs.config.LogsEnabled {
		logFileName := fmt.Sprintf("save/logs/Rollback-%s.log", rs.timestamp)
		f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			panic(err)
		}
		logger := log.New(f, "Logger:", log.Ldate|log.Ltime|log.Lshortfile)
		ggpo.EnableLogs()
		ggpo.SetLogger(logger)
	}

	var inputBits InputBits = 0
	var inputSize int = len(encodeInputs(inputBits))

	player := ggpo.NewLocalPlayer(20, 1)
	player2 := ggpo.NewRemotePlayer(20, 2, remoteIp, remotePort)
	rs.players = append(rs.players, player)
	rs.players = append(rs.players, player2)

	peer := ggpo.NewPeer(rs, localPort, numPlayers, inputSize)
	rs.backend = &peer

	peer.InitializeConnection()

	var handle ggpo.PlayerHandle
	result := peer.AddPlayer(&player, &handle)
	if result != nil {
		panic("panic")
	}
	rs.handles = append(rs.handles, handle)
	var handle2 ggpo.PlayerHandle
	result = peer.AddPlayer(&player2, &handle2)
	if result != nil {
		panic("panic")
	}
	rs.handles = append(rs.handles, handle2)
	rs.currentPlayer = int(handle)
	rs.currentPlayerHandle = handle
	rs.remotePlayerHandle = handle2

	peer.SetDisconnectTimeout(rs.config.DisconnectTimeout)
	peer.SetDisconnectNotifyStart(rs.config.DisconnectNotifyStart)
	peer.SetFrameDelay(handle, rs.config.FrameDelay)

	peer.Start()
}

func (rs *RollbackSession) InitP2(numPlayers int, localPort int, remotePort int, remoteIp string) {
	if rs.config.LogsEnabled {
		logFileName := fmt.Sprintf("save/logs/Rollback-%s.log", rs.timestamp)
		f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			panic(err)
		}
		logger := log.New(f, "Logger:", log.Ldate|log.Ltime|log.Lshortfile)
		ggpo.EnableLogs()
		ggpo.SetLogger(logger)
	}

	var inputBits InputBits = 0
	var inputSize int = len(encodeInputs(inputBits))

	player := ggpo.NewRemotePlayer(20, 1, remoteIp, remotePort)
	player2 := ggpo.NewLocalPlayer(20, 2)
	rs.players = append(rs.players, player)
	rs.players = append(rs.players, player2)

	peer := ggpo.NewPeer(rs, localPort, numPlayers, inputSize)
	rs.backend = &peer

	peer.InitializeConnection()

	var handle ggpo.PlayerHandle
	result := peer.AddPlayer(&player, &handle)
	if result != nil {
		panic("panic")
	}
	rs.handles = append(rs.handles, handle)
	var handle2 ggpo.PlayerHandle
	result = peer.AddPlayer(&player2, &handle2)
	if result != nil {
		panic("panic")
	}
	rs.handles = append(rs.handles, handle2)
	rs.currentPlayer = int(handle2)
	rs.currentPlayerHandle = handle2
	rs.remotePlayerHandle = handle

	peer.SetDisconnectTimeout(rs.config.DisconnectTimeout)
	peer.SetDisconnectNotifyStart(rs.config.DisconnectNotifyStart)
	peer.SetFrameDelay(handle2, rs.config.FrameDelay)

	peer.Start()
}

func (rs *RollbackSession) InitSyncTest(numPlayers int) {
	rs.syncTest = true
	if rs.config.LogsEnabled {
		logFileName := fmt.Sprintf("save/logs/Rollback-Desync-Test-%s.log", rs.timestamp)
		f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			panic(err)
		}
		logger := log.New(f, "Logger:", log.Ldate|log.Ltime|log.Lshortfile)
		ggpo.EnableLogs()
		ggpo.SetLogger(logger)
	}

	var inputBits InputBits = 0
	var inputSize int = len(encodeInputs(inputBits))

	player := ggpo.NewLocalPlayer(20, 1)
	player2 := ggpo.NewLocalPlayer(20, 2)
	rs.players = append(rs.players, player)
	rs.players = append(rs.players, player2)

	//peer := ggpo.NewPeer(sys.rollbackNetwork, localPort, numPlayers, inputSize)
	peer := ggpo.NewSyncTest(rs, numPlayers, rs.config.DesyncTestFrames, inputSize, true)
	rs.backend = &peer

	//
	peer.InitializeConnection()
	peer.Start()

	var handle ggpo.PlayerHandle
	result := peer.AddPlayer(&player, &handle)
	if result != nil {
		panic("panic")
	}
	rs.handles = append(rs.handles, handle)
	var handle2 ggpo.PlayerHandle
	result = peer.AddPlayer(&player2, &handle2)
	if result != nil {
		panic("panic")
	}
	rs.handles = append(rs.handles, handle2)

	peer.SetDisconnectTimeout(3000)
	peer.SetDisconnectNotifyStart(1000)
}
