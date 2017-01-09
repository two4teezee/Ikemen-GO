package main

import (
	"math"
	"strings"
)

type EnvShake struct {
	time  int32
	freq  float32
	ampl  int32
	phase float32
}

func (es *EnvShake) clear() {
	*es = EnvShake{freq: float32(math.Pi / 3), ampl: -4,
		phase: float32(math.NaN())}
}
func (es *EnvShake) setDefPhase() {
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
	}
}
func (es *EnvShake) getOffset() float32 {
	if es.time > 0 {
		return float32(es.ampl) * 0.5 * float32(math.Sin(float64(es.phase)))
	}
	return 0
}

type BgcType int32

const (
	BT_Null BgcType = iota
	BT_Anim
	BT_Visible
	BT_Enable
	BT_PosSet
	BT_PosAdd
	BT_SinX
	BT_SinY
	BT_VelSet
	BT_VelAdd
)

type bgAction struct {
	offset      [2]float32
	sinoffset   [2]float32
	pos, vel    [2]float32
	radius      [2]float32
	sintime     [2]int32
	sinlooptime [2]int32
}

func (bga *bgAction) clear() {
	*bga = bgAction{}
}

type backGround struct {
	anim         Animation
	bga          bgAction
	id           int32
	start        [2]float32
	xofs         float32
	camstartx    float32
	delta        [2]float32
	xscale       [2]float32
	rasterx      [2]float32
	yscalestart  float32
	yscaledelta  float32
	actionno     int32
	startv       [2]float32
	startrad     [2]float32
	startsint    [2]int32
	startsinlt   [2]int32
	visible      bool
	active       bool
	positionlink bool
	toplayer     bool
	startrect    [4]int32
	windowdelta  [2]float32
}

func newBackGround(sff *Sff) *backGround {
	return &backGround{anim: *newAnimation(sff), delta: [2]float32{1, 1},
		xscale: [2]float32{1, 1}, rasterx: [2]float32{1, 1}, yscalestart: 100,
		actionno: -1, visible: true, active: true,
		startrect: [4]int32{-32768, -32768, 65535, 65535}}
}
func readBackGround(is IniSection, link *backGround,
	sff *Sff, at AnimationTable, camstartx float32) *backGround {
	bg := newBackGround(sff)
	bg.camstartx = camstartx
	typ, t := is["type"], 0
	if len(typ) == 0 {
		return bg
	}
	switch typ[0] {
	case 'N', 'n':
		t = 0
	case 'A', 'a':
		t = 1
	case 'P', 'p':
		t = 2
	case 'D', 'd':
		t = 3
	default:
		return bg
	}
	var tmp int32
	if is.ReadI32("layerno", &tmp) {
		bg.toplayer = tmp == 1
		if tmp < 0 || tmp > 1 {
			t = 3
		}
	}
	if t == 0 || t == 2 {
		var g, n int32
		if is.readI32ForStage("spriteno", &g, &n) {
			bg.anim.frames = []AnimFrame{*newAnimFrame()}
			bg.anim.frames[0].Group, bg.anim.frames[0].Number =
				I32ToI16(g), I32ToI16(n)
		}
	} else if t == 1 {
		if is.ReadI32("actionno", &bg.actionno) {
			if a := at.get(bg.actionno); a != nil {
				bg.anim = *a
			}
		}
	}
	is.ReadBool("positionlink", &bg.positionlink)
	if bg.positionlink && link != nil {
		bg.startv = link.startv
		bg.delta = link.delta
	}
	is.readF32ForStage("start", &bg.start[0], &bg.start[1])
	is.readF32ForStage("delta", &bg.delta[0], &bg.delta[1])
	if t != 1 {
		if is.ReadI32("mask", &tmp) {
			if tmp != 0 {
				bg.anim.mask = 0
			} else {
				bg.anim.mask = -1
			}
		}
		switch strings.ToLower(is["trans"]) {
		case "add":
			bg.anim.mask = 0
			bg.anim.srcAlpha = 255
			bg.anim.dstAlpha = 255
		case "add1":
			bg.anim.mask = 0
			bg.anim.srcAlpha = 255
			bg.anim.dstAlpha = ^255
			var s, d int32 = 255, 255
			if is.readI32ForStage("alpha", &s, &d) {
				bg.anim.srcAlpha = int16(Min(255, s))
				bg.anim.dstAlpha = ^int16(Max(0, Min(255, s)))
			}
		case "addalpha":
			bg.anim.mask = 0
			s, d := int32(bg.anim.srcAlpha), int32(bg.anim.dstAlpha)
			if is.readI32ForStage("alpha", &s, &d) {
				bg.anim.srcAlpha = int16(Min(255, s))
				bg.anim.dstAlpha = int16(Max(0, Min(255, s)))
				if bg.anim.srcAlpha == 1 && bg.anim.dstAlpha == 255 {
					bg.anim.srcAlpha = 0
				}
			}
		case "sub":
			bg.anim.mask = 0
			bg.anim.srcAlpha = 1
			bg.anim.dstAlpha = 255
		case "none":
			bg.anim.srcAlpha = -1
			bg.anim.dstAlpha = 0
		}
	}
	if is.readI32ForStage("tile", &bg.anim.tile[2], &bg.anim.tile[3]) {
		if t == 2 {
			bg.anim.tile[3] = 0
		}
	}
	if t == 2 {
		var tw, bw int32
		if is.readI32ForStage("width", &tw, &bw) {
			if (tw != 0 || bw != 0) && len(bg.anim.frames) > 0 {
				if spr := sff.GetSprite(
					bg.anim.frames[0].Group, bg.anim.frames[0].Number); spr != nil {
					bg.xscale[0] = float32(tw) / float32(spr.Size[0])
					bg.xscale[1] = float32(bw) / float32(spr.Size[0])
					bg.xofs = -float32(tw)/2 + float32(spr.Offset[0])*bg.xscale[0]
				}
			}
		} else {
			is.readF32ForStage("xscale", &bg.rasterx[0], &bg.rasterx[1])
		}
		is.ReadF32("yscalestart", &bg.yscalestart)
		is.ReadF32("yscaledelta", &bg.yscaledelta)
	} else {
		is.ReadI32("tilespacing", &bg.anim.tile[0])
		bg.anim.tile[1] = bg.anim.tile[0]
		if bg.actionno < 0 && len(bg.anim.frames) > 0 {
			if spr := sff.GetSprite(
				bg.anim.frames[0].Group, bg.anim.frames[0].Number); spr != nil {
				bg.anim.tile[0] += int32(spr.Size[0])
				bg.anim.tile[1] += int32(spr.Size[1])
			}
		}
	}
	if is.readI32ForStage("window", &bg.startrect[0], &bg.startrect[1],
		&bg.startrect[2], &bg.startrect[3]) {
		bg.startrect[2] = Max(0, bg.startrect[2]+1-bg.startrect[0])
		bg.startrect[3] = Max(0, bg.startrect[3]+1-bg.startrect[1])
	}
	is.readF32ForStage("windowdelta", &bg.windowdelta[0], &bg.windowdelta[1])
	is.ReadI32("id", &bg.id)
	is.readF32ForStage("velocity", &bg.startv[0], &bg.startv[1])
	for i := 0; i < 2; i++ {
		var name string
		if i == 0 {
			name = "sin.x"
		} else {
			name = "sin.y"
		}
		r, slt, st := float32(math.NaN()), float32(math.NaN()), float32(math.NaN())
		if is.readF32ForStage(name, &r, &slt, &st) {
			if !math.IsNaN(float64(r)) {
				bg.startrad[i], bg.bga.radius[i] = r, r
			}
			if !math.IsNaN(float64(slt)) {
				var slti int32
				is.readI32ForStage(name, &tmp, &slti)
				bg.startsinlt[i], bg.bga.sinlooptime[i] = slti, slti
			}
			if bg.bga.sinlooptime[i] > 0 && !math.IsNaN(float64(st)) {
				bg.bga.sintime[i] = int32(st*float32(bg.bga.sinlooptime[i])/360) %
					bg.bga.sinlooptime[i]
				if bg.bga.sintime[i] < 0 {
					bg.bga.sintime[i] += bg.bga.sinlooptime[i]
				}
				bg.startsint[i] = bg.bga.sintime[i]
			}
		}
	}
	return bg
}
func (bg *backGround) reset() {
	bg.anim.Reset()
	bg.bga.clear()
	bg.bga.vel = bg.startv
	bg.bga.radius = bg.startrad
	bg.bga.sintime = bg.startsint
	bg.bga.sinlooptime = bg.startsinlt
}

type bgCtrl struct {
	bg           []*backGround
	currenttime  int32
	starttime    int32
	endtime      int32
	looptime     int32
	_type        BgcType
	x, y         float32
	v            [3]int32
	positionlink bool
	flag         bool
	idx          int
}

func newBgCtrl() *bgCtrl {
	return &bgCtrl{looptime: -1, x: float32(math.NaN()), y: float32(math.NaN())}
}
func (bgc *bgCtrl) read(is IniSection, idx int) {
	xy := false
	switch is["type"] {
	case "anim":
		bgc._type = BT_Anim
	case "visible":
		bgc._type = BT_Visible
	case "enable":
		bgc._type = BT_Enable
	case "null":
		bgc._type = BT_Null
	case "posset":
		bgc._type = BT_PosSet
		xy = true
	case "posadd":
		bgc._type = BT_PosAdd
		xy = true
	case "sinx":
		bgc._type = BT_SinX
	case "siny":
		bgc._type = BT_SinY
	case "velset":
		bgc._type = BT_VelSet
		xy = true
	case "veladd":
		bgc._type = BT_VelAdd
		xy = true
	}
	is.ReadI32("time", &bgc.starttime)
	bgc.endtime = bgc.starttime
	is.readI32ForStage("time", &bgc.starttime, &bgc.endtime, &bgc.looptime)
	is.ReadBool("positionlink", &bgc.positionlink)
	if xy {
		is.readF32ForStage("x", &bgc.x)
		is.readF32ForStage("y", &bgc.y)
	} else if is.ReadF32("value", &bgc.x) {
		is.readI32ForStage("value", &bgc.v[0], &bgc.v[1], &bgc.v[2])
	}
}

type bgctNode struct {
	bgc      []*bgCtrl
	waitTime int32
}
type bgcTimeLine struct {
	line []bgctNode
	al   []*bgCtrl
}

func (bgct *bgcTimeLine) clear() {
	*bgct = bgcTimeLine{}
}
func (bgct *bgcTimeLine) add(bgc *bgCtrl) {
	if bgc.looptime >= 0 && bgc.endtime > bgc.looptime {
		bgc.endtime = bgc.looptime
	}
	if bgc.starttime < 0 || bgc.starttime > bgc.endtime ||
		bgc.looptime >= 0 && bgc.starttime >= bgc.looptime {
		return
	}
	wtime := int32(0)
	if bgc.currenttime != 0 {
		if bgc.looptime < 0 {
			return
		}
		wtime += bgc.looptime - bgc.currenttime
	}
	wtime += bgc.starttime
	bgc.currenttime = bgc.starttime
	if wtime < 0 {
		bgc.currenttime -= wtime
		wtime = 0
	}
	i := 0
	for ; ; i++ {
		if i == len(bgct.line) {
			bgct.line = append(bgct.line,
				bgctNode{bgc: []*bgCtrl{bgc}, waitTime: wtime})
			return
		}
		if wtime <= bgct.line[i].waitTime {
			break
		}
		wtime -= bgct.line[i].waitTime
	}
	if wtime == bgct.line[i].waitTime {
		bgct.line[i].bgc = append(bgct.line[i].bgc, bgc)
	} else {
		tmp := append(bgct.line[:i:i],
			bgctNode{bgc: []*bgCtrl{bgc}, waitTime: wtime})
		bgct.line[i].waitTime -= wtime
		bgct.line = append(tmp, bgct.line...)
	}
}

type stageCamera struct {
	startx         int32
	boundleft      int32
	boundright     int32
	boundhigh      int32
	verticalfollow float32
	tension        int32
	floortension   int32
	overdrawlow    int32
}
type stageShadow struct {
	intensity int32
	color     uint32
	yscale    float32
	fadeend   int32
	fadebgn   int32
}
type stagePlayer struct {
	startx, starty int32
}
type Stage struct {
	def         string
	bgmusic     string
	name        string
	displayname string
	author      string
	sff         *Sff
	at          AnimationTable
	bg          []backGround
	bgc         []bgCtrl
	bgct        bgcTimeLine
	bga         bgAction
	cam         stageCamera
	sdw         stageShadow
	p           [2]stagePlayer
	leftbound   float32
	rightbound  float32
	screenleft  int32
	screenright int32
	zoffset     int32
	zoffsetlink int32
	ztopscale   float32
	reflection  int32
	hires       bool
	resetbg     bool
	debugbg     bool
	localcoord  [2]int32
	localscl    float32
	scale       [2]float32
	drawOffsetY float32
}

func newStage(def string) *Stage {
	s := &Stage{def: def, leftbound: float32(math.NaN()),
		rightbound: float32(math.NaN()), screenleft: 15, screenright: 15,
		zoffsetlink: -1, ztopscale: 1, resetbg: true,
		localcoord: [2]int32{320, 240}, localscl: float32(sys.gameWidth / 320),
		scale: [2]float32{1, 1}}
	s.cam.verticalfollow = 0.2
	s.cam.tension = 50
	s.sdw.intensity = 128
	s.sdw.color = 0x808080
	s.sdw.yscale = 0.4
	s.sdw.fadeend = math.MinInt32
	s.sdw.fadebgn = math.MinInt32
	s.p[0].startx, s.p[1].startx = -70, 70
	return s
}
func LoadStage(def string) (*Stage, error) {
	s := newStage(def)
	str, err := LoadText(def)
	if err != nil {
		return nil, err
	}
	s.sff = &Sff{}
	lines, i := SplitAndTrim(str, "\n"), 0
	s.at = ReadAnimationTable(s.sff, lines, &i)
	i = 0
	defmap := make(map[string][]IniSection)
	for i < len(lines) {
		is, name, _ := ReadIniSection(lines, &i)
		if i := strings.IndexAny(name, " \t"); i >= 0 {
			if name[:i] == "bg" {
				defmap["bg"] = append(defmap["bg"], is)
			}
		} else {
			defmap[name] = append(defmap[name], is)
		}
	}
	if sec := defmap["info"]; len(sec) > 0 {
		var ok bool
		s.name, ok, _ = sec[0].getText("name")
		if !ok {
			s.name = def
		}
		s.displayname, ok, _ = sec[0].getText("displayname")
		if !ok {
			s.displayname = s.name
		}
		s.author, _, _ = sec[0].getText("author")
	}
	if sec := defmap["camera"]; len(sec) > 0 {
		sec[0].ReadI32("startx", &s.cam.startx)
		sec[0].ReadI32("boundleft", &s.cam.boundleft)
		sec[0].ReadI32("boundright", &s.cam.boundright)
		sec[0].ReadI32("boundhigh", &s.cam.boundhigh)
		sec[0].ReadF32("verticalfollow", &s.cam.verticalfollow)
		sec[0].ReadI32("tension", &s.cam.tension)
		sec[0].ReadI32("floortension", &s.cam.floortension)
		sec[0].ReadI32("overdrawlow", &s.cam.overdrawlow)
	}
	if sec := defmap["playerinfo"]; len(sec) > 0 {
		sec[0].ReadI32("p1startx", &s.p[0].startx)
		sec[0].ReadI32("p1starty", &s.p[0].starty)
		sec[0].ReadI32("p2startx", &s.p[1].startx)
		sec[0].ReadI32("p2starty", &s.p[1].starty)
		sec[0].ReadF32("leftbound", &s.leftbound)
		sec[0].ReadF32("rightbound", &s.rightbound)
	}
	if sec := defmap["scaling"]; len(sec) > 0 {
		sec[0].ReadF32("topscale", &s.ztopscale)
	}
	if sec := defmap["bound"]; len(sec) > 0 {
		sec[0].ReadI32("screenleft", &s.screenleft)
		sec[0].ReadI32("screenright", &s.screenright)
	}
	if sec := defmap["stageinfo"]; len(sec) > 0 {
		sec[0].ReadI32("zoffset", &s.zoffset)
		sec[0].ReadI32("zoffsetlink", &s.zoffsetlink)
		sec[0].ReadBool("hires", &s.hires)
		sec[0].ReadBool("resetbg", &s.resetbg)
		sec[0].readI32ForStage("localcoord", &s.localcoord[0], &s.localcoord[1])
		sec[0].ReadF32("xscale", &s.scale[0])
		sec[0].ReadF32("yscale", &s.scale[1])
	}
	reflect := true
	if sec := defmap["shadow"]; len(sec) > 0 {
		var tmp int32
		if sec[0].ReadI32("intensity", &tmp) {
			s.sdw.intensity = Max(0, Min(255, tmp))
		}
		var r, g, b int32
		if sec[0].readI32ForStage("color", &r, &g, &b) {
			r, g, b = Max(0, Min(255, r)), Max(0, Min(255, g)), Max(0, Min(255, b))
		}
		s.sdw.color = uint32(r<<16 | g<<8 | b)
		sec[0].ReadF32("yscale", &s.sdw.yscale)
		sec[0].ReadBool("reflect", &reflect)
		sec[0].readI32ForStage("fade.range", &s.sdw.fadeend, &s.sdw.fadebgn)
	}
	if reflect {
		if sec := defmap["reflection"]; len(sec) > 0 {
			var tmp int32
			if sec[0].ReadI32("intensity", &tmp) {
				s.reflection = Max(0, Min(255, tmp))
			}
		}
	}
	if sec := defmap["music"]; len(sec) > 0 {
		s.bgmusic = sec[0]["bgmusic"]
	}
	if sec := defmap["bgdef"]; len(sec) > 0 {
		if sec[0].LoadFile("spr", def, func(filename string) error {
			sff, err := LoadSff(filename, false)
			if err != nil {
				return err
			}
			*s.sff = *sff
			return nil
		}); err != nil {
			return nil, err
		}
		sec[0].ReadBool("debugbg", &s.debugbg)
	}
	var bglink *backGround
	for _, bgsec := range defmap["bg"] {
		if len(s.bg) > 0 && s.bg[len(s.bg)-1].positionlink {
			bglink = &s.bg[len(s.bg)-1]
		}
		s.bg = append(s.bg, *readBackGround(bgsec, bglink,
			s.sff, s.at, float32(s.cam.startx)))
	}
	var bgcdef bgCtrl
	i = 0
	for i < len(lines) {
		is, name, _ := ReadIniSection(lines, &i)
		switch name {
		case "bgctrldef":
			bgcdef.bg, bgcdef.looptime = nil, -1
			if ids := is.readI32CsvForStage("ctrlid"); len(ids) > 0 &&
				(len(ids) > 1 || ids[0] != -1) {
				kishutu := make(map[int32]bool)
				for _, id := range ids {
					if kishutu[id] {
						continue
					}
					bgcdef.bg = append(bgcdef.bg, s.getBg(id)...)
					kishutu[id] = true
				}
			} else {
				for _, b := range s.bg {
					bgcdef.bg = append(bgcdef.bg, &b)
				}
			}
			is.ReadI32("looptime", &bgcdef.looptime)
		case "bgctrl":
			bgc := newBgCtrl()
			*bgc = bgcdef
			if ids := is.readI32CsvForStage("ctrlid"); len(ids) > 0 {
				if len(ids) > 1 || ids[0] != -1 {
					kishutu := make(map[int32]bool)
					for _, id := range ids {
						if kishutu[id] {
							continue
						}
						bgc.bg = append(bgc.bg, s.getBg(id)...)
						kishutu[id] = true
					}
				} else {
					for _, b := range s.bg {
						bgc.bg = append(bgc.bg, &b)
					}
				}
			}
			bgc.read(is, len(s.bgc))
			s.bgc = append(s.bgc, *bgc)
		}
	}
	s.localscl = float32(sys.gameWidth) / float32(s.localcoord[0])
	if math.IsNaN(float64(s.leftbound)) {
		s.leftbound = 1000
	} else {
		s.leftbound *= s.localscl
	}
	if math.IsNaN(float64(s.rightbound)) {
		s.rightbound = 1000
	} else {
		s.rightbound *= s.localscl
	}
	link, zlink := 0, -1
	for i, b := range s.bg {
		if b.positionlink && i > 0 {
			s.bg[i].start[0] += s.bg[link].start[0]
			s.bg[i].start[1] += s.bg[link].start[1]
		} else {
			link = i
		}
		if s.zoffsetlink >= 0 && zlink < 0 && b.id == s.zoffsetlink {
			zlink = i
			s.zoffset += int32(b.start[1] * s.scale[1])
		}
	}
	ratio1 := float32(s.localcoord[0]) / float32(s.localcoord[1])
	ratio2 := float32(sys.gameWidth) / 240
	if ratio1 > ratio2 {
		s.drawOffsetY =
			MinF(float32(s.localcoord[1])*s.localscl*0.5*(ratio1/ratio2-1),
				float32(Max(0, s.cam.overdrawlow)))
	}
	return s, nil
}
func (s *Stage) getBg(id int32) (bg []*backGround) {
	if id >= 0 {
		for _, b := range s.bg {
			if b.id == id {
				bg = append(bg, &b)
			}
		}
	}
	return
}
func (s *Stage) reset() {
	s.bga.clear()
	for i := range s.bg {
		s.bg[i].reset()
	}
	for i := range s.bgc {
		s.bgc[i].currenttime = 0
	}
	s.bgct.clear()
	for i := len(s.bgc) - 1; i >= 0; i-- {
		s.bgct.add(&s.bgc[i])
	}
}