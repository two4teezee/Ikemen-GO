package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

type DebugCommand struct {
	Type       string   `json:"type"`
	TargetID   int32    `json:"target_id"`
	VarType    string   `json:"var_type"`
	Index      int32    `json:"index"`
	Key        string   `json:"key"`
	Value      float32  `json:"value"`
	ValueStr   string   `json:"value_str"`
	Action     string   `json:"action"`
	Code       string   `json:"code"`
	Enable     bool     `json:"enable"`
	TargetType string   `json:"target_type"`
	PropName   string   `json:"prop_name"`
	PlayerIdx  int      `json:"player_idx"`
	ItemIndex  int      `json:"item_index"`
	Categories []string `json:"categories"`
}

type Breakpoint struct {
	Code    string
	Enabled bool
}

type DebugApiServer struct {
	ln       net.Listener
	conn     net.Conn
	connMut  sync.Mutex
	cmdQueue []DebugCommand
	cmdMut   sync.Mutex

	reqExplods     bool
	reqProjs       bool
	reqCategories  []string
	propTargetType string
	propTargetID   int32
	propPlayerIdx  int
	propItemIndex  int

	evalRequests []string
	breakpoints  map[int]*Breakpoint
}

func NewDebugApiServer(port string) *DebugApiServer {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Sprintf("Failed to start Debug API server on port %s: %v\n", port, err)
		return nil
	}
	server := &DebugApiServer{
		ln:          ln,
		breakpoints: make(map[int]*Breakpoint),
	}
	go server.acceptLoop()
	return server
}

func (s *DebugApiServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			break
		}
		s.connMut.Lock()
		if s.conn != nil {
			s.conn.Close()
		}
		s.conn = conn
		s.connMut.Unlock()
		go s.handleConnection(conn)
	}
}

func (s *DebugApiServer) handleConnection(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Bytes()
		var cmd DebugCommand
		if err := json.Unmarshal(line, &cmd); err == nil {
			s.cmdMut.Lock()
			s.cmdQueue = append(s.cmdQueue, cmd)
			s.cmdMut.Unlock()
		}
	}
	s.connMut.Lock()
	if s.conn == conn {
		s.conn = nil
	}
	s.connMut.Unlock()
	conn.Close()
}

func (s *DebugApiServer) UpdateAndDump(sys *System) {
	s.connMut.Lock()
	conn := s.conn
	s.connMut.Unlock()

	if conn == nil {
		s.cmdMut.Lock()
		s.cmdQueue = s.cmdQueue[:0]
		s.cmdMut.Unlock()
		return
	}

	s.cmdMut.Lock()
	cmds := s.cmdQueue
	s.cmdQueue = nil
	s.cmdMut.Unlock()

	for _, cmd := range cmds {
		s.applyCommand(sys, cmd)
	}

	s.checkBreakpoints(sys, conn)

	dump := s.createDump(sys)

	if len(s.evalRequests) > 0 {
		evalResults := make(map[string]interface{})
		for _, code := range s.evalRequests {
			evalResults[code] = s.evaluateLua(sys.luaLState, code)
		}
		dump["lua_results"] = evalResults
		s.evalRequests = nil
	}

	data, err := json.Marshal(dump)
	if err == nil {
		data = append(data, '\n')
		if _, err = conn.Write(data); err != nil {
			conn.Close()
		}
	}
}

func applyPalFXProp(pfx *PalFX, name string, val float32) {
	if pfx == nil {
		return
	}
	switch name {
	case "palfx_time":
		pfx.time = int32(val)
	case "palfx_color":
		pfx.color = val / 256
	case "palfx_hue":
		pfx.hue = val / 256
	case "palfx_add_r":
		pfx.add[0] = int32(val)
	case "palfx_add_g":
		pfx.add[1] = int32(val)
	case "palfx_add_b":
		pfx.add[2] = int32(val)
	case "palfx_mul_r":
		pfx.mul[0] = int32(val)
	case "palfx_mul_g":
		pfx.mul[1] = int32(val)
	case "palfx_mul_b":
		pfx.mul[2] = int32(val)
	case "palfx_sinadd_r":
		pfx.sinadd[0] = int32(val)
	case "palfx_sinadd_g":
		pfx.sinadd[1] = int32(val)
	case "palfx_sinadd_b":
		pfx.sinadd[2] = int32(val)
	case "palfx_sinadd_cycletime":
		pfx.cycletime[0] = int32(val)
	case "palfx_sinmul_r":
		pfx.sinmul[0] = int32(val)
	case "palfx_sinmul_g":
		pfx.sinmul[1] = int32(val)
	case "palfx_sinmul_b":
		pfx.sinmul[2] = int32(val)
	case "palfx_sinmul_cycletime":
		pfx.cycletime[1] = int32(val)
	case "palfx_sincolor":
		pfx.sincolor = int32(val)
	case "palfx_sincolor_cycletime":
		pfx.cycletime[2] = int32(val)
	case "palfx_sinhue":
		pfx.sinhue = int32(val)
	case "palfx_sinhue_cycletime":
		pfx.cycletime[3] = int32(val)
	case "palfx_invertall":
		pfx.invertall = val > 0
	case "palfx_invertblend":
		pfx.invertblend = int32(val)
	}
}

func applyAfterImageProp(aimg *AfterImage, name string, val float32) {
	if aimg == nil {
		return
	}
	switch name {
	case "aimg_time":
		aimg.time = int32(val)
	case "aimg_length":
		aimg.length = int32(val)
	case "aimg_timegap":
		aimg.timegap = int32(val)
	case "aimg_framegap":
		aimg.framegap = int32(val)
	case "aimg_trans":
		aimg.trans = TransType(int32(val))
	case "aimg_alpha_src":
		aimg.alpha[0] = int32(val)
	case "aimg_alpha_dst":
		aimg.alpha[1] = int32(val)
	case "aimg_palcolor":
		aimg.setPalColor(int32(val))
	case "aimg_palhue":
		aimg.setPalHueShift(int32(val))
	case "aimg_palinvertall":
		aimg.setPalInvertall(val > 0)
	case "aimg_palinvertblend":
		aimg.setPalInvertblend(int32(val))
	case "aimg_palbright_r":
		aimg.setPalBrightR(int32(val))
	case "aimg_palbright_g":
		aimg.setPalBrightG(int32(val))
	case "aimg_palbright_b":
		aimg.setPalBrightB(int32(val))
	case "aimg_palcontrast_r":
		aimg.setPalContrastR(int32(val))
	case "aimg_palcontrast_g":
		aimg.setPalContrastG(int32(val))
	case "aimg_palcontrast_b":
		aimg.setPalContrastB(int32(val))
	case "aimg_palpostbright_r":
		aimg.postbright[0] = int32(val)
	case "aimg_palpostbright_g":
		aimg.postbright[1] = int32(val)
	case "aimg_palpostbright_b":
		aimg.postbright[2] = int32(val)
	case "aimg_paladd_r":
		aimg.add[0] = int32(val)
	case "aimg_paladd_g":
		aimg.add[1] = int32(val)
	case "aimg_paladd_b":
		aimg.add[2] = int32(val)
	case "aimg_palmul_r":
		aimg.mul[0] = val
	case "aimg_palmul_g":
		aimg.mul[1] = val
	case "aimg_palmul_b":
		aimg.mul[2] = val
	case "aimg_ignorehitpause":
		aimg.ignorehitpause = val > 0
	}
}

func (s *DebugApiServer) applyCommand(sys *System, cmd DebugCommand) {
	switch cmd.Type {
	case "action":
		if cmd.Action == "pause" {
			sys.paused = !sys.paused
		} else if cmd.Action == "step" {
			sys.frameStepFlag = true
		}
	case "set_target":
		if targetChar := sys.playerID(cmd.TargetID); targetChar != nil {
			sys.debugWC = targetChar
			sys.debugRef[0] = targetChar.playerNo
			sys.debugRef[1] = targetChar.helperIndex
			sys.debugLastID = targetChar.id
			sys.debugDisplay = true
		}
	case "set_var":
		if sys.debugWC != nil {
			switch cmd.VarType {
			case "cnsvar":
				sys.debugWC.varSet(cmd.Index, int32(cmd.Value))
			case "cnsfvar":
				sys.debugWC.fvarSet(cmd.Index, cmd.Value)
			case "sysvar":
				sys.debugWC.sysVarSet(cmd.Index, int32(cmd.Value))
			case "sysfvar":
				sys.debugWC.sysFvarSet(cmd.Index, cmd.Value)
			case "map":
				sys.debugWC.mapSet(cmd.Key, cmd.Value, 0)
			}
		}
	case "set_prop":
		switch cmd.TargetType {
		case "player":
			if c := sys.playerID(cmd.TargetID); c != nil {
				switch cmd.PropName {
				case "ctrl":
					if cmd.Value > 0 {
						c.setSCF(SCF_ctrl)
					} else {
						c.unsetSCF(SCF_ctrl)
					}
				case "stateno":
					c.changeState(int32(cmd.Value), -1, -1, "")
				case "life":
					c.lifeSet(int32(cmd.Value))
				case "lifeMax":
					c.lifeMax = int32(cmd.Value)
				case "power":
					c.powerOwner().setPower(int32(cmd.Value))
				case "powerMax":
					c.powerMax = int32(cmd.Value)
				case "dizzyPoints":
					c.dizzyPointsSet(int32(cmd.Value))
				case "dizzyPointsMax":
					c.dizzyPointsMax = int32(cmd.Value)
				case "guardPoints":
					c.guardPointsSet(int32(cmd.Value))
				case "guardPointsMax":
					c.guardPointsMax = int32(cmd.Value)
				case "redLife":
					c.redLifeSet(int32(cmd.Value))
				case "standby":
					if cmd.Value > 0 {
						c.setSCF(SCF_standby)
					} else {
						c.unsetSCF(SCF_standby)
					}
				case "pos_x":
					c.pos[0] = cmd.Value
				case "pos_y":
					c.pos[1] = cmd.Value
				case "pos_z":
					c.pos[2] = cmd.Value
				case "vel_x":
					c.vel[0] = cmd.Value
				case "vel_y":
					c.vel[1] = cmd.Value
				case "vel_z":
					c.vel[2] = cmd.Value
				case "facing":
					c.facing = cmd.Value
				case "offset_x":
					c.offset[0] = cmd.Value
				case "offset_y":
					c.offset[1] = cmd.Value
				case "groundLevel":
					c.groundLevel = cmd.Value
				case "posFreeze":
					if cmd.Value > 0 {
						c.setCSF(CSF_posfreeze)
					} else {
						c.unsetCSF(CSF_posfreeze)
					}
				case "bindTime":
					c.bindTime = int32(cmd.Value)
				case "bindToId":
					c.bindToId = int32(cmd.Value)
				case "bindPos_x":
					c.bindPos[0] = cmd.Value
				case "bindPos_y":
					c.bindPos[1] = cmd.Value
				case "bindPos_z":
					c.bindPos[2] = cmd.Value
				case "bindFacing":
					c.bindFacing = cmd.Value
				case "anglerot_x":
					c.anglerot[0] = cmd.Value
				case "anglerot_y":
					c.anglerot[1] = cmd.Value
				case "anglerot_z":
					c.anglerot[2] = cmd.Value
				case "xshear":
					c.xshear = cmd.Value
				case "projection":
					c.projection = Projection(int32(cmd.Value))
				case "fLength":
					c.fLength = cmd.Value
				case "angleDrawScale_x":
					c.angleDrawScale[0] = cmd.Value
				case "angleDrawScale_y":
					c.angleDrawScale[1] = cmd.Value
				case "zScale":
					c.zScale = cmd.Value
				case "alpha_src":
					c.alpha[0] = int32(cmd.Value)
				case "alpha_dst":
					c.alpha[1] = int32(cmd.Value)
				case "sprPriority":
					c.sprPriority = int32(cmd.Value)
				case "layerNo":
					c.layerNo = int32(cmd.Value)
				case "trans":
					c.trans = TransType(int32(cmd.Value))
				case "window_x":
					c.window[0] = cmd.Value
				case "window_y":
					c.window[1] = cmd.Value
				case "window_width":
					c.window[2] = cmd.Value
				case "window_height":
					c.window[3] = cmd.Value
				case "supertime":
					sys.supertime = int32(cmd.Value)
				case "pausetime":
					sys.pausetime = int32(cmd.Value)
				case "pauseMovetime":
					c.pauseMovetime = int32(cmd.Value)
				case "superMovetime":
					c.superMovetime = int32(cmd.Value)
				case "hitPauseTime":
					c.hitPauseTime = int32(cmd.Value)
				case "teamside":
					c.teamside = int(cmd.Value - 1)
				case "airJumpCount":
					c.airJumpCount = int32(cmd.Value)
				case "hitCount":
					c.hitCount = int32(cmd.Value)
				case "guardCount":
					c.guardCount = int32(cmd.Value)
				case "uniqHitCount":
					c.uniqHitCount = int32(cmd.Value)
				case "sizeWidth_f":
					c.sizeWidth[0] = cmd.Value
				case "sizeWidth_b":
					c.sizeWidth[1] = cmd.Value
				case "edgeWidth_f":
					c.edgeWidth[0] = cmd.Value
				case "edgeWidth_b":
					c.edgeWidth[1] = cmd.Value
				case "sizeHeight_t":
					c.sizeHeight[0] = cmd.Value
				case "sizeHeight_b":
					c.sizeHeight[1] = cmd.Value
				case "depth":
					if cmd.Value > 0 {
						c.setCSF(CSF_depth)
					} else {
						c.unsetCSF(CSF_depth)
					}
				case "depthEdge":
					if cmd.Value > 0 {
						c.setCSF(CSF_depthedge)
					} else {
						c.unsetCSF(CSF_depthedge)
					}
				case "sizeDepth_t":
					c.sizeDepth[0] = cmd.Value
				case "sizeDepth_b":
					c.sizeDepth[1] = cmd.Value
				case "edgeDepth_t":
					c.edgeDepth[0] = cmd.Value
				case "edgeDepth_b":
					c.edgeDepth[1] = cmd.Value
				case "pushPriority":
					c.pushPriority = int32(cmd.Value)
				case "pushAffectTeam":
					c.pushAffectTeam = int32(cmd.Value)
				case "statetype":
					switch cmd.ValueStr {
					case "S":
						c.ss.changeStateType(ST_S)
					case "C":
						c.ss.changeStateType(ST_C)
					case "A":
						c.ss.changeStateType(ST_A)
					case "L":
						c.ss.changeStateType(ST_L)
					}
				case "movetype":
					switch cmd.ValueStr {
					case "I":
						c.ss.changeMoveType(MT_I)
					case "A":
						c.ss.changeMoveType(MT_A)
					case "H":
						c.ss.changeMoveType(MT_H)
					}
				case "physics":
					switch cmd.ValueStr {
					case "S":
						c.ss.physics = ST_S
					case "C":
						c.ss.physics = ST_C
					case "A":
						c.ss.physics = ST_A
					case "N":
						c.ss.physics = ST_N
					}
				default:
					applyPalFXProp(c.getPalfx(), cmd.PropName, cmd.Value)
					if c.aimg != nil {
						applyAfterImageProp(c.aimg, cmd.PropName, cmd.Value)
					}
				}
			}
		case "explod":
			if cmd.PlayerIdx >= 0 && cmd.PlayerIdx < len(sys.explods) {
				if cmd.ItemIndex >= 0 && cmd.ItemIndex < len(sys.explods[cmd.PlayerIdx]) {
					e := sys.explods[cmd.PlayerIdx][cmd.ItemIndex]
					if e.id == cmd.TargetID {
						switch cmd.PropName {
						case "time":
							e.time = int32(cmd.Value)
						case "removetime":
							e.removetime = int32(cmd.Value)
						case "supermovetime":
							e.supermovetime = int32(cmd.Value)
						case "pausemovetime":
							e.pausemovetime = int32(cmd.Value)
						case "postype":
							e.postype = PosType(cmd.Value)
						case "space":
							e.space = Space(cmd.Value)
						case "bindId":
							e.bindId = int32(cmd.Value)
						case "bindtime":
							e.bindtime = int32(cmd.Value)
						case "pos_x":
							e.pos[0] = cmd.Value
						case "pos_y":
							e.pos[1] = cmd.Value
						case "pos_z":
							e.pos[2] = cmd.Value
						case "relativePos_x":
							e.relativePos[0] = cmd.Value
						case "relativePos_y":
							e.relativePos[1] = cmd.Value
						case "relativePos_z":
							e.relativePos[2] = cmd.Value
						case "facing":
							e.facing = cmd.Value
						case "vfacing":
							e.vfacing = cmd.Value
						case "velocity_x":
							e.velocity[0] = cmd.Value
						case "velocity_y":
							e.velocity[1] = cmd.Value
						case "velocity_z":
							e.velocity[2] = cmd.Value
						case "friction_x":
							e.friction[0] = cmd.Value
						case "friction_y":
							e.friction[1] = cmd.Value
						case "friction_z":
							e.friction[2] = cmd.Value
						case "accel_x":
							e.accel[0] = cmd.Value
						case "accel_y":
							e.accel[1] = cmd.Value
						case "accel_z":
							e.accel[2] = cmd.Value
						case "scale_x":
							e.scale[0] = cmd.Value
						case "scale_y":
							e.scale[1] = cmd.Value
						case "sprpriority":
							e.sprpriority = int32(cmd.Value)
						case "layerno":
							e.layerno = int32(cmd.Value)
						case "animfreeze":
							e.animfreeze = cmd.Value > 0
						case "trans":
							e.trans = TransType(cmd.Value)
						case "alpha_src":
							e.alpha[0] = int32(cmd.Value)
						case "alpha_dst":
							e.alpha[1] = int32(cmd.Value)
						case "anglerot_x":
							e.anglerot[0] = cmd.Value
						case "anglerot_y":
							e.anglerot[1] = cmd.Value
						case "anglerot_z":
							e.anglerot[2] = cmd.Value
						case "xshear":
							e.xshear = cmd.Value
						case "projection":
							e.projection = Projection(cmd.Value)
						case "fLength":
							e.fLength = cmd.Value
						case "window_x":
							e.window[0] = cmd.Value
						case "window_y":
							e.window[1] = cmd.Value
						case "window_width":
							e.window[2] = cmd.Value
						case "window_height":
							e.window[3] = cmd.Value
						case "ignorehitpause":
							e.ignorehitpause = cmd.Value > 0
						default:
							applyPalFXProp(e.palfx, cmd.PropName, cmd.Value)
							if e.aimg != nil {
								applyAfterImageProp(e.aimg, cmd.PropName, cmd.Value)
							}
						}
					}
				}
			}
		case "proj":
			if cmd.PlayerIdx >= 0 && cmd.PlayerIdx < len(sys.projs) {
				if cmd.ItemIndex >= 0 && cmd.ItemIndex < len(sys.projs[cmd.PlayerIdx]) {
					p := sys.projs[cmd.PlayerIdx][cmd.ItemIndex]
					if p.id == cmd.TargetID {
						switch cmd.PropName {
						case "totalhits":
							p.totalhits = int32(cmd.Value)
						case "priorityPoints":
							p.priorityPoints = int32(cmd.Value)
						case "scale_x":
							p.scale[0] = cmd.Value
						case "scale_y":
							p.scale[1] = cmd.Value
						case "anglerot_x":
							p.anglerot[0] = cmd.Value
						case "anglerot_y":
							p.anglerot[1] = cmd.Value
						case "anglerot_z":
							p.anglerot[2] = cmd.Value
						case "projection":
							p.projection = Projection(cmd.Value)
						case "fLength":
							p.fLength = cmd.Value
						case "clsnScale_x":
							p.clsnScale[0] = cmd.Value
						case "clsnScale_y":
							p.clsnScale[1] = cmd.Value
						case "clsnAngle":
							p.clsnAngle = cmd.Value
						case "zScale":
							p.zScale = cmd.Value
						case "window_x":
							p.window[0] = cmd.Value
						case "window_y":
							p.window[1] = cmd.Value
						case "window_width":
							p.window[2] = cmd.Value
						case "window_height":
							p.window[3] = cmd.Value
						case "xshear":
							p.xshear = cmd.Value
						case "sprpriority":
							p.sprpriority = int32(cmd.Value)
						case "layerno":
							p.layerno = int32(cmd.Value)
						case "pos_x":
							p.pos[0] = cmd.Value
						case "pos_y":
							p.pos[1] = cmd.Value
						case "pos_z":
							p.pos[2] = cmd.Value
						case "facing":
							p.facing = cmd.Value
						case "velocity_x":
							p.velocity[0] = cmd.Value
						case "velocity_y":
							p.velocity[1] = cmd.Value
						case "velocity_z":
							p.velocity[2] = cmd.Value
						case "remvelocity_x":
							p.remvelocity[0] = cmd.Value
						case "remvelocity_y":
							p.remvelocity[1] = cmd.Value
						case "remvelocity_z":
							p.remvelocity[2] = cmd.Value
						case "accel_x":
							p.accel[0] = cmd.Value
						case "accel_y":
							p.accel[1] = cmd.Value
						case "accel_z":
							p.accel[2] = cmd.Value
						case "velmul_x":
							p.velmul[0] = cmd.Value
						case "velmul_y":
							p.velmul[1] = cmd.Value
						case "velmul_z":
							p.velmul[2] = cmd.Value
						case "removetime":
							p.removetime = int32(cmd.Value)
						case "supermovetime":
							p.supermovetime = int32(cmd.Value)
						case "pausemovetime":
							p.pausemovetime = int32(cmd.Value)
						case "curmisstime":
							p.curmisstime = int32(cmd.Value)
						case "hitpause":
							p.hitpause = int32(cmd.Value)
						case "time":
							p.time = int32(cmd.Value)
						default:
							applyPalFXProp(p.palfx, cmd.PropName, cmd.Value)
							if p.aimg != nil {
								applyAfterImageProp(p.aimg, cmd.PropName, cmd.Value)
							}
						}
					}
				}
			}
		}
	case "eval_lua":
		s.evalRequests = append(s.evalRequests, cmd.Code)
	case "add_bp":
		s.breakpoints[int(cmd.Index)] = &Breakpoint{Code: cmd.Code, Enabled: true}
	case "toggle_bp":
		if bp, ok := s.breakpoints[int(cmd.Index)]; ok {
			bp.Enabled = cmd.Enable
		}
	case "clear_bp":
		s.breakpoints = make(map[int]*Breakpoint)
	case "req_details":
		if cmd.Action == "explods" {
			s.reqExplods = cmd.Enable
		}
		if cmd.Action == "projs" {
			s.reqProjs = cmd.Enable
		}
	case "req_props":
		s.propTargetType = cmd.TargetType
		s.propTargetID = cmd.TargetID
		s.propPlayerIdx = cmd.PlayerIdx
		s.propItemIndex = cmd.ItemIndex
		s.reqCategories = cmd.Categories
	}
}

func (s *DebugApiServer) evaluateLua(l *lua.LState, code string) interface{} {
	top := l.GetTop()
	defer l.SetTop(top)
	if err := l.DoString(code); err != nil {
		return fmt.Sprintf("Error: %s", err.Error())
	}
	if l.GetTop() > top {
		val := l.Get(-1)
		switch val.Type() {
		case lua.LTNumber:
			return float64(val.(lua.LNumber))
		case lua.LTString:
			return string(val.(lua.LString))
		case lua.LTBool:
			return bool(val.(lua.LBool))
		default:
			return val.String()
		}
	}
	return nil
}

func (s *DebugApiServer) checkBreakpoints(sys *System, conn net.Conn) {
	for id, bp := range s.breakpoints {
		if !bp.Enabled {
			continue
		}
		res := s.evaluateLua(sys.luaLState, bp.Code)
		if b, ok := res.(bool); ok && b {
			sys.paused = true
			event := map[string]interface{}{
				"event": "breakpoint_hit",
				"id":    id,
				"code":  bp.Code,
			}
			data, _ := json.Marshal(event)
			data = append(data, '\n')
			conn.Write(data)
		}
	}
}

func attrToStr(attr int32) string {
	if attr == 0 {
		return "None"
	}
	str := ""
	st := attr & int32(ST_MASK)
	at := attr & ^int32(ST_MASK)

	if st&int32(ST_S) != 0 {
		str += "S"
	}
	if st&int32(ST_C) != 0 {
		str += "C"
	}
	if st&int32(ST_A) != 0 {
		str += "A"
	}

	if at != 0 {
		str += ", "
		if at&int32(AT_AN) != 0 {
			str += "N"
		}
		if at&int32(AT_AS) != 0 {
			str += "S"
		}
		if at&int32(AT_AH) != 0 {
			str += "H"
		}
		if at&int32(AT_AA) != 0 {
			str += "A"
		}
		if at&int32(AT_AT) != 0 {
			str += "T"
		}
		if at&int32(AT_AP) != 0 {
			str += "P"
		}
	}
	return str
}

func dumpAfterImage(aimg *AfterImage) map[string]interface{} {
	var pcolor, phue float32
	var pial bool
	var pibl, pbr, pbg, pbb, pcr, pcg, pcb int32
	if len(aimg.palfx) > 0 && aimg.palfx[0] != nil {
		pfx := aimg.palfx[0]
		pcolor = pfx.eColor * 256
		phue = pfx.eHue * 256
		pial = pfx.eInvertall
		pibl = pfx.invertblend
		pbr, pbg, pbb = pfx.eAdd[0], pfx.eAdd[1], pfx.eAdd[2]
		pcr, pcg, pcb = pfx.eMul[0], pfx.eMul[1], pfx.eMul[2]
	}
	return map[string]interface{}{
		"aimg_time":            aimg.time,
		"aimg_length":          aimg.length,
		"aimg_timegap":         aimg.timegap,
		"aimg_framegap":        aimg.framegap,
		"aimg_trans":           aimg.trans,
		"aimg_alpha_src":       aimg.alpha[0],
		"aimg_alpha_dst":       aimg.alpha[1],
		"aimg_palcolor":        pcolor,
		"aimg_palhue":          phue,
		"aimg_palinvertall":    pial,
		"aimg_palinvertblend":  pibl,
		"aimg_palbright_r":     pbr,
		"aimg_palbright_g":     pbg,
		"aimg_palbright_b":     pbb,
		"aimg_palcontrast_r":   pcr,
		"aimg_palcontrast_g":   pcg,
		"aimg_palcontrast_b":   pcb,
		"aimg_palpostbright_r": aimg.postbright[0],
		"aimg_palpostbright_g": aimg.postbright[1],
		"aimg_palpostbright_b": aimg.postbright[2],
		"aimg_paladd_r":        aimg.add[0],
		"aimg_paladd_g":        aimg.add[1],
		"aimg_paladd_b":        aimg.add[2],
		"aimg_palmul_r":        aimg.mul[0],
		"aimg_palmul_g":        aimg.mul[1],
		"aimg_palmul_b":        aimg.mul[2],
		"aimg_ignorehitpause":  aimg.ignorehitpause,
	}
}

func dumpPalFX(pfx *PalFX) map[string]interface{} {
	return map[string]interface{}{
		"palfx_time":               pfx.time,
		"palfx_color":              pfx.color * 256,
		"palfx_hue":                pfx.hue * 256,
		"palfx_add_r":              pfx.add[0],
		"palfx_add_g":              pfx.add[1],
		"palfx_add_b":              pfx.add[2],
		"palfx_mul_r":              pfx.mul[0],
		"palfx_mul_g":              pfx.mul[1],
		"palfx_mul_b":              pfx.mul[2],
		"palfx_sinadd_r":           pfx.sinadd[0],
		"palfx_sinadd_g":           pfx.sinadd[1],
		"palfx_sinadd_b":           pfx.sinadd[2],
		"palfx_sinadd_cycletime":   pfx.cycletime[0],
		"palfx_sinmul_r":           pfx.sinmul[0],
		"palfx_sinmul_g":           pfx.sinmul[1],
		"palfx_sinmul_b":           pfx.sinmul[2],
		"palfx_sinmul_cycletime":   pfx.cycletime[1],
		"palfx_sincolor":           pfx.sincolor,
		"palfx_sincolor_cycletime": pfx.cycletime[2],
		"palfx_sinhue":             pfx.sinhue,
		"palfx_sinhue_cycletime":   pfx.cycletime[3],
		"palfx_invertall":          pfx.invertall,
		"palfx_invertblend":        pfx.invertblend,
	}
}

func (s *DebugApiServer) createDump(sys *System) map[string]interface{} {
	dump := map[string]interface{}{
		"tick":   sys.tickCount,
		"paused": sys.paused,
	}

	entities := make([]map[string]interface{}, 0)
	for i := range sys.chars {
		for _, c := range sys.chars[i] {
			if c != nil && !c.csf(CSF_destroy) {
				ent := map[string]interface{}{
					"id":           c.id,
					"player":       c.playerNo + 1,
					"helperindex":  c.helperIndex,
					"name":         c.name,
					"stateno":      c.ss.no,
					"player_index": c.indexTrigger(),
				}
				if c.helperIndex > 0 {
					ent["helperid"] = c.helperId
					ent["parentid"] = c.parentId
				}
				entities = append(entities, ent)
			}
		}
	}
	dump["entities"] = entities

	if sys.debugWC != nil {
		c := sys.debugWC

		st, mt, ph, pst, pmt := "S", "I", "N", "S", "I"
		if c.ss.stateType == ST_C {
			st = "C"
		} else if c.ss.stateType == ST_A {
			st = "A"
		} else if c.ss.stateType == ST_L {
			st = "L"
		}
		if c.ss.moveType == MT_A {
			mt = "A"
		} else if c.ss.moveType == MT_H {
			mt = "H"
		}
		if c.ss.physics == ST_S {
			ph = "S"
		} else if c.ss.physics == ST_C {
			ph = "C"
		} else if c.ss.physics == ST_A {
			ph = "A"
		}
		if c.ss.prevStateType == ST_C {
			pst = "C"
		} else if c.ss.prevStateType == ST_A {
			pst = "A"
		} else if c.ss.prevStateType == ST_L {
			pst = "L"
		}
		if c.ss.prevMoveType == MT_A {
			pmt = "A"
		} else if c.ss.prevMoveType == MT_H {
			pmt = "H"
		}

		props := make(map[string]map[string]interface{})

		// 指定されたターゲットに一致する場合のみプロパティを含める
		if s.propTargetType == "player" && s.propTargetID == c.id {
			for _, cat := range s.reqCategories {
				switch cat {
				case "Core":
					props["Core"] = map[string]interface{}{
						"stateno": c.ss.no, "life": c.life, "lifeMax": c.lifeMax, "power": c.getPower(), "powerMax": c.powerMax,
						"statetype": st, "movetype": mt, "physics": ph, "dizzyPoints": c.dizzyPoints, "dizzyPointsMax": c.dizzyPointsMax,
						"guardPoints": c.guardPoints, "guardPointsMax": c.guardPointsMax, "redLife": c.redLife, "teamside": c.teamside + 1,
						"ctrl": c.scf(SCF_ctrl), "standby": c.scf(SCF_standby),
					}
				case "Position":
					props["Position"] = map[string]interface{}{
						"pos_x": c.pos[0], "pos_y": c.pos[1], "pos_z": c.pos[2],
						"vel_x": c.vel[0], "vel_y": c.vel[1], "vel_z": c.vel[2],
						"facing": c.facing, "offset_x": c.offset[0], "offset_y": c.offset[1],
						"bindTime": c.bindTime, "bindToId": c.bindToId, "bindPos_x": c.bindPos[0], "bindPos_y": c.bindPos[1], "bindPos_z": c.bindPos[2], "bindFacing": c.bindFacing,
						"posFreeze": c.csf(CSF_posfreeze), "groundLevel": c.groundLevel,
					}
				case "Render":
					props["Render"] = map[string]interface{}{
						"angledraw": c.csf(CSF_angledraw), "anglerot_x": c.anglerot[0], "anglerot_y": c.anglerot[1], "anglerot_z": c.anglerot[2],
						"xshear": c.xshear, "projection": c.projection, "fLength": c.fLength,
						"angleDrawScale_x": c.angleDrawScale[0], "angleDrawScale_y": c.angleDrawScale[1], "zScale": c.zScale,
						"trans": c.trans, "alpha_src": c.alpha[0], "alpha_dst": c.alpha[1],
						"window_x": c.window[0], "window_y": c.window[1], "window_width": c.window[2], "window_height": c.window[3],
						"sprPriority": c.sprPriority, "layerNo": c.layerNo,
					}
				case "PauseTime":
					props["PauseTime"] = map[string]interface{}{
						"supertime": sys.supertime, "pausetime": sys.pausetime, "pauseMovetime": c.pauseMovetime, "superMovetime": c.superMovetime, "hitPauseTime": c.hitPauseTime,
					}
				case "Counters":
					props["Counters"] = map[string]interface{}{
						"airJumpCount": c.airJumpCount, "hitCount": c.hitCount, "guardCount": c.guardCount, "uniqHitCount": c.uniqHitCount,
					}
				case "Size":
					props["Size"] = map[string]interface{}{
						"sizeWidth_f": c.sizeWidth[0], "sizeWidth_b": c.sizeWidth[1], "edgeWidth_f": c.edgeWidth[0], "edgeWidth_b": c.edgeWidth[1],
						"sizeHeight_t": c.sizeHeight[0], "sizeHeight_b": c.sizeHeight[1], "sizeDepth_t": c.sizeDepth[0], "sizeDepth_b": c.sizeDepth[1],
						"edgeDepth_t": c.edgeDepth[0], "edgeDepth_b": c.edgeDepth[1], "pushPriority": c.pushPriority, "pushAffectTeam": c.pushAffectTeam,
						"depth": c.csf(CSF_depth), "depthEdge": c.csf(CSF_depthedge),
					}
				case "PalFX":
					if c.palfx != nil {
						props["PalFX"] = dumpPalFX(c.palfx)
					}
				case "AfterImage":
					if c.aimg != nil {
						props["AfterImage"] = dumpAfterImage(c.aimg)
					}
				case "ReadOnly_Char":
					props["ReadOnly_Char"] = map[string]interface{}{
						"helperId": c.helperId, "parentId": c.parentId, "time": c.ss.time, "anim": c.animNo,
						"moveContact": c.moveContact(), "ProjContacttime": c.projContactTime(BytecodeInt(-1)).ToI(),
						"attackMul": c.attackMul[0], "defencemul": float32(c.finalDefense / float64(c.gi().defenceBase) * 100),
						"attack":  (float32(c.gi().attackBase) * c.ocd().attackRatio / 100) * c.attackMul[0] * 100,
						"defence": float32(c.finalDefense * 100), "inguarddist": c.inguarddist,
						"prevstateno": c.ss.prevno, "prevanim": c.prevAnimNo, "prevStateType": pst, "prevMoveType": pmt,
					}
				case "ReadOnly_Distance":

					props["ReadOnly_Distance"] = map[string]interface{}{
						"screenpos_x": c.screenPosX() / c.localscl, "screenpos_y": c.screenPosY() / c.localscl,
						"camerapos_x": sys.cam.Pos[0] / c.localscl, "camerapos_y": (sys.cam.Pos[1] + sys.cam.aspectcorrection + sys.cam.zoomanchorcorrection) / c.localscl,
						"camerazoom": sys.cam.Scale, "backedgedist": c.backEdgeDist(), "backedgebodydist": c.backEdgeBodyDist(),
						"frontedgedist": c.frontEdgeDist(), "frontedgebodydist": c.frontEdgeBodyDist(),
						"p2dist_x": c.rdDistX(c.p2(), c).ToF(), "p2dist_y": c.rdDistY(c.p2(), c).ToF(), "p2dist_z": c.rdDistZ(c.p2(), c).ToF(),
						"p2bodydist_x": c.p2BodyDistX(c).ToF(), "p2bodydist_y": c.p2BodyDistY(c).ToF(), "p2bodydist_z": c.p2BodyDistZ(c).ToF(),
						"parentdist_x": c.rdDistX(c.parent(false), c).ToF(), "parentdist_y": c.rdDistY(c.parent(false), c).ToF(), "parentdist_z": c.rdDistZ(c.parent(false), c).ToF(),
						"rootdist_x": c.rdDistX(c.root(false), c).ToF(), "rootdist_y": c.rdDistY(c.root(false), c).ToF(), "rootdist_z": c.rdDistZ(c.root(false), c).ToF(),
						"stagebackedgedist": c.stageBackEdgeDist(), "stagefrontedgedist": c.stageFrontEdgeDist(),
						"topbounddist": c.topBoundDist(), "topboundbodydist": c.topBoundBodyDist(),
						"botbounddist": c.botBoundDist(), "botboundbodydist": c.botBoundBodyDist(),
					}
				case "ReadOnly_System":
					props["ReadOnly_System"] = map[string]interface{}{
						"gametime": sys.gameTime(), "roundtime": sys.tickCount, "roundstate": sys.roundState(),
						"matchno": sys.match, "roundno": sys.round, "win": c.win(), "lose": c.lose(), "drawgame": c.drawgame(),
						"introstate": sys.introState(), "outrostate": sys.outroState(), "matchover": sys.matchOver(),
						"roundsexisted": c.roundsExisted(), "ishometeam": c.teamside == sys.home, "roundswon": c.roundsWon(),
						"winko": c.winKO(), "wintime": c.winTime(), "winclutch": c.winClutch(),
						"winperfect": c.winPerfect(), "winspecial": c.winType(WT_Special), "winhyper": c.winType(WT_Hyper),
						"loseko": c.loseKO(), "losetime": c.loseTime(),
					}
				case "ReadOnly_Arrays":
					if len(c.enemyNearList) == 0 {
						c.enemyNearTrigger(0)
					}
					if len(c.p2EnemyList) == 0 {
						c.p2()
					}

					var hitbyList []string
					for i, hb := range c.hitby {
						if hb.time > 0 {
							prefix := "HitBy"
							if hb.not {
								prefix = "NotHitBy"
							}
							hitbyList = append(hitbyList, fmt.Sprintf("%d:[%s(%s) t:%d]", i, prefix, attrToStr(hb.flag), hb.time))
						}
					}

					var hoverList []string
					for i, ho := range c.hover {
						if ho.time > 0 {
							hoverList = append(hoverList, fmt.Sprintf("%d:[Hover(%s)->st:%d t:%d]", i, attrToStr(ho.attr), ho.stateno, ho.time))
						}
					}

					props["ReadOnly_Arrays"] = map[string]interface{}{
						"targets": c.targets, "children": c.children, "enemyNearList": c.enemyNearList, "p2EnemyList": c.p2EnemyList,
						"hitby": hitbyList, "hitoverride": hoverList,
					}
				}
			}
		}

		dump["debug_target"] = map[string]interface{}{
			"id":          c.id,
			"player":      c.playerNo + 1,
			"stateno":     c.ss.no,
			"prevstateno": c.ss.prevno,
			"life":        c.life,
			"power":       c.getPower(),
			"pos_x":       c.pos[0],
			"pos_y":       c.pos[1],
			"cnsvar":      c.cnsvar,
			"cnsfvar":     c.cnsfvar,
			"sysvar":      c.cnssysvar,
			"sysfvar":     c.cnssysfvar,
			"map":         c.mapArray,
			"props":       props,
		}
	}

	if s.reqExplods {
		explods := make([]map[string]interface{}, 0)
		for i := range sys.explods {
			for j, e := range sys.explods[i] {
				if e.id != IErr {
					explodData := map[string]interface{}{
						"player_idx": i,
						"item_index": j,
						"id":         e.id,
						"ownerid":    e.playerId,
						"anim":       e.animNo,
					}

					if s.propTargetType == "explod" && s.propPlayerIdx == i && s.propItemIndex == j && s.propTargetID == e.id {
						props := make(map[string]map[string]interface{})
						for _, cat := range s.reqCategories {
							switch cat {
							case "Time":
								props["Time"] = map[string]interface{}{
									"time": e.time, "removetime": e.removetime, "supermovetime": e.supermovetime, "pausemovetime": e.pausemovetime,
								}
							case "Position":
								props["Position"] = map[string]interface{}{
									"postype": e.postype, "space": e.space, "bindId": e.bindId, "bindtime": e.bindtime,
									"pos_x": e.pos[0], "pos_y": e.pos[1], "pos_z": e.pos[2],
									"relativePos_x": e.relativePos[0], "relativePos_y": e.relativePos[1], "relativePos_z": e.relativePos[2],
									"facing": e.facing, "vfacing": e.vfacing,
									"velocity_x": e.velocity[0], "velocity_y": e.velocity[1], "velocity_z": e.velocity[2],
									"friction_x": e.friction[0], "friction_y": e.friction[1], "friction_z": e.friction[2],
									"accel_x": e.accel[0], "accel_y": e.accel[1], "accel_z": e.accel[2],
								}
							case "Render":
								props["Render"] = map[string]interface{}{
									"scale_x": e.scale[0], "scale_y": e.scale[1],
									"sprpriority": e.sprpriority, "layerno": e.layerno,
									"animfreeze": e.animfreeze, "trans": e.trans,
									"alpha_src": e.alpha[0], "alpha_dst": e.alpha[1],
									"anglerot_x": e.anglerot[0], "anglerot_y": e.anglerot[1], "anglerot_z": e.anglerot[2],
									"xshear": e.xshear, "projection": e.projection, "fLength": e.fLength,
									"window_x": e.window[0], "window_y": e.window[1], "window_width": e.window[2], "window_height": e.window[3],
									"ignorehitpause": e.ignorehitpause,
								}
							case "AfterImage":
								if e.aimg != nil {
									props["AfterImage"] = dumpAfterImage(e.aimg)
								}
							case "PalFX":
								if e.palfx != nil {
									props["PalFX"] = dumpPalFX(e.palfx)
								}
							}
						}
						explodData["props"] = props
					}
					explods = append(explods, explodData)
				}
			}
		}
		dump["explods"] = explods
	}

	if s.reqProjs {
		projs := make([]map[string]interface{}, 0)
		for i := range sys.projs {
			for j, p := range sys.projs[i] {
				if p.id >= 0 {
					projData := map[string]interface{}{
						"player_idx": i,
						"item_index": j,
						"id":         p.id,
						"player":     i + 1,
					}

					if s.propTargetType == "proj" && s.propPlayerIdx == i && s.propItemIndex == j && s.propTargetID == p.id {
						props := make(map[string]map[string]interface{})
						for _, cat := range s.reqCategories {
							switch cat {
							case "Hits":
								props["Hits"] = map[string]interface{}{
									"totalhits": p.totalhits, "priorityPoints": p.priorityPoints,
								}
							case "Render":
								props["Render"] = map[string]interface{}{
									"scale_x": p.scale[0], "scale_y": p.scale[1],
									"anglerot_x": p.anglerot[0], "anglerot_y": p.anglerot[1], "anglerot_z": p.anglerot[2],
									"projection": p.projection, "fLength": p.fLength,
									"clsnScale_x": p.clsnScale[0], "clsnScale_y": p.clsnScale[1],
									"clsnAngle": p.clsnAngle, "zScale": p.zScale,
									"window_x": p.window[0], "window_y": p.window[1], "window_width": p.window[2], "window_height": p.window[3],
									"xshear": p.xshear, "sprpriority": p.sprpriority, "layerno": p.layerno,
								}
							case "Position":
								props["Position"] = map[string]interface{}{
									"pos_x": p.pos[0], "pos_y": p.pos[1], "pos_z": p.pos[2],
									"facing":     p.facing,
									"velocity_x": p.velocity[0], "velocity_y": p.velocity[1], "velocity_z": p.velocity[2],
									"remvelocity_x": p.remvelocity[0], "remvelocity_y": p.remvelocity[1], "remvelocity_z": p.remvelocity[2],
									"accel_x": p.accel[0], "accel_y": p.accel[1], "accel_z": p.accel[2],
									"velmul_x": p.velmul[0], "velmul_y": p.velmul[1], "velmul_z": p.velmul[2],
								}
							case "Time":
								props["Time"] = map[string]interface{}{
									"removetime": p.removetime, "supermovetime": p.supermovetime, "pausemovetime": p.pausemovetime,
									"curmisstime": p.curmisstime, "hitpause": p.hitpause, "time": p.time,
								}
							case "AfterImage":
								if p.aimg != nil {
									props["AfterImage"] = dumpAfterImage(p.aimg)
								}
							case "PalFX":
								if p.palfx != nil {
									props["PalFX"] = dumpPalFX(p.palfx)
								}
							}
						}
						projData["props"] = props
					}
					projs = append(projs, projData)
				}
			}
		}
		dump["projs"] = projs
	}

	return dump
}
