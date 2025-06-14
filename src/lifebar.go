package main

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

type FinishType int32

const (
	FT_NotYet FinishType = iota
	FT_KO
	FT_DKO
	FT_TO
	FT_TODraw
)

type WinType int32

const (
	WT_Normal WinType = iota
	WT_Special
	WT_Hyper
	WT_Cheese
	WT_Time
	WT_Throw
	WT_Suicide
	WT_Teammate
	WT_Perfect
	WT_NumTypes
	WT_PNormal
	WT_PSpecial
	WT_PHyper
	WT_PCheese
	WT_PTime
	WT_PThrow
	WT_PSuicide
	WT_PTeammate
)

func (wt *WinType) SetPerfect() {
	if *wt >= WT_Normal && *wt < WT_Perfect {
		*wt += WT_PNormal - WT_Normal
	}
}

type FightFx struct {
	fat        AnimationTable
	fsff       *Sff
	fsnd       *Snd
	fx_scale   float32
	localcoord [2]float32
}

func newFightFx() *FightFx {
	return &FightFx{
		fsff:       &Sff{},
		fx_scale:   1.0,
		localcoord: [...]float32{320, 240},
	}
}

func loadFightFx(def string) error {
	str, err := LoadText(def)
	if err != nil {
		return err
	}
	ffx := newFightFx()
	prefix := ""
	lines, i := SplitAndTrim(str, "\n"), 0
	info, files := true, true
	for i < len(lines) {
		// Parse each ini section
		is, name, _ := ReadIniSection(lines, &i)
		switch name {
		case "info":
			// Read info for FightFx storing and scaling
			if info {
				info = false
				var ok bool
				prefix, ok, _ = is.getText("prefix")
				if !ok || prefix == "" {
					return Error("A prefix must be declared")
				}
				prefix = strings.ToLower(prefix)
				if prefix == "f" || prefix == "s" {
					return Error(fmt.Sprintf("%v prefix is reserved for the system and cannot be used", strings.ToUpper(prefix)))
				}
				is.ReadF32("fx.scale", &ffx.fx_scale)
				// Read localcoord
				// Merely used for automatic fx.scale adjustment
				// If localcoord is not available, we use the old method for scaling
				if is.ReadF32("localcoord", &ffx.localcoord[0], &ffx.localcoord[1]) {
					ffx.fx_scale *= float32(320 / ffx.localcoord[0])
				} else {
					ffx.fx_scale *= sys.lifebarScale
				}
			}
		case "files":
			// Read files section
			if files {
				files = false
				if is.LoadFile("sff", []string{def, sys.motifDir, "", "data/"},
					func(filename string) error {
						s, err := loadSff(filename, false)
						if err != nil {
							return err
						}
						*ffx.fsff = *s
						return nil
					}); err != nil {
					return err
				}
				if is.LoadFile("air", []string{def, sys.motifDir, "", "data/"},
					func(filename string) error {
						str, err := LoadText(filename)
						if err != nil {
							return err
						}
						lines, i := SplitAndTrim(str, "\n"), 0
						ffx.fat = ReadAnimationTable(ffx.fsff, &ffx.fsff.palList, lines, &i)
						return nil
					}); err != nil {
					return err
				}
				if is.LoadFile("snd", []string{def, sys.motifDir, "", "data/"},
					func(filename string) error {
						ffx.fsnd, err = LoadSnd(filename)
						return err
					}); err != nil {
					return err
				}
			}
		}
	}
	// Set fx scale to anims
	for _, a := range ffx.fat {
		a.start_scale = [...]float32{ffx.fx_scale, ffx.fx_scale}
	}
	if sys.ffx[prefix] == nil {
		sys.ffxRegexp += "|^(" + prefix + ")"
	}
	sys.ffx[prefix] = ffx
	return nil
}

type LbText struct {
	font       [6]int32
	text       string
	lay        Layout
	palfx      *PalFX
	frgba      [4]float32 // ttf fonts
	forcecolor bool
	pfxinit    int32
}

func newLbText(align int32) *LbText {
	return &LbText{
		font:  [...]int32{-1, 0, align, 255, 255, 255},
		palfx: newPalFX(),
		frgba: [...]float32{1.0, 1.0, 1.0, 1.0},
	}
}

func readLbText(pre string, is IniSection, str string, ln int16, f []*Fnt, align int32) *LbText {
	txt := newLbText(align)
	txt.font[3], txt.font[4], txt.font[5] = -1, -1, -1
	is.ReadI32(pre+"font", &txt.font[0], &txt.font[1], &txt.font[2],
		&txt.font[3], &txt.font[4], &txt.font[5])
	if txt.font[0] >= 0 && int(txt.font[0]) < len(f) && f[txt.font[0]] == nil {
		sys.errLog.Printf("Undefined font %v referenced by lifebar parameter: %v\n", txt.font[0], pre+"font")
		txt.font[0] = -1
	}
	if _, ok := is[pre+"text"]; ok {
		txt.text, _, _ = is.getText(pre + "text")
	} else {
		txt.text = str
	}
	txt.lay = *ReadLayout(pre, is, ln)
	if txt.font[3] >= 0 && txt.font[4] >= 0 && txt.font[5] >= 0 {
		txt.SetColor(txt.font[3], txt.font[4], txt.font[5])
	}
	txt.pfxinit = ReadPalFX(pre+"palfx.", is, txt.palfx)
	return txt
}

func (txt *LbText) SetColor(r, g, b int32) {
	txt.forcecolor = true
	txt.palfx.setColor(r, g, b)
}

func (txt *LbText) step() {
	if txt.palfx != nil && !txt.forcecolor {
		txt.palfx.step()
	}
}

func (txt *LbText) resetTxtPfx() {
	txt.palfx.time = txt.pfxinit
}

type LbBgTextSnd struct {
	pos         [2]int32
	text        LbText
	bg          AnimLayout
	time        int32
	displaytime int32
	snd         [2]int32
	sndtime     int32
	timer       int32
}

func newLbBgTextSnd() LbBgTextSnd {
	return LbBgTextSnd{snd: [2]int32{-1}}
}

func readLbBgTextSnd(pre string, is IniSection,
	sff *Sff, at AnimationTable, ln int16, f []*Fnt) LbBgTextSnd {
	bts := newLbBgTextSnd()
	is.ReadI32(pre+"pos", &bts.pos[0], &bts.pos[1])
	bts.text = *readLbText(pre+"text.", is, "", ln, f, 0)
	bts.bg = *ReadAnimLayout(pre+"bg.", is, sff, at, ln)
	is.ReadI32(pre+"time", &bts.time)
	is.ReadI32(pre+"displaytime", &bts.displaytime)
	is.ReadI32(pre+"snd", &bts.snd[0], &bts.snd[1])
	bts.sndtime = bts.time
	is.ReadI32(pre+"sndtime", &bts.sndtime)
	return bts
}

func (bts *LbBgTextSnd) step(snd *Snd) {
	if bts.timer == bts.sndtime {
		snd.play(bts.snd, 100, 0, 0, 0, 0)
	}
	if bts.timer >= bts.time {
		bts.bg.Action()
	}
	bts.timer++
	bts.text.step()
}

func (bts *LbBgTextSnd) reset() {
	bts.timer = 0
	bts.bg.Reset()
	bts.text.resetTxtPfx()
}

func (bts *LbBgTextSnd) bgDraw(layerno int16) {
	if bts.timer > bts.time && bts.timer <= bts.time+bts.displaytime {
		bts.bg.Draw(float32(bts.pos[0])+sys.lifebarOffsetX, float32(bts.pos[1]), layerno, sys.lifebarScale)
	}
}

func (bts *LbBgTextSnd) draw(layerno int16, f []*Fnt) {
	if bts.timer > bts.time && bts.timer <= bts.time+bts.displaytime &&
		bts.text.font[0] >= 0 && int(bts.text.font[0]) < len(f) && f[bts.text.font[0]] != nil {
		bts.text.lay.DrawText(float32(bts.pos[0])+sys.lifebarOffsetX, float32(bts.pos[1]), sys.lifebarScale, layerno,
			bts.text.text, f[bts.text.font[0]], bts.text.font[1], bts.text.font[2], bts.text.palfx, bts.text.frgba)
	}
}

// Reads multiple lifebar values e.g. multiple front elements
func readMultipleValues(pre string, name string, is IniSection, sff *Sff, at AnimationTable) map[int32]*AnimLayout {
	result := make(map[int32]*AnimLayout)
	r, _ := regexp.Compile(pre + name + "[0-9]+\\.")
	for k := range is {
		if r.MatchString(k) {
			re := regexp.MustCompile("[0-9]+")
			submatchall := re.FindAllString(k, -1)
			if len(submatchall) == 2 {
				v := Atoi(submatchall[1])
				if _, ok := result[v]; !ok {
					result[v] = ReadAnimLayout(pre+name+fmt.Sprintf("%v", v)+".", is, sff, at, 0)
				}
			}
		}
	}
	return result
}

// Float version of readMultipleValues
func readMultipleValuesF(pre string, name string, is IniSection, sff *Sff, at AnimationTable) map[float32]*AnimLayout {
	result := make(map[float32]*AnimLayout)
	r, _ := regexp.Compile(pre + name + "[0-9]+\\.")
	for k := range is {
		if r.MatchString(k) {
			re := regexp.MustCompile("[0-9]+")
			submatchall := re.FindAllString(k, -1)
			if len(submatchall) == 2 {
				v := Atof(submatchall[1])
				if _, ok := result[float32(v)]; !ok {
					result[float32(v)] = ReadAnimLayout(pre+name+fmt.Sprintf("%v", v)+".", is, sff, at, 0)
				}
			}
		}
	}
	return result
}

// Text version of readMultipleValues
func readMultipleLbText(pre string, name string, is IniSection, fmtstr string, ln int16, f []*Fnt, align int32) map[int32]*LbText {
	result := make(map[int32]*LbText)
	r, _ := regexp.Compile(pre + name + "[0-9]+\\.")
	for k := range is {
		if r.MatchString(k) {
			re := regexp.MustCompile("[0-9]+")
			submatchall := re.FindAllString(k, -1)
			if len(submatchall) >= 1 {
				v := Atoi(submatchall[len(submatchall)-1])
				if _, ok := result[v]; !ok {
					result[v] = readLbText(pre+name+fmt.Sprintf("%v", v)+".", is, fmtstr, ln, f, align)
				}
			}
		}
	}
	return result
}

// Calculates the visible portion (rect) of a fill bar element
func calcBarFillRect(pos int32, range_ [2]int32, offset, scale, screenScale, midPos float32, fill float32) (start, size int32) {
	isDescending := range_[0] > range_[1]
	var r0, r1 int32

	if isDescending {
		r0, r1 = range_[1], range_[0]
	} else {
		r0, r1 = range_[0], range_[1]
	}

	fillLength := float32(r1 - r0 + 1)
	size = int32((fillLength * scale * fill * screenScale) + 0.5)

	base := float32(pos + r0)
	start = int32(((base+offset)*scale+midPos)*screenScale + 0.5)

	if isDescending {
		start = int32(((float32(pos+r1+1)+offset)*scale+midPos)*screenScale+0.5) - size
	}
	return
}

type HealthBar struct {
	pos        [2]int32
	range_x    [2]int32
	range_y    [2]int32
	bg0        AnimLayout
	bg1        AnimLayout
	bg2        AnimLayout
	top        AnimLayout
	mid        AnimLayout
	red        map[int32]*AnimLayout
	front      map[float32]*AnimLayout
	shift      AnimLayout
	warn       AnimLayout
	warn_range [2]int32
	value      LbText
	toplife    float32
	oldlife    float32
	midlife    float32
	midlifeMin float32
	mlifetime  int32
	mid_shift  bool
	mid_freeze bool
	mid_delay  int32
	mid_mult   float32
	mid_steps  float32
	gethit     bool
	scalefill  bool
}

func newHealthBar() *HealthBar {
	return &HealthBar{
		oldlife:    1,
		midlife:    1,
		midlifeMin: 1,
		red:        make(map[int32]*AnimLayout),
		front:      make(map[float32]*AnimLayout),
		mid_freeze: true,
		mid_delay:  30,
		mid_mult:   1.0,
		mid_steps:  8.0,
	}
}

func readHealthBar(pre string, is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt) *HealthBar {
	hb := newHealthBar()
	is.ReadI32(pre+"pos", &hb.pos[0], &hb.pos[1])
	is.ReadI32(pre+"range.x", &hb.range_x[0], &hb.range_x[1])
	is.ReadI32(pre+"range.y", &hb.range_y[0], &hb.range_y[1])
	hb.bg0 = *ReadAnimLayout(pre+"bg0.", is, sff, at, 0)
	hb.bg1 = *ReadAnimLayout(pre+"bg1.", is, sff, at, 0)
	hb.bg2 = *ReadAnimLayout(pre+"bg2.", is, sff, at, 0)
	hb.top = *ReadAnimLayout(pre+"top.", is, sff, at, 0)
	hb.mid = *ReadAnimLayout(pre+"mid.", is, sff, at, 0)
	hb.front[0] = ReadAnimLayout(pre+"front.", is, sff, at, 0)
	for k, v := range readMultipleValuesF(pre, "front", is, sff, at) {
		hb.front[k] = v
	}
	hb.shift = *ReadAnimLayout(pre+"shift.", is, sff, at, 0)
	hb.red[0] = ReadAnimLayout(pre+"red.", is, sff, at, 0)
	for k, v := range readMultipleValues(pre, "red", is, sff, at) {
		hb.red[k] = v
	}
	hb.value = *readLbText(pre+"value.", is, "%d", 0, f, 0)
	is.ReadBool("mid.shift", &hb.mid_shift)
	is.ReadBool("mid.freeze", &hb.mid_freeze)
	is.ReadI32("mid.delay", &hb.mid_delay)
	is.ReadF32("mid.mult", &hb.mid_mult)
	is.ReadF32("mid.steps", &hb.mid_steps)
	hb.mid_steps = MaxF(1, hb.mid_steps)
	is.ReadI32(pre+"warn.range", &hb.warn_range[0], &hb.warn_range[1])
	hb.warn = *ReadAnimLayout(pre+"warn.", is, sff, at, 0)
	is.ReadBool(pre+"scalefill", &hb.scalefill)
	return hb
}

func (hb *HealthBar) step(ref int, hbr *HealthBar) {
	var life float32 = float32(sys.chars[ref][0].life) / float32(sys.chars[ref][0].lifeMax)
	//redlife := (float32(sys.chars[ref][0].life) + float32(sys.chars[ref][0].redLife)) / float32(sys.chars[ref][0].lifeMax)
	var redVal int32 = sys.chars[ref][0].redLife - sys.chars[ref][0].life
	var getHit bool = (sys.chars[ref][0].receivedHits != 0 || sys.chars[ref][0].ss.moveType == MT_H) && !sys.chars[ref][0].scf(SCF_over_ko)

	if hbr.toplife > life {
		hbr.toplife += (life - hbr.toplife) / 2
	} else {
		hbr.toplife = life
	}

	// Element shifting gradient
	hb.shift.anim.srcAlpha = int16(255 * (1 - life))
	hb.shift.anim.dstAlpha = int16(255 * life)

	if !hb.mid_freeze && getHit && !hb.gethit && len(hb.mid.anim.frames) > 0 {
		hbr.mlifetime = hb.mid_delay
		hbr.midlife = hbr.oldlife
		hbr.midlifeMin = hbr.oldlife
	}
	hb.gethit = getHit
	if hb.mid_freeze && getHit && len(hb.mid.anim.frames) > 0 {
		if hbr.mlifetime < hb.mid_delay {
			hbr.mlifetime = hb.mid_delay
			hbr.midlife = hbr.oldlife
			hbr.midlifeMin = hbr.oldlife
		}
	} else {
		if hbr.mlifetime > 0 {
			hbr.mlifetime--
		}
		if len(hb.mid.anim.frames) > 0 && hbr.mlifetime <= 0 && life < hbr.midlifeMin {
			hbr.midlifeMin += (life - hbr.midlifeMin) * (1 / (12 - (life-hbr.midlifeMin)*144)) * hb.mid_mult
			if hbr.midlifeMin < life {
				hbr.midlifeMin = life
			}
		} else {
			hbr.midlifeMin = life
		}
		if (len(hb.mid.anim.frames) == 0 || hbr.mlifetime <= 0) && hbr.midlife > hbr.midlifeMin {
			hbr.midlife += (hbr.midlifeMin - hbr.midlife) / hb.mid_steps
		}
		hbr.oldlife = life
	}

	mlmin := MaxF(hbr.midlifeMin, life)
	if hbr.midlife < mlmin {
		hbr.midlife += (mlmin - hbr.midlife) / 2
	}

	hb.bg0.Action()
	hb.bg1.Action()
	hb.bg2.Action()
	hb.top.Action()
	hb.mid.Action()
	hb.value.step()
	// Multiple front elements - red life
	if sys.lifebar.redlifebar {
		var rv int32
		for k := range hb.red {
			if k > rv && redVal >= k {
				rv = k
			}
		}
		hb.red[rv].Action()
	}

	// Multiple front elements - life
	var fv float32
	for k := range hb.front {
		if k > fv && life >= k/100 {
			fv = k
		}
	}
	hb.front[fv].Action()

	hb.shift.Action()
	hb.warn.Action()
}

func (hb *HealthBar) reset() {
	hb.bg0.Reset()
	hb.bg1.Reset()
	hb.bg2.Reset()
	hb.top.Reset()
	hb.mid.Reset()
	for i := range hb.front {
		hb.front[i].Reset()
	}
	hb.shift.Reset()
	hb.shift.anim.srcAlpha = 0
	hb.shift.anim.dstAlpha = 255
	for i := range hb.red {
		hb.red[i].Reset()
	}
	hb.warn.Reset()
}

func (hb *HealthBar) bgDraw(layerno int16) {
	hb.bg0.Draw(float32(hb.pos[0])+sys.lifebarOffsetX, float32(hb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
	hb.bg1.Draw(float32(hb.pos[0])+sys.lifebarOffsetX, float32(hb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
	hb.bg2.Draw(float32(hb.pos[0])+sys.lifebarOffsetX, float32(hb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
}

func (hb *HealthBar) draw(layerno int16, ref int, hbr *HealthBar, f []*Fnt) {
	life := float32(sys.chars[ref][0].life) / float32(sys.chars[ref][0].lifeMax)
	redlife := float32(sys.chars[ref][0].redLife) / float32(sys.chars[ref][0].lifeMax)
	redval := sys.chars[ref][0].redLife - sys.chars[ref][0].life
	var MidPosX = (float32(sys.gameWidth-320) / 2)
	var MidPosY = (float32(sys.gameHeight-240) / 2)
	// Calculates the clipping rectangle based on current bar settings
	getBarClipRect := func(life float32) [4]int32 {
		r := sys.scrrect

		if hb.scalefill {
			life = 1
		}

		if hb.range_x != [2]int32{0, 0} {
			r[0], r[2] = calcBarFillRect(hb.pos[0], hb.range_x, sys.lifebarOffsetX, sys.lifebarScale, sys.widthScale, MidPosX, life)
		}

		if hb.range_y != [2]int32{0, 0} {
			r[1], r[3] = calcBarFillRect(hb.pos[1], hb.range_y, sys.lifebarOffsetY, sys.lifebarScale, sys.heightScale, MidPosY, life)
		}
		return r
	}

	if len(hb.mid.anim.frames) == 0 || life > hbr.midlife {
		life = hbr.midlife
	}

	// Draw the three rectangles: top, mid, and red
	lr, mr, rr := getBarClipRect(hbr.toplife), getBarClipRect(hbr.midlife), getBarClipRect(redlife)

	var (
		lxs, mxs, rxs float32 = 1.0, 1.0, 1.0
		lys, mys, rys float32 = 1.0, 1.0, 1.0
	)
	if hb.scalefill { // Scale the sprite's size instead of adjusting the rectangle
		v := [3]float32{hbr.toplife, hbr.midlife, redlife}
		if hb.range_y != [2]int32{0, 0} {
			lys, mys, rys = v[0], v[1], v[2]
		} else {
			lxs, mxs, rxs = v[0], v[1], v[2]
		}
	} else {
		if hb.range_y != [2]int32{0, 0} {
			if hb.range_y[0] < hb.range_y[1] {
				mr[1] += lr[3]
			}
			mr[3] -= Min(mr[3], lr[3])
		} else {
			if hb.range_x[0] < hb.range_x[1] {
				mr[0] += lr[2]
				//rr[0] += lr[2]
			}
			mr[2] -= Min(mr[2], lr[2])
			//rr[2] -= Min(rr[2], lr[2])
		}
	}
	if sys.lifebar.redlifebar {
		var rv int32
		for k := range hb.red {
			if k > rv && redval >= k {
				rv = k
			}
		}
		hb.red[rv].lay.DrawAnim(&rr, float32(hb.pos[0])+sys.lifebarOffsetX, float32(hb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, rxs, rys,
			layerno, &hb.red[rv].anim, hb.red[rv].palfx)
	}

	hb.mid.lay.DrawAnim(&mr, float32(hb.pos[0])+sys.lifebarOffsetX, float32(hb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, mxs, mys,
		layerno, &hb.mid.anim, hb.mid.palfx)

	if hb.mid_shift {
		hb.shift.lay.DrawAnim(&mr, float32(hb.pos[0])+sys.lifebarOffsetX, float32(hb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, mxs, mys,
			layerno, &hb.shift.anim, hb.shift.palfx)
	}

	// Multiple front elements
	var fv float32
	for k := range hb.front {
		if k > fv && life >= k/100 {
			fv = k
		}
	}
	hb.front[fv].lay.DrawAnim(&lr, float32(hb.pos[0])+sys.lifebarOffsetX, float32(hb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, lxs, lys,
		layerno, &hb.front[fv].anim, hb.front[fv].palfx)

	hb.shift.lay.DrawAnim(&lr, float32(hb.pos[0])+sys.lifebarOffsetX, float32(hb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, lxs, lys,
		layerno, &hb.shift.anim, hb.shift.palfx)

	if hb.value.font[0] >= 0 && int(hb.value.font[0]) < len(f) && f[hb.value.font[0]] != nil {
		text := strings.Replace(hb.value.text, "%d", fmt.Sprintf("%v", sys.chars[ref][0].life), 1)
		text = strings.Replace(text, "%p", fmt.Sprintf("%v", math.Round(float64(life)*100)), 1)
		hb.value.lay.DrawText(float32(hb.pos[0])+sys.lifebarOffsetX, float32(hb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale,
			layerno, text, f[hb.value.font[0]], hb.value.font[1], hb.value.font[2], hb.value.palfx, hb.value.frgba)
	}

	hb.top.Draw(float32(hb.pos[0])+sys.lifebarOffsetX, float32(hb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)

	if life <= float32(hb.warn_range[0])/100 && life >= float32(hb.warn_range[1])/100 {
		hb.warn.Draw(float32(hb.pos[0])+sys.lifebarOffsetX, float32(hb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
	}
}

type PowerBar struct {
	pos              [2]int32
	range_x          [2]int32
	range_y          [2]int32
	bg0              map[int32]*AnimLayout
	bg1              AnimLayout
	bg2              AnimLayout
	top              AnimLayout
	mid              AnimLayout
	front            map[int32]*AnimLayout
	shift            AnimLayout
	counter          map[int32]*LbText
	counter_rounding int32
	value            LbText
	value_rounding   int32
	level_snd        [9][2]int32
	midpower         float32
	midpowerMin      float32
	prevLevel        int32
	levelbars        bool
	scalefill        bool
}

func newPowerBar() *PowerBar {
	newBar := &PowerBar{
		front:            make(map[int32]*AnimLayout),
		bg0:              make(map[int32]*AnimLayout),
		counter:          make(map[int32]*LbText),
		counter_rounding: 1000,
		value_rounding:   1,
	}
	// Default power level sounds to -1,-1
	for i := range newBar.level_snd {
		newBar.level_snd[i] = [2]int32{-1, -1}
	}
	return newBar
}

func readPowerBar(pre string, is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt) *PowerBar {
	pb := newPowerBar()
	is.ReadI32(pre+"pos", &pb.pos[0], &pb.pos[1])
	is.ReadI32(pre+"range.x", &pb.range_x[0], &pb.range_x[1])
	is.ReadI32(pre+"range.y", &pb.range_y[0], &pb.range_y[1])
	pb.bg0[0] = ReadAnimLayout(pre+"bg0.", is, sff, at, 0)
	for k, v := range readMultipleValues(pre, "bg0", is, sff, at) {
		pb.bg0[k] = v
	}
	pb.bg1 = *ReadAnimLayout(pre+"bg1.", is, sff, at, 0)
	pb.bg2 = *ReadAnimLayout(pre+"bg2.", is, sff, at, 0)
	pb.top = *ReadAnimLayout(pre+"top.", is, sff, at, 0)
	pb.mid = *ReadAnimLayout(pre+"mid.", is, sff, at, 0)
	pb.front[0] = ReadAnimLayout(pre+"front.", is, sff, at, 0)
	for k, v := range readMultipleValues(pre, "front", is, sff, at) {
		pb.front[k] = v
	}
	// Lifebar power counter.
	pb.shift = *ReadAnimLayout(pre+"shift.", is, sff, at, 0)
	pb.counter[0] = readLbText(pre+"counter.", is, "%i", 0, f, 0)
	for k, v := range readMultipleLbText(pre, "counter", is, "%i", 0, f, 0) {
		pb.counter[k] = v
	}
	pb.value = *readLbText(pre+"value.", is, "", 0, f, 0)
	// Format options.
	is.ReadI32(pre+"counter.format.rounding", &pb.counter_rounding)
	is.ReadI32(pre+"value.format.power.rounding", &pb.value_rounding)
	// Avoid division by 0.
	if pb.counter_rounding < 1 {
		pb.counter_rounding = 1000
	}
	if pb.value_rounding < 1 {
		pb.value_rounding = 1
	}
	// Level sounds.
	for i := range pb.level_snd {
		if !is.ReadI32(fmt.Sprintf("%vlevel%v.snd", pre, i+1), &pb.level_snd[i][0], &pb.level_snd[i][1]) {
			is.ReadI32(fmt.Sprintf("level%v.snd", i+1), &pb.level_snd[i][0], &pb.level_snd[i][1])
		}
	}
	is.ReadBool(pre+"levelbars", &pb.levelbars)
	is.ReadBool(pre+"scalefill", &pb.scalefill)
	return pb
}

func (pb *PowerBar) step(ref int, pbr *PowerBar, snd *Snd) {
	pbval := sys.chars[ref][0].getPower()
	power := float32(pbval) / float32(sys.chars[ref][0].powerMax)
	level := pbval / 1000
	if pb.levelbars {
		power = float32(pbval)/1000 - MinF(float32(level), float32(sys.chars[ref][0].powerMax)/1000-1)
	}

	// Element shifting gradient
	pb.shift.anim.srcAlpha = int16(255 * (1 - power))
	pb.shift.anim.dstAlpha = int16(255 * power)

	pbr.midpower -= 1.0 / 144
	if power < pbr.midpowerMin {
		pbr.midpowerMin += (power - pbr.midpowerMin) * (1 / (12 - (power-pbr.midpowerMin)*144))
	} else {
		pbr.midpowerMin = power
	}
	if pbr.midpower < pbr.midpowerMin {
		pbr.midpower = pbr.midpowerMin
	}

	// Level sounds
	// TODO: These probably shouldn't play when the powerbar is invisible
	if level > pbr.prevLevel {
		i := int(level - 1)
		if i >= 0 && i < len(pb.level_snd) {
			snd.play(pb.level_snd[i], 100, 0, 0, 0, 0)
		}
		for i := range pb.counter {
			pb.counter[i].resetTxtPfx()
		}
	}
	pbr.prevLevel = level

	// Multiple front elements
	var fv1 int32
	for k := range pb.bg0 {
		if k > fv1 && pbval >= k {
			fv1 = k
		}
	}
	pb.bg0[fv1].Action()

	pb.bg1.Action()
	pb.bg2.Action()
	pb.top.Action()
	pb.mid.Action()
	pb.value.step()
	// Multiple front elements
	var fv2 int32
	for k := range pb.front {
		if k > fv2 && pbval >= k {
			fv2 = k
		}
	}
	pb.front[fv2].Action()

	pb.shift.Action()

	// Multiple counter fonts
	var cv int32
	for k := range pb.counter {
		if k > cv && pbval >= k {
			cv = k
		}
	}
	pb.counter[cv].step()
	pb.value.step()
}

func (pb *PowerBar) reset() {
	for i := range pb.bg0 {
		pb.bg0[i].Reset()
	}
	pb.bg1.Reset()
	pb.bg2.Reset()
	pb.top.Reset()
	pb.mid.Reset()
	for i := range pb.front {
		pb.front[i].Reset()
	}
	pb.shift.Reset()
	pb.shift.anim.srcAlpha = 0
	pb.shift.anim.dstAlpha = 255
}

func (pb *PowerBar) bgDraw(layerno int16, ref int) {
	pbval := sys.chars[ref][0].getPower()
	var fv int32
	for k := range pb.bg0 {
		if k > fv && pbval >= k {
			fv = k
		}
	}
	pb.bg0[fv].Draw(float32(pb.pos[0])+sys.lifebarOffsetX, float32(pb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
	pb.bg1.Draw(float32(pb.pos[0])+sys.lifebarOffsetX, float32(pb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
	pb.bg2.Draw(float32(pb.pos[0])+sys.lifebarOffsetX, float32(pb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
}

func (pb *PowerBar) draw(layerno int16, ref int, pbr *PowerBar, f []*Fnt) {
	pbval := sys.chars[ref][0].getPower()
	power := float32(pbval) / float32(sys.chars[ref][0].powerMax)
	level := pbval / 1000

	if pb.levelbars {
		power = float32(pbval)/1000 - MinF(float32(level), float32(sys.chars[ref][0].powerMax)/1000-1)
	}

	var MidPosX = (float32(sys.gameWidth-320) / 2)
	var MidPosY = (float32(sys.gameHeight-240) / 2)
	getBarClipRect := func(power float32) [4]int32 {
		r := sys.scrrect

		if pb.scalefill {
			power = 1
		}

		if pb.range_x != [2]int32{0, 0} {
			r[0], r[2] = calcBarFillRect(pb.pos[0], pb.range_x, sys.lifebarOffsetX, sys.lifebarScale, sys.widthScale, MidPosX, power)
		}

		if pb.range_y != [2]int32{0, 0} {
			r[1], r[3] = calcBarFillRect(pb.pos[1], pb.range_y, sys.lifebarOffsetY, sys.lifebarScale, sys.heightScale, MidPosY, power)
		}
		return r
	}
	pr, mr := getBarClipRect(power), getBarClipRect(pbr.midpower)

	var (
		pxs, mxs float32 = 1.0, 1.0
		pys, mys float32 = 1.0, 1.0
	)
	if pb.scalefill {
		v := [3]float32{power, pbr.midpower}
		if pb.range_y != [2]int32{0, 0} {
			pys, mys = v[0], v[1]
		} else {
			pxs, mxs = v[0], v[1]
		}
	} else {
		if pb.range_y != [2]int32{0, 0} {
			if pb.range_y[0] < pb.range_y[1] {
				mr[1] += pr[3]
			}
			mr[3] -= Min(mr[3], pr[3])
		} else {
			if pb.range_x[0] < pb.range_x[1] {
				mr[0] += pr[2]
			}
			mr[2] -= Min(mr[2], pr[2])
		}
	}
	pb.mid.lay.DrawAnim(&mr, float32(pb.pos[0])+sys.lifebarOffsetX, float32(pb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, mxs, mys,
		layerno, &pb.mid.anim, pb.mid.palfx)

	// Multiple front elements
	var fv int32
	for k := range pb.front {
		if k > fv && pbval >= k {
			fv = k
		}
	}
	pb.front[fv].lay.DrawAnim(&pr, float32(pb.pos[0])+sys.lifebarOffsetX, float32(pb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, pxs, pys,
		layerno, &pb.front[fv].anim, pb.front[fv].palfx)

	pb.shift.lay.DrawAnim(&pr, float32(pb.pos[0])+sys.lifebarOffsetX, float32(pb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, pxs, pys,
		layerno, &pb.shift.anim, pb.shift.palfx)

	// Powerbar text.
	if pb.counter[0].font[0] >= 0 && int(pb.counter[0].font[0]) < len(f) && f[pb.counter[0].font[0]] != nil {
		// Multiple counter fonts according to powerbar level
		var cv int32
		for k := range pb.counter {
			if k > cv && pbval >= k {
				cv = k
			}
		}

		pb.counter[cv].lay.DrawText(
			float32(pb.pos[0])+sys.lifebarOffsetX,
			float32(pb.pos[1])+sys.lifebarOffsetY,
			sys.lifebarScale,
			layerno,
			strings.Replace(pb.counter[cv].text, "%i", fmt.Sprintf("%v", pbval/pb.counter_rounding), 1),
			f[pb.counter[cv].font[0]],
			pb.counter[cv].font[1],
			pb.counter[cv].font[2],
			pb.counter[cv].palfx,
			pb.counter[cv].frgba,
		)
	}

	// Per-level powerbar text.
	if pb.value.font[0] >= 0 && int(pb.value.font[0]) < len(f) && f[pb.value.font[0]] != nil {
		text := strings.Replace(pb.value.text, "%d", fmt.Sprintf("%v", pbval/pb.value_rounding), 1)
		text = strings.Replace(text, "%p", fmt.Sprintf("%v", math.Round(float64(power)*100)), 1)

		pb.value.lay.DrawText(
			float32(pb.pos[0])+sys.lifebarOffsetX,
			float32(pb.pos[1])+sys.lifebarOffsetY,
			sys.lifebarScale,
			layerno,
			text,
			f[pb.value.font[0]],
			pb.value.font[1],
			pb.value.font[2],
			pb.value.palfx,
			pb.value.frgba,
		)
	}
	pb.top.Draw(float32(pb.pos[0])+sys.lifebarOffsetX, float32(pb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
}

type GuardBar struct {
	pos         [2]int32
	range_x     [2]int32
	range_y     [2]int32
	bg0         AnimLayout
	bg1         AnimLayout
	bg2         AnimLayout
	top         AnimLayout
	mid         AnimLayout
	warn        AnimLayout
	warn_range  [2]int32
	value       LbText
	front       map[float32]*AnimLayout
	shift       AnimLayout
	midpower    float32
	midpowerMin float32
	invertfill  bool
	scalefill   bool
}

func newGuardBar() (gb *GuardBar) {
	gb = &GuardBar{front: make(map[float32]*AnimLayout)}
	return
}

func readGuardBar(pre string, is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt) *GuardBar {
	gb := newGuardBar()
	is.ReadI32(pre+"pos", &gb.pos[0], &gb.pos[1])
	is.ReadI32(pre+"range.x", &gb.range_x[0], &gb.range_x[1])
	is.ReadI32(pre+"range.y", &gb.range_y[0], &gb.range_y[1])
	gb.bg0 = *ReadAnimLayout(pre+"bg0.", is, sff, at, 0)
	gb.bg1 = *ReadAnimLayout(pre+"bg1.", is, sff, at, 0)
	gb.bg2 = *ReadAnimLayout(pre+"bg2.", is, sff, at, 0)
	gb.top = *ReadAnimLayout(pre+"top.", is, sff, at, 0)
	gb.mid = *ReadAnimLayout(pre+"mid.", is, sff, at, 0)
	gb.front[0] = ReadAnimLayout(pre+"front.", is, sff, at, 0)
	for k, v := range readMultipleValuesF(pre, "front", is, sff, at) {
		gb.front[k] = v
	}
	gb.shift = *ReadAnimLayout(pre+"shift.", is, sff, at, 0)
	gb.value = *readLbText(pre+"value.", is, "%d", 0, f, 0)
	is.ReadI32(pre+"warn.range", &gb.warn_range[0], &gb.warn_range[1])
	gb.warn = *ReadAnimLayout(pre+"warn.", is, sff, at, 0)
	is.ReadBool(pre+"invertfill", &gb.invertfill)
	is.ReadBool(pre+"scalefill", &gb.scalefill)
	return gb
}

func (gb *GuardBar) step(ref int, gbr *GuardBar, snd *Snd) {
	if !sys.lifebar.guardbar {
		return
	}

	points := float32(sys.chars[ref][0].guardPoints) / float32(sys.chars[ref][0].guardPointsMax)
	if gb.invertfill {
		points = 1 - points
	}

	// Element shifting gradient
	gb.shift.anim.srcAlpha = int16(255 * (1 - points))
	gb.shift.anim.dstAlpha = int16(255 * points)

	gbr.midpower -= 1.0 / 144
	if points < gbr.midpowerMin {
		gbr.midpowerMin += (points - gbr.midpowerMin) * (1 / (12 - (points-gbr.midpowerMin)*144))
	} else {
		gbr.midpowerMin = points
	}
	if gbr.midpower < gbr.midpowerMin {
		gbr.midpower = gbr.midpowerMin
	}
	gb.bg0.Action()
	gb.bg1.Action()
	gb.bg2.Action()
	gb.top.Action()
	gb.mid.Action()

	// Multiple front elements
	var mv float32
	for k := range gb.front {
		if k > mv && points >= k/100 {
			mv = k
		}
	}
	gb.front[mv].Action()

	gb.shift.Action()
	gb.warn.Action()
}

func (gb *GuardBar) reset() {
	gb.bg0.Reset()
	gb.bg1.Reset()
	gb.bg2.Reset()
	gb.top.Reset()
	gb.mid.Reset()
	for _, v := range gb.front {
		v.Reset()
	}
	gb.shift.Reset()
	gb.shift.anim.srcAlpha = 0
	gb.shift.anim.dstAlpha = 255
	gb.warn.Reset()
}

func (gb *GuardBar) bgDraw(layerno int16) {
	if !sys.lifebar.guardbar {
		return
	}
	gb.bg0.Draw(float32(gb.pos[0])+sys.lifebarOffsetX, float32(gb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
	gb.bg1.Draw(float32(gb.pos[0])+sys.lifebarOffsetX, float32(gb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
	gb.bg2.Draw(float32(gb.pos[0])+sys.lifebarOffsetX, float32(gb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
}

func (gb *GuardBar) draw(layerno int16, ref int, gbr *GuardBar, f []*Fnt) {
	if !sys.lifebar.guardbar {
		return
	}

	points := float32(sys.chars[ref][0].guardPoints) / float32(sys.chars[ref][0].guardPointsMax)
	if gb.invertfill {
		points = 1 - points
	}

	var MidPosX = (float32(sys.gameWidth-320) / 2)
	var MidPosY = (float32(sys.gameHeight-240) / 2)
	getBarClipRect := func(points float32) [4]int32 {
		r := sys.scrrect

		if gb.scalefill {
			points = 1
		}

		if gb.range_x != [2]int32{0, 0} {
			r[0], r[2] = calcBarFillRect(gb.pos[0], gb.range_x, sys.lifebarOffsetX, sys.lifebarScale, sys.widthScale, MidPosX, points)
		}

		if gb.range_y != [2]int32{0, 0} {
			r[1], r[3] = calcBarFillRect(gb.pos[1], gb.range_y, sys.lifebarOffsetY, sys.lifebarScale, sys.heightScale, MidPosY, points)
		}
		return r
	}

	pr, mr := getBarClipRect(points), getBarClipRect(gbr.midpower)

	var (
		pxs, mxs float32 = 1.0, 1.0
		pys, mys float32 = 1.0, 1.0
	)
	if gb.scalefill {
		v := [3]float32{points, gbr.midpower}
		if gb.range_y != [2]int32{0, 0} {
			pys, mys = v[0], v[1]
		} else {
			pxs, mxs = v[0], v[1]
		}
	} else {
		if gb.range_y != [2]int32{0, 0} {
			if gb.range_y[0] < gb.range_y[1] {
				mr[1] += pr[3]
			}
			mr[3] -= Min(mr[3], pr[3])
		} else {
			if gb.range_x[0] < gb.range_x[1] {
				mr[0] += pr[2]
			}
			mr[2] -= Min(mr[2], pr[2])
		}
	}
	gb.mid.lay.DrawAnim(&mr, float32(gb.pos[0])+sys.lifebarOffsetX, float32(gb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, mxs, mys,
		layerno, &gb.mid.anim, gb.mid.palfx)

	// Multiple front elements
	var mv float32
	for k := range gb.front {
		if k > mv && points >= k/100 {
			mv = k
		}
	}
	gb.front[mv].lay.DrawAnim(&pr, float32(gb.pos[0])+sys.lifebarOffsetX, float32(gb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, pxs, pys,
		layerno, &gb.front[mv].anim, gb.front[mv].palfx)

	gb.shift.lay.DrawAnim(&pr, float32(gb.pos[0])+sys.lifebarOffsetX, float32(gb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, pxs, pys,
		layerno, &gb.shift.anim, gb.shift.palfx)

	if gb.value.font[0] >= 0 && int(gb.value.font[0]) < len(f) && f[gb.value.font[0]] != nil {
		text := strings.Replace(gb.value.text, "%d", fmt.Sprintf("%v", sys.chars[ref][0].guardPoints), 1)
		text = strings.Replace(text, "%p", fmt.Sprintf("%v", math.Round(float64(points)*100)), 1)
		gb.value.lay.DrawText(float32(gb.pos[0])+sys.lifebarOffsetX, float32(gb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale,
			layerno, text, f[gb.value.font[0]], gb.value.font[1], gb.value.font[2], gb.value.palfx, gb.value.frgba)
	}

	if points <= float32(gb.warn_range[0])/100 && points >= float32(gb.warn_range[1])/100 {
		gb.warn.Draw(float32(gb.pos[0])+sys.lifebarOffsetX, float32(gb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
	}

	gb.top.Draw(float32(gb.pos[0])+sys.lifebarOffsetX, float32(gb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
}

type StunBar struct {
	pos         [2]int32
	range_x     [2]int32
	range_y     [2]int32
	bg0         AnimLayout
	bg1         AnimLayout
	bg2         AnimLayout
	top         AnimLayout
	mid         AnimLayout
	warn_range  [2]int32
	warn        AnimLayout
	value       LbText
	front       map[float32]*AnimLayout
	shift       AnimLayout
	midpower    float32
	midpowerMin float32
	invertfill  bool
	scalefill   bool
}

func newStunBar() (sb *StunBar) {
	sb = &StunBar{front: make(map[float32]*AnimLayout)}
	return
}

func readStunBar(pre string, is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt) *StunBar {
	sb := newStunBar()
	is.ReadI32(pre+"pos", &sb.pos[0], &sb.pos[1])
	is.ReadI32(pre+"range.x", &sb.range_x[0], &sb.range_x[1])
	is.ReadI32(pre+"range.y", &sb.range_y[0], &sb.range_y[1])
	sb.bg0 = *ReadAnimLayout(pre+"bg0.", is, sff, at, 0)
	sb.bg1 = *ReadAnimLayout(pre+"bg1.", is, sff, at, 0)
	sb.bg2 = *ReadAnimLayout(pre+"bg2.", is, sff, at, 0)
	sb.top = *ReadAnimLayout(pre+"top.", is, sff, at, 0)
	sb.mid = *ReadAnimLayout(pre+"mid.", is, sff, at, 0)
	sb.front[0] = ReadAnimLayout(pre+"front.", is, sff, at, 0)
	for k, v := range readMultipleValuesF(pre, "front", is, sff, at) {
		sb.front[k] = v
	}
	sb.shift = *ReadAnimLayout(pre+"shift.", is, sff, at, 0)
	sb.value = *readLbText(pre+"value.", is, "%d", 0, f, 0)
	is.ReadI32(pre+"warn.range", &sb.warn_range[0], &sb.warn_range[1])
	sb.warn = *ReadAnimLayout(pre+"warn.", is, sff, at, 0)
	is.ReadBool(pre+"invertfill", &sb.invertfill)
	is.ReadBool(pre+"scalefill", &sb.scalefill)
	return sb
}

func (sb *StunBar) step(ref int, sbr *StunBar, snd *Snd) {
	if !sys.lifebar.stunbar {
		return
	}

	points := float32(sys.chars[ref][0].dizzyPoints) / float32(sys.chars[ref][0].dizzyPointsMax)
	if sb.invertfill {
		points = 1 - points
	}

	// Element shifting gradient
	sb.shift.anim.srcAlpha = int16(255 * (1 - points))
	sb.shift.anim.dstAlpha = int16(255 * points)

	sbr.midpower -= 1.0 / 144
	if points < sbr.midpowerMin {
		sbr.midpowerMin += (points - sbr.midpowerMin) * (1 / (12 - (points-sbr.midpowerMin)*144))
	} else {
		sbr.midpowerMin = points
	}
	if sbr.midpower < sbr.midpowerMin {
		sbr.midpower = sbr.midpowerMin
	}
	sb.bg0.Action()
	sb.bg1.Action()
	sb.bg2.Action()
	sb.top.Action()
	sb.mid.Action()
	sb.value.step()
	// Multiple front elements
	var mv float32
	for k := range sb.front {
		if k > mv && points >= k/100 {
			mv = k
		}
	}
	sb.front[mv].Action()

	sb.shift.Action()
	sb.warn.Action()
}

func (sb *StunBar) reset() {
	sb.bg0.Reset()
	sb.bg1.Reset()
	sb.bg2.Reset()
	sb.top.Reset()
	sb.mid.Reset()
	for i := range sb.front {
		sb.front[i].Reset()
	}
	sb.shift.Reset()
	sb.shift.anim.srcAlpha = 255
	sb.shift.anim.dstAlpha = 0
	sb.warn.Reset()
}

func (sb *StunBar) bgDraw(layerno int16) {
	if !sys.lifebar.stunbar {
		return
	}
	sb.bg0.Draw(float32(sb.pos[0])+sys.lifebarOffsetX, float32(sb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
	sb.bg1.Draw(float32(sb.pos[0])+sys.lifebarOffsetX, float32(sb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
	sb.bg2.Draw(float32(sb.pos[0])+sys.lifebarOffsetX, float32(sb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
}

func (sb *StunBar) draw(layerno int16, ref int, sbr *StunBar, f []*Fnt) {
	if !sys.lifebar.stunbar {
		return
	}

	points := float32(sys.chars[ref][0].dizzyPoints) / float32(sys.chars[ref][0].dizzyPointsMax)
	if sb.invertfill {
		points = 1 - points
	}

	var MidPosX = (float32(sys.gameWidth-320) / 2)
	var MidPosY = (float32(sys.gameHeight-240) / 2)
	getBarClipRect := func(points float32) [4]int32 {
		r := sys.scrrect

		if sb.scalefill {
			points = 1
		}

		if sb.range_x != [2]int32{0, 0} {
			r[0], r[2] = calcBarFillRect(sb.pos[0], sb.range_x, sys.lifebarOffsetX, sys.lifebarScale, sys.widthScale, MidPosX, points)
		}

		if sb.range_y != [2]int32{0, 0} {
			r[1], r[3] = calcBarFillRect(sb.pos[1], sb.range_y, sys.lifebarOffsetY, sys.lifebarScale, sys.heightScale, MidPosY, points)
		}
		return r
	}

	pr, mr := getBarClipRect(points), getBarClipRect(sbr.midpower)

	var (
		pxs, mxs float32 = 1.0, 1.0
		pys, mys float32 = 1.0, 1.0
	)
	if sb.scalefill {
		v := [3]float32{points, sbr.midpower}
		if sb.range_y != [2]int32{0, 0} {
			pys, mys = v[0], v[1]
		} else {
			pxs, mxs = v[0], v[1]
		}
	} else {
		if sb.range_y != [2]int32{0, 0} {
			if sb.range_y[0] < sb.range_y[1] {
				mr[1] += pr[3]
			}
			mr[3] -= Min(mr[3], pr[3])
		} else {
			if sb.range_x[0] < sb.range_x[1] {
				mr[0] += pr[2]
			}
			mr[2] -= Min(mr[2], pr[2])
		}
	}
	sb.mid.lay.DrawAnim(&mr, float32(sb.pos[0])+sys.lifebarOffsetX, float32(sb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, mxs, mys,
		layerno, &sb.mid.anim, sb.mid.palfx)

	// Multiple front elements
	var mv float32
	for k := range sb.front {
		if k > mv && points >= k/100 {
			mv = k
		}
	}
	sb.front[mv].lay.DrawAnim(&pr, float32(sb.pos[0])+sys.lifebarOffsetX, float32(sb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, pxs, pys,
		layerno, &sb.front[mv].anim, sb.front[mv].palfx)

	sb.shift.lay.DrawAnim(&pr, float32(sb.pos[0])+sys.lifebarOffsetX, float32(sb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale, pxs, pys,
		layerno, &sb.shift.anim, sb.shift.palfx)

	if sb.value.font[0] >= 0 && int(sb.value.font[0]) < len(f) && f[sb.value.font[0]] != nil {
		text := strings.Replace(sb.value.text, "%d", fmt.Sprintf("%v", sys.chars[ref][0].dizzyPoints), 1)
		text = strings.Replace(text, "%p", fmt.Sprintf("%v", math.Round(float64(points)*100)), 1)
		sb.value.lay.DrawText(float32(sb.pos[0])+sys.lifebarOffsetX, float32(sb.pos[1])+sys.lifebarOffsetY, sys.lifebarScale,
			layerno, text, f[sb.value.font[0]], sb.value.font[1], sb.value.font[2], sb.value.palfx, sb.value.frgba)
	}

	if points >= float32(sb.warn_range[0])/100 && points <= float32(sb.warn_range[1])/100 {
		sb.warn.Draw(float32(sb.pos[0])+sys.lifebarOffsetX, float32(sb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
	}

	sb.top.Draw(float32(sb.pos[0])+sys.lifebarOffsetX, float32(sb.pos[1])+sys.lifebarOffsetY, layerno, sys.lifebarScale)
}

type LifeBarFace struct {
	pos               [2]int32
	bg                AnimLayout
	bg0               AnimLayout
	bg1               AnimLayout
	bg2               AnimLayout
	top               AnimLayout
	ko                AnimLayout
	face_spr          [2]int32
	face              *Sprite
	face_lay          Layout
	palshare          bool
	palfxshare        bool
	teammate_pos      [2]int32
	teammate_spacing  [2]int32
	teammate_bg       AnimLayout
	teammate_bg0      AnimLayout
	teammate_bg1      AnimLayout
	teammate_bg2      AnimLayout
	teammate_top      AnimLayout
	teammate_ko       AnimLayout
	teammate_face_spr [2]int32
	teammate_face     []*Sprite
	teammate_face_lay Layout
	teammate_scale    []float32
	teammate_ko_hide  bool
	numko             int32
	old_spr           [2]int32
	old_pal           [2]int32
}

func newLifeBarFace() *LifeBarFace {
	return &LifeBarFace{
		face_spr:          [2]int32{-1},
		teammate_face_spr: [2]int32{-1},
		palshare:          true,
	}
}

func readLifeBarFace(pre string, is IniSection, sff *Sff, at AnimationTable) *LifeBarFace {
	fa := newLifeBarFace()
	is.ReadI32(pre+"pos", &fa.pos[0], &fa.pos[1])
	fa.bg = *ReadAnimLayout(pre+"bg.", is, sff, at, 0)
	fa.bg0 = *ReadAnimLayout(pre+"bg0.", is, sff, at, 0)
	fa.bg1 = *ReadAnimLayout(pre+"bg1.", is, sff, at, 0)
	fa.bg2 = *ReadAnimLayout(pre+"bg2.", is, sff, at, 0)
	fa.top = *ReadAnimLayout(pre+"top.", is, sff, at, 0)
	fa.ko = *ReadAnimLayout(pre+"ko.", is, sff, at, 0)
	is.ReadI32(pre+"face.spr", &fa.face_spr[0], &fa.face_spr[1])
	fa.face_lay = *ReadLayout(pre+"face.", is, 0)
	is.ReadBool(pre+"face.palshare", &fa.palshare)
	is.ReadBool(pre+"face.palfxshare", &fa.palfxshare)
	is.ReadI32(pre+"teammate.pos", &fa.teammate_pos[0], &fa.teammate_pos[1])
	is.ReadI32(pre+"teammate.spacing", &fa.teammate_spacing[0],
		&fa.teammate_spacing[1])
	fa.teammate_bg = *ReadAnimLayout(pre+"teammate.bg.", is, sff, at, 0)
	fa.teammate_bg0 = *ReadAnimLayout(pre+"teammate.bg0.", is, sff, at, 0)
	fa.teammate_bg1 = *ReadAnimLayout(pre+"teammate.bg1.", is, sff, at, 0)
	fa.teammate_bg2 = *ReadAnimLayout(pre+"teammate.bg2.", is, sff, at, 0)
	fa.teammate_top = *ReadAnimLayout(pre+"teammate.top.", is, sff, at, 0)
	fa.teammate_ko = *ReadAnimLayout(pre+"teammate.ko.", is, sff, at, 0)
	is.ReadI32(pre+"teammate.face.spr", &fa.teammate_face_spr[0],
		&fa.teammate_face_spr[1])
	if fa.teammate_face_spr[0] != -1 {
		sys.sel.charSpritePreload[[...]int16{int16(fa.teammate_face_spr[0]), int16(fa.teammate_face_spr[1])}] = true
	}
	fa.teammate_face_lay = *ReadLayout(pre+"teammate.face.", is, 0)
	is.ReadBool(pre+"teammate.ko.hide", &fa.teammate_ko_hide)
	return fa
}

func (fa *LifeBarFace) step(ref int, far *LifeBarFace) {
	group, number := int16(fa.face_spr[0]), int16(fa.face_spr[1])
	if sys.chars[ref][0] != nil && sys.chars[ref][0].anim != nil {
		if mg, ok := sys.chars[ref][0].anim.remap[group]; ok {
			if mn, ok := mg[number]; ok {
				group, number = mn[0], mn[1]
			}
		}
	}
	if far.old_spr[0] != int32(group) || far.old_spr[1] != int32(number) ||
		far.old_pal[0] != sys.cgi[ref].remappedpal[0] || far.old_pal[1] != sys.cgi[ref].remappedpal[1] {
		far.face = sys.cgi[ref].sff.getOwnPalSprite(group, number, &sys.cgi[ref].palettedata.palList)
		far.old_spr = [...]int32{int32(group), int32(number)}
		far.old_pal = [...]int32{sys.cgi[ref].remappedpal[0], sys.cgi[ref].remappedpal[1]}
	}
	fa.bg.Action()
	fa.bg0.Action()
	fa.bg1.Action()
	fa.bg2.Action()
	fa.top.Action()
	fa.ko.Action()
	fa.teammate_bg.Action()
	fa.teammate_bg0.Action()
	fa.teammate_bg1.Action()
	fa.teammate_bg2.Action()
	fa.teammate_top.Action()
	fa.teammate_ko.Action()
}

func (fa *LifeBarFace) reset() {
	fa.bg.Reset()
	fa.bg0.Reset()
	fa.bg1.Reset()
	fa.bg2.Reset()
	fa.top.Reset()
	fa.ko.Reset()
	fa.teammate_bg.Reset()
	fa.teammate_bg0.Reset()
	fa.teammate_bg1.Reset()
	fa.teammate_bg2.Reset()
	fa.teammate_top.Reset()
	fa.teammate_ko.Reset()
	if !sys.roundResetFlg {
		fa.old_spr = [2]int32{}
		fa.old_pal = [2]int32{}
	}
}

func (fa *LifeBarFace) bgDraw(layerno int16) {
	fa.bg.Draw(float32(fa.pos[0])+sys.lifebarOffsetX, float32(fa.pos[1]), layerno, sys.lifebarScale)
	fa.bg0.Draw(float32(fa.pos[0])+sys.lifebarOffsetX, float32(fa.pos[1]), layerno, sys.lifebarScale)
	fa.bg1.Draw(float32(fa.pos[0])+sys.lifebarOffsetX, float32(fa.pos[1]), layerno, sys.lifebarScale)
	fa.bg2.Draw(float32(fa.pos[0])+sys.lifebarOffsetX, float32(fa.pos[1]), layerno, sys.lifebarScale)
}

func (fa *LifeBarFace) draw(layerno int16, ref int, far *LifeBarFace) {
	if far.face != nil {
		// Get player current PalFX if applicable
		pfx := newPalFX()
		if far.palfxshare {
			pfx = sys.chars[ref][0].getPalfx()
		}

		// Swap palette maps to get the player's current palette
		if far.palshare {
			sys.cgi[ref].palettedata.palList.SwapPalMap(&sys.chars[ref][0].getPalfx().remap)
		}

		// Get texture
		far.face.Pal = nil
		if far.face.PalTex != nil {
			far.face.PalTex = far.face.GetPalTex(&sys.cgi[ref].palettedata.palList)
		} else {
			far.face.Pal = far.face.GetPal(&sys.cgi[ref].palettedata.palList)
		}

		// Revert palette maps to initial state
		if far.palshare {
			sys.cgi[ref].palettedata.palList.SwapPalMap(&sys.chars[ref][0].getPalfx().remap)
		}

		// TODO: PalFX sharing has a bug in Tag in that it uses the parameter from the char's original placement in the team
		// For instance if player 3 tags in, they will use p3 palette options instead of p1
		// https://github.com/ikemen-engine/Ikemen-GO/issues/2269

		// Reset system brightness if player initiated SuperPause (cancel "darken" parameter)
		ob := sys.brightness
		if ref == sys.superplayerno {
			sys.brightness = 256
		}

		// Draw the actual face sprite
		fa.face_lay.DrawFaceSprite((float32(fa.pos[0])+sys.lifebarOffsetX)*sys.lifebarScale, float32(fa.pos[1])*sys.lifebarScale, layerno,
			far.face, pfx, sys.cgi[ref].portraitscale*sys.lifebarPortraitScale, &fa.face_lay.window)

		// Draw KO layer
		if !sys.chars[ref][0].alive() {
			fa.ko.Draw(float32(fa.pos[0])+sys.lifebarOffsetX, float32(fa.pos[1]), layerno, sys.lifebarScale)
		}

		// Restore original system brightness
		sys.brightness = ob

		// Turns mode teammates
		i := int32(len(far.teammate_face)) - 1
		x := float32(fa.teammate_pos[0] + fa.teammate_spacing[0]*(i-1))
		y := float32(fa.teammate_pos[1] + fa.teammate_spacing[1]*(i-1))

		if fa.teammate_ko_hide {
			x -= float32(fa.teammate_spacing[0] * fa.numko)
			y -= float32(fa.teammate_spacing[1] * fa.numko)
		}

		for ; i >= 0; i-- {
			if i != fa.numko {
				// Skip in case of KO hiding
				if i < fa.numko && fa.teammate_ko_hide == true {
					x -= float32(fa.teammate_spacing[0])
					y -= float32(fa.teammate_spacing[1])
					continue
				}
				// Draw background
				fa.teammate_bg.Draw((x + sys.lifebarOffsetX), y, layerno, sys.lifebarScale)
				fa.teammate_bg0.Draw((x + sys.lifebarOffsetX), y, layerno, sys.lifebarScale)
				fa.teammate_bg1.Draw((x + sys.lifebarOffsetX), y, layerno, sys.lifebarScale)
				fa.teammate_bg2.Draw((x + sys.lifebarOffsetX), y, layerno, sys.lifebarScale)
				// Draw face
				fa.teammate_face_lay.DrawFaceSprite((x+sys.lifebarOffsetX)*sys.lifebarScale, y*sys.lifebarScale, layerno, far.teammate_face[i], nil,
					far.teammate_scale[i]*sys.lifebarPortraitScale, &fa.teammate_face_lay.window)
				// Draw KO layer
				if i < fa.numko {
					fa.teammate_ko.Draw((x + sys.lifebarOffsetX), y, layerno, sys.lifebarScale)
				}
				// Add spacing
				x -= float32(fa.teammate_spacing[0])
				y -= float32(fa.teammate_spacing[1])
			}
		}
	}

	// Draw top layer
	fa.top.Draw(float32(fa.pos[0])+sys.lifebarOffsetX, float32(fa.pos[1]), layerno, sys.lifebarScale)
}

type LifeBarName struct {
	pos              [2]int32
	name             LbText
	bg               AnimLayout
	top              AnimLayout
	teammate_pos     [2]int32
	teammate_spacing [2]int32
	teammate_name    LbText
	teammate_bg      AnimLayout
	numko            int32
}

func newLifeBarName() *LifeBarName {
	return &LifeBarName{}
}

func readLifeBarName(pre string, is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt) *LifeBarName {
	nm := newLifeBarName()
	is.ReadI32(pre+"pos", &nm.pos[0], &nm.pos[1])
	nm.name = *readLbText(pre+"name.", is, "", 0, f, 0)
	nm.bg = *ReadAnimLayout(pre+"bg.", is, sff, at, 0)
	is.ReadI32(pre+"teammate.pos", &nm.teammate_pos[0], &nm.teammate_pos[1])
	is.ReadI32(pre+"teammate.spacing", &nm.teammate_spacing[0],
		&nm.teammate_spacing[1])
	nm.teammate_name = *readLbText(pre+"teammate.name.", is, "", 0, f, 0)
	nm.teammate_bg = *ReadAnimLayout(pre+"teammate.bg.", is, sff, at, 0)
	nm.top = *ReadAnimLayout(pre+"top.", is, sff, at, 0)
	return nm
}

func (nm *LifeBarName) step() {
	nm.bg.Action()
	nm.teammate_bg.Action()
	nm.top.Action()
	nm.name.step()
	nm.teammate_name.step()
}

func (nm *LifeBarName) reset() {
	nm.bg.Reset()
	nm.teammate_bg.Reset()
	nm.top.Reset()
}

func (nm *LifeBarName) bgDraw(layerno int16) {
	nm.bg.Draw(float32(nm.pos[0])+sys.lifebarOffsetX, float32(nm.pos[1]), layerno, sys.lifebarScale)
}

func (nm *LifeBarName) draw(layerno int16, ref int, f []*Fnt, side int) {
	if nm.name.font[0] >= 0 && int(nm.name.font[0]) < len(f) && f[nm.name.font[0]] != nil {
		nm.name.lay.DrawText((float32(nm.pos[0]) + sys.lifebarOffsetX), float32(nm.pos[1]), sys.lifebarScale, layerno,
			sys.cgi[ref].lifebarname, f[nm.name.font[0]], nm.name.font[1], nm.name.font[2], nm.name.palfx, nm.name.frgba)
	}
	// Get Turns mode partner names from system
	if sys.tmode[side] == TM_Turns {
		i := int32(len(sys.sel.selected[side])) - 1
		x := float32(nm.teammate_pos[0] + nm.teammate_spacing[0]*(i-nm.numko-1))
		y := float32(nm.teammate_pos[1] + nm.teammate_spacing[1]*(i-nm.numko-1))
		for ; i >= nm.numko+1; i-- {
			nm.teammate_bg.Draw((x + sys.lifebarOffsetX), y, layerno, sys.lifebarScale)
			if nm.teammate_name.font[0] >= 0 && int(nm.teammate_name.font[0]) < len(f) && f[nm.teammate_name.font[0]] != nil {
				nm.teammate_name.lay.DrawText((float32(x) + sys.lifebarOffsetX), float32(y), sys.lifebarScale, layerno,
					sys.sel.GetChar(sys.sel.selected[side][i][0]).lifebarname, f[nm.teammate_name.font[0]], nm.teammate_name.font[1],
					nm.teammate_name.font[2], nm.teammate_name.palfx, nm.teammate_name.frgba)
			}
			x -= float32(nm.teammate_spacing[0])
			y -= float32(nm.teammate_spacing[1])
		}
	}
	nm.top.Draw(float32(nm.pos[0])+sys.lifebarOffsetX, float32(nm.pos[1]), layerno, sys.lifebarScale)
}

type LifeBarWinIcon struct {
	pos           [2]int32
	iconoffset    [2]int32
	useiconupto   int32
	counter       LbText
	bg0           AnimLayout
	top           AnimLayout
	icon          [WT_NumTypes]AnimLayout
	wins          []WinType
	numWins       int
	added, addedP *Animation
}

func newLifeBarWinIcon() *LifeBarWinIcon {
	return &LifeBarWinIcon{useiconupto: 4}
}

func readLifeBarWinIcon(pre string, is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt) *LifeBarWinIcon {
	wi := newLifeBarWinIcon()
	is.ReadI32(pre+"pos", &wi.pos[0], &wi.pos[1])
	is.ReadI32(pre+"iconoffset", &wi.iconoffset[0], &wi.iconoffset[1])
	is.ReadI32("useiconupto", &wi.useiconupto)
	wi.counter = *readLbText(pre+"counter.", is, "%i", 0, f, 0)
	wi.bg0 = *ReadAnimLayout(pre+"bg0.", is, sff, at, 0)
	wi.top = *ReadAnimLayout(pre+"top.", is, sff, at, 0)
	wi.icon[WT_Normal] = *ReadAnimLayout(pre+"n.", is, sff, at, 0)
	wi.icon[WT_Special] = *ReadAnimLayout(pre+"s.", is, sff, at, 0)
	wi.icon[WT_Hyper] = *ReadAnimLayout(pre+"h.", is, sff, at, 0)
	wi.icon[WT_Cheese] = *ReadAnimLayout(pre+"c.", is, sff, at, 0)
	wi.icon[WT_Time] = *ReadAnimLayout(pre+"t.", is, sff, at, 0)
	wi.icon[WT_Throw] = *ReadAnimLayout(pre+"throw.", is, sff, at, 0)
	wi.icon[WT_Suicide] = *ReadAnimLayout(pre+"suicide.", is, sff, at, 0)
	wi.icon[WT_Teammate] = *ReadAnimLayout(pre+"teammate.", is, sff, at, 0)
	wi.icon[WT_Perfect] = *ReadAnimLayout(pre+"perfect.", is, sff, at, 0)
	return wi
}

func (wi *LifeBarWinIcon) add(wt WinType) {
	wi.wins = append(wi.wins, wt)
	if wt >= WT_PNormal {
		wi.addedP = &Animation{}
		*wi.addedP = wi.icon[WT_Perfect].anim
		wi.addedP.Reset()
		wt -= WT_PNormal
	}
	wi.added = &Animation{}
	*wi.added = wi.icon[wt].anim
	wi.added.Reset()
}

func (wi *LifeBarWinIcon) step(numwin int32) {
	wi.bg0.Action()
	wi.top.Action()
	if int(numwin) < len(wi.wins) {
		wi.wins = wi.wins[:numwin]
		wi.reset()
	}
	for i := range wi.icon {
		wi.icon[i].Action()
	}
	if wi.added != nil {
		wi.added.Action()
	}
	if wi.addedP != nil {
		wi.addedP.Action()
	}
}

func (wi *LifeBarWinIcon) reset() {
	wi.bg0.Reset()
	wi.top.Reset()
	for i := range wi.icon {
		wi.icon[i].Reset()
	}
	wi.numWins = len(wi.wins)
	wi.added, wi.addedP = nil, nil
}

func (wi *LifeBarWinIcon) clear() {
	wi.wins = nil
}

func (wi *LifeBarWinIcon) draw(layerno int16, f []*Fnt, side int) {
	bg0num := float64(sys.lifebar.ro.match_wins[^side&1])
	if sys.tmode[^side&1] == TM_Turns {
		bg0num = float64(sys.numTurns[^side&1])
	}
	for i := 0; i < int(math.Min(float64(wi.useiconupto), bg0num)); i++ {
		wi.bg0.Draw(float32(wi.pos[0]+wi.iconoffset[0]*int32(i))+sys.lifebarOffsetX,
			float32(wi.pos[1]+wi.iconoffset[1]*int32(i)), layerno, sys.lifebarScale)
	}
	if len(wi.wins) > int(wi.useiconupto) {
		if wi.counter.font[0] >= 0 && int(wi.counter.font[0]) < len(f) && f[wi.counter.font[0]] != nil {
			wi.counter.lay.DrawText(float32(wi.pos[0])+sys.lifebarOffsetX, float32(wi.pos[1]), sys.lifebarScale,
				layerno, strings.Replace(wi.counter.text, "%i", fmt.Sprintf("%v", len(wi.wins)), 1),
				f[wi.counter.font[0]], wi.counter.font[1], wi.counter.font[2], wi.counter.palfx, wi.counter.frgba)
		}
	} else {
		i := 0
		for ; i < wi.numWins; i++ {
			wt, p := wi.wins[i], false
			if wt >= WT_PNormal {
				wt -= WT_PNormal
				p = true
			}
			wi.icon[wt].Draw(float32(wi.pos[0]+wi.iconoffset[0]*int32(i))+sys.lifebarOffsetX,
				float32(wi.pos[1]+wi.iconoffset[1]*int32(i)), layerno, sys.lifebarScale)
			if p {
				wi.icon[WT_Perfect].Draw(float32(wi.pos[0]+wi.iconoffset[0]*int32(i))+sys.lifebarOffsetX,
					float32(wi.pos[1]+wi.iconoffset[1]*int32(i)), layerno, sys.lifebarScale)
			}
		}
		if wi.added != nil {
			wt, p := wi.wins[i], false
			if wi.addedP != nil {
				wt -= WT_PNormal
				p = true
			}
			wi.icon[wt].lay.DrawAnim(&wi.icon[wt].lay.window,
				float32(wi.pos[0]+wi.iconoffset[0]*int32(i))+sys.lifebarOffsetX,
				float32(wi.pos[1]+wi.iconoffset[1]*int32(i)), sys.lifebarScale, 1, 1, layerno, wi.added, nil)
			if p {
				wi.icon[WT_Perfect].lay.DrawAnim(&wi.icon[WT_Perfect].lay.window,
					float32(wi.pos[0]+wi.iconoffset[0]*int32(i))+sys.lifebarOffsetX,
					float32(wi.pos[1]+wi.iconoffset[1]*int32(i)), sys.lifebarScale, 1, 1, layerno, wi.addedP, nil)
			}
		}
	}
	for i := 0; i < int(math.Min(float64(wi.useiconupto), bg0num)); i++ {
		wi.top.Draw(float32(wi.pos[0]+wi.iconoffset[0]*int32(i))+sys.lifebarOffsetX,
			float32(wi.pos[1]+wi.iconoffset[1]*int32(i)), layerno, sys.lifebarScale)
	}
}

type LifeBarTime struct {
	pos            [2]int32
	counter        map[int32]*LbText
	bg             AnimLayout
	top            AnimLayout
	framespercount int32
	activeIdx      int32
}

func newLifeBarTime() *LifeBarTime {
	return &LifeBarTime{
		counter:        make(map[int32]*LbText),
		framespercount: 60,
	}
}

func readLifeBarTime(is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt) *LifeBarTime {
	ti := newLifeBarTime()
	is.ReadI32("pos", &ti.pos[0], &ti.pos[1])
	ti.counter[0] = readLbText("counter.", is, "", 0, f, 0)
	for k, v := range readMultipleLbText("", "counter", is, "", 0, f, 0) {
		ti.counter[k] = v
	}
	ti.bg = *ReadAnimLayout("bg.", is, sff, at, 0)
	ti.top = *ReadAnimLayout("top.", is, sff, at, 0)
	is.ReadI32("framespercount", &ti.framespercount)
	return ti
}

func (ti *LifeBarTime) step() {
	ti.bg.Action()
	ti.top.Action()
	ti.counter[ti.activeIdx].step()
}

func (ti *LifeBarTime) reset() {
	ti.bg.Reset()
	ti.top.Reset()
}

func (ti *LifeBarTime) bgDraw(layerno int16) {
	ti.bg.Draw(float32(ti.pos[0])+sys.lifebarOffsetX, float32(ti.pos[1]), layerno, sys.lifebarScale)
}

func (ti *LifeBarTime) draw(layerno int16, f []*Fnt) {
	if ti.framespercount > 0 &&
		ti.counter[0].font[0] >= 0 && int(ti.counter[0].font[0]) < len(f) && f[ti.counter[0].font[0]] != nil {
		var timeval int32 = -1
		time := "o"
		if sys.time >= 0 {
			timeval = int32(math.Ceil(float64(sys.time) / float64(ti.framespercount)))
			time = fmt.Sprintf("%v", timeval)
		}
		// Multiple fonts according to time remaining
		var tv int32
		if timeval < 0 { // Infinite time. Select the highest group
			for k := range ti.counter {
				if k > tv {
					tv = k
				}
			}
		} else {
			for k := range ti.counter {
				if k > tv && timeval >= k {
					tv = k
				}
			}
		}
		ti.activeIdx = tv
		ti.counter[tv].lay.DrawText(float32(ti.pos[0])+sys.lifebarOffsetX, float32(ti.pos[1]), sys.lifebarScale, layerno,
			time, f[ti.counter[tv].font[0]], ti.counter[tv].font[1], ti.counter[tv].font[2], ti.counter[tv].palfx,
			ti.counter[tv].frgba)
	}
	ti.top.Draw(float32(ti.pos[0])+sys.lifebarOffsetX, float32(ti.pos[1]), layerno, sys.lifebarScale)
}

type LifeBarCombo struct {
	pos            [2]int32
	start_x        float32
	counter        map[int32]*LbText
	counter_shake  bool
	counter_time   int32
	counter_mult   float32
	text           map[int32]*LbText
	bg             AnimLayout
	top            AnimLayout
	displaytime    int32
	showspeed      float32
	hidespeed      float32
	separator      string
	places         int32
	curhit, oldhit int32
	curdmg, olddmg int32
	curpct, oldpct float32
	resttime       int32
	counterX       float32
	shaketime      int32
	combo          int32
	tracker        int32
	autoalign      bool
}

func newLifeBarCombo() *LifeBarCombo {
	return &LifeBarCombo{
		displaytime:  90,
		showspeed:    8,
		hidespeed:    4,
		counter:      make(map[int32]*LbText),
		counter_time: 7,
		counter_mult: 1.0 / 20,
		text:         make(map[int32]*LbText),
		autoalign:    true,
	}
}

func readLifeBarCombo(pre string, is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt, side int) *LifeBarCombo {
	co := newLifeBarCombo()
	is.ReadI32(pre+"pos", &co.pos[0], &co.pos[1])
	is.ReadF32(pre+"start.x", &co.start_x)
	if side == 1 {
		// mugen 1.0 implementation reuses winmugen code where both sides shared the same values
		if pre == "team2." {
			co.start_x = float32(sys.lifebarLocalcoord[0]) - co.start_x
		} else {
			co.pos[0] = sys.lifebarLocalcoord[0] - co.pos[0]
		}
	}
	var align int32
	if pre == "" {
		if side == 0 {
			align = 1
		} else {
			align = -1
		}
	}
	co.counter[0] = readLbText(pre+"counter.", is, "%i", 2, f, align)
	for k, v := range readMultipleLbText(pre, "counter", is, "%i", 2, f, align) {
		co.counter[k] = v
	}
	is.ReadBool(pre+"counter.shake", &co.counter_shake)
	is.ReadI32(pre+"counter.time", &co.counter_time)
	is.ReadF32(pre+"counter.mult", &co.counter_mult)
	co.text[0] = readLbText(pre+"text.", is, "", 2, f, align)
	for k, v := range readMultipleLbText(pre, "text", is, "", 2, f, align) {
		co.text[k] = v
	}
	co.bg = *ReadAnimLayout(pre+"bg0.", is, sff, at, 2)
	co.top = *ReadAnimLayout(pre+"top.", is, sff, at, 2)
	is.ReadI32(pre+"displaytime", &co.displaytime)
	is.ReadF32(pre+"showspeed", &co.showspeed)
	co.showspeed = MaxF(1, co.showspeed)
	is.ReadF32(pre+"hidespeed", &co.hidespeed)
	co.separator, _, _ = is.getText("format.decimal.separator")
	is.ReadI32("format.decimal.places", &co.places)
	is.ReadBool(pre+"autoalign", &co.autoalign)
	return co
}

func (co *LifeBarCombo) step(combo, damage int32, percentage float32, dizzy bool) {
	co.bg.Action()
	co.top.Action()

	// Update team combo tracker
	if combo > 0 {
		combo = co.combo
	} else {
		co.combo = 0
	}

	if co.resttime > 0 {
		co.counterX -= co.counterX / co.showspeed
	} else if combo < 2 {
		co.counterX -= sys.lifebar.fnt_scale * co.hidespeed * float32(sys.lifebarLocalcoord[0]) / 320
		if co.counterX < co.start_x*2 {
			co.counterX = co.start_x * 2
		}
	}

	if co.shaketime > 0 {
		co.shaketime--
	}

	if AbsF(co.counterX) < 1 && !dizzy {
		co.resttime--
	}

	// Update if number of hits or total damage change
	if combo >= 2 {
		if co.oldhit != combo || co.olddmg != damage {
			for i := range co.counter {
				co.counter[i].resetTxtPfx()
			}
			for i := range co.text {
				co.text[i].resetTxtPfx()
			}
			co.curhit = combo
			co.curdmg = damage
			co.curpct = percentage
			co.resttime = co.displaytime
			co.tracker = co.combo
			if co.counter_shake && co.oldhit != combo {
				co.shaketime = co.counter_time
			}
		}
	}
	co.oldhit = combo
	co.olddmg = damage
	co.oldpct = percentage

	// Multiple counter fonts
	var cv int32
	for k := range co.counter {
		if k > cv && co.tracker >= k {
			cv = k
		}
	}

	// Multiple text fonts
	var tv int32
	for k := range co.text {
		if k > tv && co.tracker >= k {
			tv = k
		}
	}

	if co.counterX != co.start_x*2 {
		co.counter[cv].step()
		co.text[tv].step()
	}
}

func (co *LifeBarCombo) reset() {
	co.bg.Reset()
	co.top.Reset()
	co.curhit, co.oldhit = 0, 0
	co.curdmg, co.olddmg = 0, 0
	co.curpct, co.oldpct = 0, 0
	co.resttime = 0
	co.combo = 0
	co.counterX = co.start_x * 2
	co.shaketime = 0
}

func (co *LifeBarCombo) draw(layerno int16, f []*Fnt, side int) {
	if co.resttime <= 0 && co.counterX == co.start_x*2 {
		return
	}

	// Multiple counter fonts according to combo hits
	var cv int32
	for k := range co.counter {
		if k > cv && co.tracker >= k {
			cv = k
		}
	}

	// Multiple text fonts according to combo hits
	var tv int32
	for k := range co.text {
		if k > tv && co.tracker >= k {
			tv = k
		}
	}

	counter := strings.Replace(co.counter[cv].text, "%i", fmt.Sprintf("%v", co.curhit), 1)
	x := float32(co.pos[0])
	if side == 0 {
		if co.start_x <= 0 {
			x += co.counterX
		}
		if co.counter[cv].font[0] >= 0 && int(co.counter[cv].font[0]) < len(f) && f[co.counter[cv].font[0]] != nil && co.autoalign {
			x += float32(f[co.counter[cv].font[0]].TextWidth(counter, co.counter[cv].font[1])) *
				co.counter[cv].lay.scale[0] * sys.lifebar.fnt_scale
		}
	} else {
		if co.start_x <= 0 {
			x -= co.counterX
		}
	}
	co.bg.Draw(x+sys.lifebarOffsetX, float32(co.pos[1]), layerno, sys.lifebarScale)
	var length float32
	if co.text[tv].font[0] >= 0 && int(co.text[tv].font[0]) < len(f) && f[co.text[tv].font[0]] != nil {
		text := strings.Replace(co.text[tv].text, "%i", fmt.Sprintf("%v", co.curhit), 1)
		text = strings.Replace(text, "%d", fmt.Sprintf("%v", co.curdmg), 1)
		// Truncate the percentage to avoid rounding to 100% unless the enemy is defeated
		truncatedPct := math.Floor(float64(co.curpct)*math.Pow10(int(co.places))) / math.Pow10(int(co.places))
		// Split float value
		s := strings.Split(fmt.Sprintf("%.[2]*[1]f", truncatedPct, co.places), ".")
		// Decimal separator
		if co.places > 0 {
			if len(s) > 1 {
				s[0] = s[0] + co.separator + s[1]
			}
		}
		// Replace %p with formatted string
		text = strings.Replace(text, "%p", s[0], 1)
		// Split on new line
		for k, v := range strings.Split(text, "\\n") {
			if side == 1 && co.autoalign {
				if lt := float32(f[co.text[tv].font[0]].TextWidth(v, co.text[tv].font[1])) * co.text[tv].lay.scale[0] * sys.lifebar.fnt_scale; lt > length {
					length = lt
				}
			}
			co.text[tv].lay.DrawText(x+sys.lifebarOffsetX, float32(co.pos[1])+
				float32(k)*(float32(f[co.text[tv].font[0]].Size[1])*co.text[tv].lay.scale[1]*sys.lifebar.fnt_scale+
					float32(f[co.text[tv].font[0]].Spacing[1])*co.text[tv].lay.scale[1]*sys.lifebar.fnt_scale),
				sys.lifebarScale, layerno, v, f[co.text[tv].font[0]], co.text[tv].font[1], co.text[tv].font[2],
				co.text[tv].palfx, co.text[tv].frgba)
		}
	}
	if co.counter[cv].font[0] >= 0 && int(co.counter[cv].font[0]) < len(f) && f[co.counter[cv].font[0]] != nil {
		if side == 0 && co.autoalign {
			length = float32(f[co.counter[cv].font[0]].TextWidth(counter, co.counter[cv].font[1])) * co.counter[cv].lay.scale[0] * sys.lifebar.fnt_scale
		}

		z := 1 + float32(co.shaketime)*co.counter_mult*float32(math.Sin(float64(co.shaketime)*(math.Pi/2.5)))
		co.counter[cv].lay.DrawText((x-length+sys.lifebarOffsetX)/z, float32(co.pos[1])/z, z*sys.lifebarScale, layerno,
			counter, f[co.counter[cv].font[0]], co.counter[cv].font[1], co.counter[cv].font[2], co.counter[cv].palfx, co.counter[cv].frgba)
	}
	co.top.Draw(x+sys.lifebarOffsetX, float32(co.pos[1]), layerno, sys.lifebarScale)
}

type LbMsg struct {
	resttime int32
	agetimer int32
	counterX float32
	text     string
	bg       AnimLayout
	front    AnimLayout
	del      bool
}

func newLbMsg(text string, time int32, side int) *LbMsg {
	return &LbMsg{
		resttime: time,
		counterX: sys.lifebar.ac[side].start_x * 2,
		text:     text,
	}
}

func insertLbMsg(array []*LbMsg, value *LbMsg, index int) []*LbMsg {
	// Remove one empty index before appending the new one
	// This reduces the chance of an empty space pushing older messages
	if index == 0 {
		// Check from the top
		for i := 0; i < len(array); i++ {
			if array[i].del {
				// Remove the element at index i by shifting elements down
				copy(array[i:], array[i+1:])
				array = array[:len(array)-1]
				break
			}
		}
	} else {
		// Check from the bottom
		for i := len(array) - 1; i >= index; i-- {
			if array[i].del {
				// Remove the element at index i by shifting elements down
				copy(array[i:], array[i+1:])
				array = array[:len(array)-1]
				break
			}
		}
	}
	// Insert the new message at the specified index
	if index == len(array) {
		array = append(array, value)
	} else {
		array = append(array[:index], append([]*LbMsg{value}, array[index:]...)...)
	}
	return array
}

func removeLbMsg(array []*LbMsg, index int) []*LbMsg {
	return append(array[:index], array[index+1:]...)
}

type LifeBarAction struct {
	oldleader   int
	pos         [2]int32
	spacing     [2]int32
	start_x     float32
	text        LbText
	displaytime int32
	showspeed   float32
	hidespeed   float32
	messages    []*LbMsg
	is          IniSection
	max         int32
}

func newLifeBarAction() *LifeBarAction {
	return &LifeBarAction{
		displaytime: 90,
		showspeed:   8,
		hidespeed:   4,
		max:         8,
		is:          make(map[string]string),
	}
}

func readLifeBarAction(pre string, is IniSection, f []*Fnt) *LifeBarAction {
	ac := newLifeBarAction()
	ac.is = is
	is.ReadI32(pre+"pos", &ac.pos[0], &ac.pos[1])
	is.ReadI32(pre+"spacing", &ac.spacing[0], &ac.spacing[1])
	is.ReadF32(pre+"start.x", &ac.start_x)
	if pre == "team2." {
		ac.start_x = float32(sys.lifebarLocalcoord[0]) - ac.start_x
	}
	ac.text = *readLbText(pre+"text.", is, "", 2, f, 0)
	is.ReadI32(pre+"displaytime", &ac.displaytime)
	is.ReadF32(pre+"showspeed", &ac.showspeed)
	ac.showspeed = MaxF(1, ac.showspeed)
	is.ReadF32(pre+"hidespeed", &ac.hidespeed)
	is.ReadI32(pre+"max", &ac.max)
	return ac
}

func (ac *LifeBarAction) step(leader int) {
	if ac.oldleader != leader {
		ac.oldleader = leader
	}
	for _, v := range ac.messages {
		v.bg.Action()
		v.front.Action()
		if v.resttime > 0 {
			v.counterX -= v.counterX / ac.showspeed
		} else {
			v.counterX -= sys.lifebar.fnt_scale * ac.hidespeed * float32(sys.lifebarLocalcoord[0]) / 320
			if v.counterX < ac.start_x*2 {
				v.del = true
			}
		}
		if AbsF(v.counterX) < 1 {
			v.resttime--
		}
		v.agetimer++
	}
	if len(ac.messages) > 0 && ac.messages[len(ac.messages)-1].del {
		ac.messages = removeLbMsg(ac.messages, len(ac.messages)-1)
	}
	ac.text.step()
}

func (ac *LifeBarAction) reset(leader int) {
	ac.oldleader = leader
	ac.messages = []*LbMsg{}
}

func (ac *LifeBarAction) draw(layerno int16, f []*Fnt, side int) {
	for k, v := range ac.messages {
		if v.resttime <= 0 && v.counterX == ac.start_x*2 {
			continue
		}
		x := float32(ac.pos[0])
		if side == 0 {
			if ac.start_x <= 0 {
				x += v.counterX
			}
		} else {
			if ac.start_x <= 0 {
				x -= v.counterX
			}
		}
		// Previously the spacing accounted for the size of the current element, like sprite tilespacing
		// That created a lot of trouble when aligning things, so currently the spacing is fixed, like typical font and animation spacing
		// https://github.com/ikemen-engine/Ikemen-GO/issues/2166
		// Draw background
		if v.bg.anim.spr != nil {
			v.bg.Draw(x+sys.lifebarOffsetX+float32(k)*float32(ac.spacing[0]),
				float32(ac.pos[1])+float32(k)*float32(ac.spacing[1]),
				layerno, sys.lifebarScale)
		}
		// Draw animation/sprite
		if v.front.anim.spr != nil {
			v.front.Draw(x+sys.lifebarOffsetX+float32(k)*float32(ac.spacing[0]),
				float32(ac.pos[1])+float32(k)*float32(ac.spacing[1]),
				layerno, sys.lifebarScale)
		}
		// Draw text
		if v.text != "" && ac.text.font[0] >= 0 && int(ac.text.font[0]) < len(f) && f[ac.text.font[0]] != nil {
			ac.text.lay.DrawText(x+sys.lifebarOffsetX+float32(k)*float32(ac.spacing[0]),
				float32(ac.pos[1])+float32(k)*float32(ac.spacing[1]),
				sys.lifebarScale, layerno, v.text, f[ac.text.font[0]], ac.text.font[1], ac.text.font[2],
				ac.text.palfx, ac.text.frgba)
		}
	}
}

type LifeBarRound struct {
	snd                 *Snd
	pos                 [2]int32
	match_wins          [2]int32
	match_maxdrawgames  [2]int32
	start_waittime      int32
	round_time          int32
	round_sndtime       int32
	round               [9]AnimTextSnd
	round_default       AnimTextSnd
	round_default_top   AnimLayout
	round_default_bg    [32]AnimLayout
	round_single        AnimTextSnd
	round_single_top    AnimLayout
	round_single_bg     [32]AnimLayout
	round_final         AnimTextSnd
	round_final_top     AnimLayout
	round_final_bg      [32]AnimLayout
	fight_time          int32
	fight_sndtime       int32
	fight               AnimTextSnd
	fight_top           AnimLayout
	fight_bg            [32]AnimLayout
	ctrl_time           int32
	ko_time             int32
	ko_sndtime          int32
	ko, dko, to         AnimTextSnd
	ko_top              AnimLayout
	ko_bg               [32]AnimLayout
	dko_top             AnimLayout
	dko_bg              [32]AnimLayout
	to_top              AnimLayout
	to_bg               [32]AnimLayout
	slow_time           int32
	slow_fadetime       int32
	slow_speed          float32
	over_waittime       int32
	over_hittime        int32
	over_wintime        int32
	over_time           int32
	win_time            int32
	win_sndtime         int32
	win, win2           [2]AnimTextSnd
	win_top, win2_top   [2]AnimLayout
	win_bg, win2_bg     [2][32]AnimLayout
	win3, win4          [2]AnimTextSnd
	win3_top, win4_top  [2]AnimLayout
	win3_bg, win4_bg    [2][32]AnimLayout
	drawgame            AnimTextSnd
	drawgame_top        AnimLayout
	drawgame_bg         [32]AnimLayout
	current             int32
	waitTimer           [4]int32 // 0 round call; 1 fight call; 2 KO screen; 3 winner announcement
	waitSoundTimer      [4]int32
	drawTimer           [4]int32
	roundCallOver       bool
	fightCallOver       bool
	timerActive         bool
	winType             [WT_NumTypes * 2]LbBgTextSnd
	fadein_time         int32
	fadein_col          uint32
	fadeout_time        int32
	fadeout_col         uint32
	shutter_time        int32
	shutter_col         uint32
	callfight_time      int32
	triggerRoundDisplay bool // FightScreenState trigger
	triggerFightDisplay bool
	triggerKODisplay    bool
	triggerWinDisplay   bool
}

func newLifeBarRound(snd *Snd) *LifeBarRound {
	return &LifeBarRound{
		snd:                snd,
		match_wins:         [...]int32{2, 2},
		match_maxdrawgames: [...]int32{1, 1},
		start_waittime:     30,
		ctrl_time:          30,
		slow_time:          60,
		slow_fadetime:      45,
		slow_speed:         0.25,
		over_waittime:      45,
		over_hittime:       10,
		over_wintime:       45,
		over_time:          210,
		fadein_time:        30,
		fadeout_time:       30,
		shutter_time:       15,
		callfight_time:     60,
	}
}

func readLifeBarRound(is IniSection,
	sff *Sff, at AnimationTable, snd *Snd, f []*Fnt) *LifeBarRound {
	ro := newLifeBarRound(snd)
	var tmp int32
	var ftmp float32
	is.ReadI32("pos", &ro.pos[0], &ro.pos[1])
	is.ReadI32("match.wins", &ro.match_wins[0], &ro.match_wins[1])
	is.ReadI32("match.maxdrawgames", &ro.match_maxdrawgames[0], &ro.match_maxdrawgames[1])
	if is.ReadI32("start.waittime", &tmp) {
		ro.start_waittime = Max(1, tmp)
	}
	is.ReadI32("round.time", &ro.round_time)
	ro.round_sndtime = ro.round_time
	is.ReadI32("round.sndtime", &ro.round_sndtime)
	for i := range ro.round {
		ro.round[i] = *ReadAnimTextSnd(fmt.Sprintf("round%v.", i+1), is, sff, at, 2, f)
	}
	ro.round_default = *ReadAnimTextSnd("round.default.", is, sff, at, 2, f)
	ro.round_default_top = *ReadAnimLayout("round.default.top.", is, sff, at, 2)
	for i := range ro.round_default_bg {
		ro.round_default_bg[i] = *ReadAnimLayout(fmt.Sprintf("round.default.bg%v.", i), is, sff, at, 2)
	}
	// Single round animations and sounds
	ro.round_single = *ReadAnimTextSnd("round.single.", is, sff, at, 2, f)
	ro.round_single_top = *ReadAnimLayout("round.single.top.", is, sff, at, 2)
	for i := range ro.round_single_bg {
		ro.round_single_bg[i] = *ReadAnimLayout(fmt.Sprintf("round.single.bg%v.", i), is, sff, at, 2)
	}
	// Final round animations and sounds
	ro.round_final = *ReadAnimTextSnd("round.final.", is, sff, at, 2, f)
	ro.round_final_top = *ReadAnimLayout("round.final.top.", is, sff, at, 2)
	for i := range ro.round_final_bg {
		ro.round_final_bg[i] = *ReadAnimLayout(fmt.Sprintf("round.final.bg%v.", i), is, sff, at, 2)
	}
	is.ReadI32("fight.time", &ro.fight_time)
	ro.fight_sndtime = ro.fight_time
	is.ReadI32("fight.sndtime", &ro.fight_sndtime)
	ro.fight = *ReadAnimTextSnd("fight.", is, sff, at, 2, f)
	ro.fight_top = *ReadAnimLayout("fight.top.", is, sff, at, 2)
	for i := range ro.fight_bg {
		ro.fight_bg[i] = *ReadAnimLayout(fmt.Sprintf("fight.bg%v.", i), is, sff, at, 2)
	}
	if is.ReadI32("ctrl.time", &tmp) {
		ro.ctrl_time = Max(1, tmp)
	}
	is.ReadI32("ko.time", &ro.ko_time)
	ro.ko_sndtime = ro.ko_time
	is.ReadI32("ko.sndtime", &ro.ko_sndtime)
	ro.ko = *ReadAnimTextSnd("ko.", is, sff, at, 1, f)
	ro.ko_top = *ReadAnimLayout("ko.top.", is, sff, at, 1)
	for i := range ro.ko_bg {
		ro.ko_bg[i] = *ReadAnimLayout(fmt.Sprintf("ko.bg%v.", i), is, sff, at, 2)
	}
	ro.dko = *ReadAnimTextSnd("dko.", is, sff, at, 1, f)
	ro.dko_top = *ReadAnimLayout("dko.top.", is, sff, at, 1)
	for i := range ro.dko_bg {
		ro.dko_bg[i] = *ReadAnimLayout(fmt.Sprintf("dko.bg%v.", i), is, sff, at, 2)
	}
	ro.to = *ReadAnimTextSnd("to.", is, sff, at, 1, f)
	ro.to_top = *ReadAnimLayout("to.top.", is, sff, at, 1)
	for i := range ro.to_bg {
		ro.to_bg[i] = *ReadAnimLayout(fmt.Sprintf("to.bg%v.", i), is, sff, at, 2)
	}
	is.ReadI32("slow.time", &ro.slow_time)
	if is.ReadI32("slow.fadetime", &tmp) {
		ro.slow_fadetime = Min(ro.slow_time, tmp)
	} else {
		ro.slow_fadetime = int32(float32(ro.slow_time) * 0.75)
	}
	if is.ReadF32("slow.speed", &ftmp) {
		ro.slow_speed = MinF(1, ftmp)
	}
	if is.ReadI32("over.hittime", &tmp) {
		ro.over_hittime = Max(1, tmp)
	}
	if is.ReadI32("over.waittime", &tmp) {
		ro.over_waittime = Max(1, tmp)
	}
	if is.ReadI32("over.wintime", &tmp) {
		ro.over_wintime = Max(1, tmp)
	}
	if is.ReadI32("over.time", &tmp) {
		ro.over_time = Max(1, tmp)
	}
	is.ReadI32("win.time", &ro.win_time)
	ro.win_sndtime = ro.win_time
	is.ReadI32("win.sndtime", &ro.win_sndtime)
	for i := 0; i < 2; i++ {
		var ok, bg bool
		// win
		if _, ok = is[fmt.Sprintf("p%v.win.text", i+1)]; !ok {
			if _, ok = is[fmt.Sprintf("p%v.win.spr", i+1)]; !ok {
				_, ok = is[fmt.Sprintf("p%v.win.anim", i+1)]
			}
		}
		if ok {
			ro.win[i] = *ReadAnimTextSnd(fmt.Sprintf("p%v.win.", i+1), is, sff, at, 1, f)
		} else {
			ro.win[i] = *ReadAnimTextSnd("win.", is, sff, at, 1, f)
		}
		if _, ok = is[fmt.Sprintf("p%v.win.top.anim", i+1)]; !ok {
			_, ok = is[fmt.Sprintf("p%v.win.top.spr", i+1)]
		}
		if ok {
			ro.win_top[i] = *ReadAnimLayout(fmt.Sprintf("p%v.win.top.", i+1), is, sff, at, 1)
		} else {
			ro.win_top[i] = *ReadAnimLayout("win.top.", is, sff, at, 1)
		}
		for j := range ro.win_bg[i] {
			if _, ok = is[fmt.Sprintf("p%v.win.bg%v.anim", i+1, j)]; !ok {
				_, ok = is[fmt.Sprintf("p%v.win.bg%v.spr", i+1, j)]
			}
			if ok {
				ro.win_bg[i][j] = *ReadAnimLayout(fmt.Sprintf("p%v.win.bg%v.", i+1, j), is, sff, at, 1)
			} else {
				ro.win_bg[i][j] = *ReadAnimLayout(fmt.Sprintf("win.bg%v.", j), is, sff, at, 1)
			}
		}
		// win2
		if _, ok = is[fmt.Sprintf("p%v.win2.text", i+1)]; !ok {
			if _, ok = is[fmt.Sprintf("p%v.win2.spr", i+1)]; !ok {
				_, ok = is[fmt.Sprintf("p%v.win2.anim", i+1)]
			}
		}
		if ok {
			ro.win2[i] = *ReadAnimTextSnd(fmt.Sprintf("p%v.win2.", i+1), is, sff, at, 1, f)
		} else {
			ro.win2[i] = *ReadAnimTextSnd("win2.", is, sff, at, 1, f)
		}
		if _, ok = is[fmt.Sprintf("p%v.win2.top.anim", i+1)]; !ok {
			_, ok = is[fmt.Sprintf("p%v.win2.top.spr", i+1)]
		}
		if ok {
			ro.win2_top[i] = *ReadAnimLayout(fmt.Sprintf("p%v.win2.top.", i+1), is, sff, at, 1)
		} else {
			ro.win2_top[i] = *ReadAnimLayout("win2.top.", is, sff, at, 1)
		}
		for j := range ro.win2_bg[i] {
			if _, ok = is[fmt.Sprintf("p%v.win2.bg%v.anim", i+1, j)]; !ok {
				_, ok = is[fmt.Sprintf("p%v.win2.bg%v.spr", i+1, j)]
			}
			if ok {
				ro.win2_bg[i][j] = *ReadAnimLayout(fmt.Sprintf("p%v.win2.bg%v.", i+1, j), is, sff, at, 1)
			} else {
				ro.win2_bg[i][j] = *ReadAnimLayout(fmt.Sprintf("win2.bg%v.", j), is, sff, at, 1)
			}
		}
		// win3
		if _, ok = is[fmt.Sprintf("p%v.win3.text", i+1)]; !ok {
			if _, ok = is[fmt.Sprintf("p%v.win3.spr", i+1)]; !ok {
				_, ok = is[fmt.Sprintf("p%v.win3.anim", i+1)]
			}
		}
		if ok {
			ro.win3[i] = *ReadAnimTextSnd(fmt.Sprintf("p%v.win3.", i+1), is, sff, at, 1, f)
		} else {
			if _, ok = is["win3.text"]; !ok {
				if _, ok = is["win3.spr"]; !ok {
					_, ok = is["win3.anim"]
				}
			}
			if ok {
				ro.win3[i] = *ReadAnimTextSnd("win3.", is, sff, at, 1, f)
			} else {
				ro.win3[i] = ro.win2[i]
			}
		}
		if _, ok = is[fmt.Sprintf("p%v.win3.top.anim", i+1)]; !ok {
			_, ok = is[fmt.Sprintf("p%v.win3.top.spr", i+1)]
		}
		if ok {
			ro.win3_top[i] = *ReadAnimLayout(fmt.Sprintf("p%v.win3.top.", i+1), is, sff, at, 1)
		} else {
			if _, ok = is["win3.top.anim"]; !ok {
				_, ok = is["win3.top.spr"]
			}
			if ok {
				ro.win3_top[i] = *ReadAnimLayout("win3.top.", is, sff, at, 1)
			} else {
				ro.win3_top[i] = ro.win2_top[i]
			}
		}
		for j := range ro.win3_bg[i] {
			if _, ok = is[fmt.Sprintf("p%v.win3.bg%v.anim", i+1, j)]; !ok {
				_, ok = is[fmt.Sprintf("p%v.win3.bg%v.spr", i+1, j)]
			}
			if ok {
				ro.win3_bg[i][j] = *ReadAnimLayout(fmt.Sprintf("p%v.win3.bg%v.", i+1, j), is, sff, at, 1)
				bg = true
			} else {
				if _, ok = is[fmt.Sprintf("win3.bg%v.anim", j)]; !ok {
					_, ok = is[fmt.Sprintf("win3.bg%v.spr", j)]
				}
				if ok {
					ro.win3_bg[i][j] = *ReadAnimLayout(fmt.Sprintf("win3.bg%v.", j), is, sff, at, 1)
					bg = true
				}
			}
		}
		if !bg {
			ro.win3_bg[i] = ro.win2_bg[i]
		}
		bg = false
		// win4
		if _, ok = is[fmt.Sprintf("p%v.win4.text", i+1)]; !ok {
			if _, ok = is[fmt.Sprintf("p%v.win4.spr", i+1)]; !ok {
				_, ok = is[fmt.Sprintf("p%v.win4.anim", i+1)]
			}
		}
		if ok {
			ro.win4[i] = *ReadAnimTextSnd(fmt.Sprintf("p%v.win4.", i+1), is, sff, at, 1, f)
		} else {
			if _, ok = is["win4.text"]; !ok {
				if _, ok = is["win4.spr"]; !ok {
					_, ok = is["win4.anim"]
				}
			}
			if ok {
				ro.win4[i] = *ReadAnimTextSnd("win4.", is, sff, at, 1, f)
			} else {
				ro.win4[i] = ro.win2[i]
			}
		}
		if _, ok = is[fmt.Sprintf("p%v.win4.top.anim", i+1)]; !ok {
			_, ok = is[fmt.Sprintf("p%v.win4.top.spr", i+1)]
		}
		if ok {
			ro.win4_top[i] = *ReadAnimLayout(fmt.Sprintf("p%v.win4.top.", i+1), is, sff, at, 1)
		} else {
			if _, ok = is["win4.top.anim"]; !ok {
				_, ok = is["win4.top.spr"]
			}
			if ok {
				ro.win4_top[i] = *ReadAnimLayout("win4.top.", is, sff, at, 1)
			} else {
				ro.win4_top[i] = ro.win2_top[i]
			}
		}
		for j := range ro.win4_bg[i] {
			if _, ok = is[fmt.Sprintf("p%v.win4.bg%v.anim", i+1, j)]; !ok {
				_, ok = is[fmt.Sprintf("p%v.win4.bg%v.spr", i+1, j)]
			}
			if ok {
				ro.win4_bg[i][j] = *ReadAnimLayout(fmt.Sprintf("p%v.win4.bg%v.", i+1, j), is, sff, at, 1)
				bg = true
			} else {
				if _, ok = is[fmt.Sprintf("win4.bg%v.anim", j)]; !ok {
					_, ok = is[fmt.Sprintf("win4.bg%v.spr", j)]
				}
				if ok {
					ro.win4_bg[i][j] = *ReadAnimLayout(fmt.Sprintf("win4.bg%v.", j), is, sff, at, 1)
					bg = true
				}
			}
		}
		if !bg {
			ro.win4_bg[i] = ro.win2_bg[i]
		}
	}
	ro.drawgame = *ReadAnimTextSnd("draw.", is, sff, at, 1, f)
	ro.drawgame_top = *ReadAnimLayout("draw.top.", is, sff, at, 1)
	for i := range ro.drawgame_bg {
		ro.drawgame_bg[i] = *ReadAnimLayout(fmt.Sprintf("draw.bg%v.", i), is, sff, at, 1)
	}
	ro.winType[WT_Normal] = readLbBgTextSnd("p1.n.", is, sff, at, 0, f)
	ro.winType[WT_Special] = readLbBgTextSnd("p1.s.", is, sff, at, 0, f)
	ro.winType[WT_Hyper] = readLbBgTextSnd("p1.h.", is, sff, at, 0, f)
	ro.winType[WT_Cheese] = readLbBgTextSnd("p1.c.", is, sff, at, 0, f)
	ro.winType[WT_Time] = readLbBgTextSnd("p1.t.", is, sff, at, 0, f)
	ro.winType[WT_Throw] = readLbBgTextSnd("p1.throw.", is, sff, at, 0, f)
	ro.winType[WT_Suicide] = readLbBgTextSnd("p1.suicide.", is, sff, at, 0, f)
	ro.winType[WT_Teammate] = readLbBgTextSnd("p1.teammate.", is, sff, at, 0, f)
	ro.winType[WT_Perfect] = readLbBgTextSnd("p1.perfect.", is, sff, at, 0, f)
	ro.winType[WT_Normal+WT_NumTypes] = readLbBgTextSnd("p2.n.", is, sff, at, 0, f)
	ro.winType[WT_Special+WT_NumTypes] = readLbBgTextSnd("p2.s.", is, sff, at, 0, f)
	ro.winType[WT_Hyper+WT_NumTypes] = readLbBgTextSnd("p2.h.", is, sff, at, 0, f)
	ro.winType[WT_Cheese+WT_NumTypes] = readLbBgTextSnd("p2.c.", is, sff, at, 0, f)
	ro.winType[WT_Time+WT_NumTypes] = readLbBgTextSnd("p2.t.", is, sff, at, 0, f)
	ro.winType[WT_Throw+WT_NumTypes] = readLbBgTextSnd("p2.throw.", is, sff, at, 0, f)
	ro.winType[WT_Suicide+WT_NumTypes] = readLbBgTextSnd("p2.suicide.", is, sff, at, 0, f)
	ro.winType[WT_Teammate+WT_NumTypes] = readLbBgTextSnd("p2.teammate.", is, sff, at, 0, f)
	ro.winType[WT_Perfect+WT_NumTypes] = readLbBgTextSnd("p2.perfect.", is, sff, at, 0, f)
	is.ReadI32("fadein.time", &ro.fadein_time)
	var col [3]int32
	if is.ReadI32("fadein.col", &col[0], &col[1], &col[2]) {
		ro.fadein_col = uint32(col[0]&0xff<<16 | col[1]&0xff<<8 | col[2]&0xff)
	}
	is.ReadI32("fadeout.time", &ro.fadeout_time)
	ro.over_time = Max(ro.fadeout_time, ro.over_time)
	col = [...]int32{0, 0, 0}
	if is.ReadI32("fadeout.col", &col[0], &col[1], &col[2]) {
		ro.fadeout_col = uint32(col[0]&0xff<<16 | col[1]&0xff<<8 | col[2]&0xff)
	}
	is.ReadI32("shutter.time", &ro.shutter_time)
	col = [...]int32{0, 0, 0}
	if is.ReadI32("shutter.col", &col[0], &col[1], &col[2]) {
		ro.shutter_col = uint32(col[0]&0xff<<16 | col[1]&0xff<<8 | col[2]&0xff)
	}
	is.ReadI32("callfight.time", &ro.callfight_time)
	return ro
}

func (ro *LifeBarRound) isSingleRound() bool {
	return !sys.consecutiveRounds && sys.round == 1 && sys.decisiveRound[0] && sys.decisiveRound[1]
}

func (ro *LifeBarRound) isFinalRound() bool {
	return !sys.consecutiveRounds && sys.round > 1 && sys.decisiveRound[0] && sys.decisiveRound[1] &&
		(sys.draws >= sys.lifebar.ro.match_maxdrawgames[0] || sys.draws >= sys.lifebar.ro.match_maxdrawgames[1])
}

func (ro *LifeBarRound) act() bool {
	// Reset FightScreenState trigger flags
	// This method is easier and more accurate than computing the times again for the trigger
	ro.triggerRoundDisplay = false
	ro.triggerFightDisplay = false
	ro.triggerKODisplay = false
	ro.triggerWinDisplay = false
	// Early exits
	if (sys.paused && !sys.step) || sys.gsf(GSF_roundfreeze) {
		return false
	}
	// Pre-intro
	if sys.intro > ro.ctrl_time {
		ro.current = 0
		ro.waitTimer[0], ro.waitSoundTimer[0], ro.drawTimer[0] = ro.round_time, ro.round_sndtime, 0
		ro.waitTimer[1] = ro.callfight_time
	} else if (sys.intro >= 0 && !sys.tickNextFrame()) || sys.shuttertime > 0 || sys.dialogueFlg {
		// Skip announcements during the middle of the round, "shuttertime" or dialogues
		// Mugen ignores the "shuttertime" here, but that makes the round/fight announcement too abrupt
		return false
	} else {
		// Check if current round animation can be skipped
		// This is to prevent suddenly ending the animations if a flag is enabled
		canSkip := func(phase int) bool {
			if phase < len(ro.waitTimer) && phase < len(ro.waitSoundTimer) && phase < len(ro.drawTimer) {
				return ro.waitTimer[phase] >= 0 && ro.waitSoundTimer[phase] >= 0 && ro.drawTimer[phase] <= 0
			}
			return false
		}
		// Round intro. Consists of round and fight calls
		if !ro.roundCallOver || !ro.fightCallOver {

			if sys.round == 1 && sys.intro == ro.ctrl_time && len(sys.cfg.Common.Lua) > 0 {
				for _, p := range sys.chars {
					if len(p) > 0 && len(p[0].dialogue) > 0 {
						sys.posReset()
						sys.dialogueFlg = true
						return false
					}
				}
			}
			// Previously skipping the char intros took us to the fight call, like Mugen
			// Most games go to the round call instead so this was changed
			//if sys.introSkipped && !sys.dialogueFlg {
			//	ro.roundCallOver = true
			//	ro.callFight()
			//	sys.introSkipped = false
			//}
			// Round call
			if sys.gsf(GSF_skiprounddisplay) && canSkip(0) { // Skip
				ro.roundCallOver = true
				ro.waitTimer[1] = 0
			}
			if !ro.roundCallOver {
				roundNum := sys.round
				if sys.consecutiveRounds {
					roundNum = sys.consecutiveWins[0] + 1
				}
				// Sounds
				if ro.waitSoundTimer[0] == 0 {
					if ro.isSingleRound() && ro.round_single.snd[0] != -1 {
						ro.snd.play(ro.round_single.snd, 100, 0, 0, 0, 0)
					} else if ro.isFinalRound() && ro.round_final.snd[0] != -1 {
						ro.snd.play(ro.round_final.snd, 100, 0, 0, 0, 0)
					} else if int(roundNum) <= len(ro.round) && ro.round[roundNum-1].snd[0] != -1 {
						ro.snd.play(ro.round[roundNum-1].snd, 100, 0, 0, 0, 0)
					} else {
						ro.snd.play(ro.round_default.snd, 100, 0, 0, 0, 0)
					}
				}
				ro.waitSoundTimer[0]--
				// Animations
				if ro.waitTimer[0] <= 0 {
					ro.triggerRoundDisplay = true
					ro.drawTimer[0]++
					if ro.isSingleRound() && ro.round_single.snd[0] != -1 {
						if len(ro.round_single_top.anim.frames) > 0 {
							ro.round_single_top.Action()
						} else {
							ro.round_default_top.Action()
						}
						ro.round_single.Action()
						ro.round_default.Action()
						if len(ro.round_single_bg[0].anim.frames) > 0 {
							for i := len(ro.round_single_bg) - 1; i >= 0; i-- {
								ro.round_single_bg[i].Action()
							}
						} else {
							for i := len(ro.round_default_bg) - 1; i >= 0; i-- {
								ro.round_default_bg[i].Action()
							}
						}
						ro.roundCallOver = ro.round_single.End(ro.drawTimer[0], true) && ro.round_default.End(ro.drawTimer[0], true)
					} else if ro.isFinalRound() && ro.round_final.snd[0] != -1 {
						if len(ro.round_final_top.anim.frames) > 0 {
							ro.round_final_top.Action()
						} else {
							ro.round_default_top.Action()
						}
						ro.round_final.Action()
						ro.round_default.Action()
						if len(ro.round_final_bg[0].anim.frames) > 0 {
							for i := len(ro.round_final_bg) - 1; i >= 0; i-- {
								ro.round_final_bg[i].Action()
							}
						} else {
							for i := len(ro.round_default_bg) - 1; i >= 0; i-- {
								ro.round_default_bg[i].Action()
							}
						}
						ro.roundCallOver = ro.round_final.End(ro.drawTimer[0], true) && ro.round_default.End(ro.drawTimer[0], true)
					} else if int(roundNum) <= len(ro.round) {
						ro.round_default_top.Action()
						ro.round[roundNum-1].Action()
						ro.round_default.Action()
						for i := len(ro.round_default_bg) - 1; i >= 0; i-- {
							ro.round_default_bg[i].Action()
						}
						ro.roundCallOver = ro.round[roundNum-1].End(ro.drawTimer[0], true) && ro.round_default.End(ro.drawTimer[0], true)
					} else {
						ro.round_default_top.Action()
						ro.round_default.Action()
						for i := len(ro.round_default_bg) - 1; i >= 0; i-- {
							ro.round_default_bg[i].Action()
						}
						ro.roundCallOver = ro.round_default.End(ro.drawTimer[0], true)
					}
				}
				ro.waitTimer[0]--
			}
			// Fight call
			endFightCall := func() {
				ro.current = 2
				ro.waitTimer[2], ro.waitSoundTimer[2], ro.drawTimer[2] = ro.ko_time, ro.ko_sndtime, 0
				ro.waitTimer[3], ro.waitSoundTimer[3], ro.drawTimer[3] = ro.win_time, ro.win_sndtime, 0
				ro.fightCallOver = true
			}
			// Skip fight call
			// Cannot be skipped unless round call is finished or also skipped
			if ro.roundCallOver && sys.gsf(GSF_skipfightdisplay) && canSkip(1) {
				endFightCall()
				if sys.intro > 1 {
					sys.intro = 1 // Skip ctrl waiting time
				}
			}
			if !ro.fightCallOver {
				if ro.current == 0 {
					if ro.waitTimer[1] == 0 {
						// This used to be callFight()
						ro.fight.Reset()
						ro.fight_top.Reset()
						ro.current = 1
						ro.waitTimer[1] = ro.fight_time
						ro.waitSoundTimer[1] = ro.fight_sndtime
						ro.drawTimer[1] = 0
						sys.timerCount = append(sys.timerCount, sys.gameTime)
						ro.timerActive = true
					}
					ro.waitTimer[1]--
				} else if !ro.fightCallOver {
					if ro.waitSoundTimer[1] == 0 {
						ro.snd.play(ro.fight.snd, 100, 0, 0, 0, 0)
					}
					ro.waitSoundTimer[1]--
					if ro.waitTimer[1] <= 0 {
						ro.triggerFightDisplay = true
						ro.drawTimer[1]++
						ro.fight_top.Action()
						ro.fight.Action()
						for i := len(ro.fight_bg) - 1; i >= 0; i-- {
							ro.fight_bg[i].Action()
						}
						if ro.fight.End(ro.drawTimer[1], true) && ro.waitSoundTimer[1] < 0 {
							endFightCall()
						}
					}
					ro.waitTimer[1]--
				}
			}
		}
		// Round over. Consists of KO screen and winner messages
		if ro.current == 2 && sys.intro < 0 && (sys.finishType != FT_NotYet || sys.time == 0) {
			if ro.timerActive {
				if sys.gameTime-sys.timerCount[sys.round-1] > 0 {
					sys.timerCount[sys.round-1] = sys.gameTime - sys.timerCount[sys.round-1]
					sys.timerRounds = append(sys.timerRounds, sys.roundTime-sys.time)
				} else {
					sys.timerCount[sys.round-1] = 0
				}
				ro.timerActive = false
			}
			steptimers := func(ats *AnimTextSnd, t int, delay int32, name string) {
				if ro.waitSoundTimer[t]+delay == 0 {
					ro.snd.play(ats.snd, 100, 0, 0, 0, 0)
					ro.waitSoundTimer[t]--
				}
				ro.waitSoundTimer[t]--
				if ats.End(ro.drawTimer[t], false) {
					ro.waitTimer[t] = 2
				}
				if ro.waitTimer[t]+delay <= 0 {
					ro.drawTimer[t]++
					ats.Action()
					// Flag FightScreenState while anims are playing
					if !ats.End(ro.drawTimer[t], true) {
						switch name {
						case "ko":
							ro.triggerKODisplay = true
						case "win":
							ro.triggerWinDisplay = true
						}
					}
				}
				ro.waitTimer[t]--
			}
			// KO screen
			if !(sys.gsf(GSF_skipkodisplay) && canSkip(2)) {
				switch sys.finishType {
				case FT_KO:
					ro.ko_top.Action()
					steptimers(&ro.ko, 2, 9, "ko")
					for i := len(ro.ko_bg) - 1; i >= 0; i-- {
						ro.ko_bg[i].Action()
					}
				case FT_DKO:
					ro.dko_top.Action()
					steptimers(&ro.dko, 2, 9, "ko")
					for i := len(ro.dko_bg) - 1; i >= 0; i-- {
						ro.dko_bg[i].Action()
					}
				default:
					ro.to_top.Action()
					steptimers(&ro.to, 2, 0, "ko") // In Mugen there's no delay between the time over text and the sound
					for i := len(ro.to_bg) - 1; i >= 0; i-- {
						ro.to_bg[i].Action()
					}
				}
			}
			// Winner announcement
			if sys.intro < -(ro.over_waittime) && !(sys.gsf(GSF_skipwindisplay) && canSkip(3)) {
				wt := sys.winTeam
				if wt < 0 {
					wt = 0
				}
				if sys.finishType == FT_TODraw {
					ro.drawgame_top.Action()
					steptimers(&ro.drawgame, 3, 0, "win")
					for i := len(ro.drawgame_bg) - 1; i >= 0; i-- {
						ro.drawgame_bg[i].Action()
					}
				} else if sys.winTeam >= 0 { // Skip if draw game (double KO)
					if sys.tmode[sys.winTeam] == TM_Simul || sys.tmode[sys.winTeam] == TM_Tag {
						if sys.numSimul[sys.winTeam] == 2 {
							ro.win2_top[wt].Action()
							steptimers(&ro.win2[wt], 3, 0, "win")
							for i := len(ro.win2_bg[wt]) - 1; i >= 0; i-- {
								ro.win2_bg[wt][i].Action()
							}
						} else if sys.numSimul[sys.winTeam] == 3 {
							ro.win3_top[wt].Action()
							steptimers(&ro.win3[wt], 3, 0, "win")
							for i := len(ro.win3_bg[wt]) - 1; i >= 0; i-- {
								ro.win3_bg[wt][i].Action()
							}
						} else {
							ro.win4_top[wt].Action()
							steptimers(&ro.win4[wt], 3, 0, "win")
							for i := len(ro.win4_bg[wt]) - 1; i >= 0; i-- {
								ro.win4_bg[wt][i].Action()
							}
						}
					} else {
						ro.win_top[wt].Action()
						steptimers(&ro.win[wt], 3, 0, "win")
						for i := len(ro.win_bg[wt]) - 1; i >= 0; i-- {
							ro.win_bg[wt][i].Action()
						}
					}
				}
				// Perfect and other special win types
				if sys.winTeam >= 0 {
					index := sys.winType[sys.winTeam]
					if index > WT_NumTypes {
						if sys.winTeam == 0 {
							ro.winType[WT_Perfect].step(ro.snd)
							index = index - WT_NumTypes - 1
						} else {
							ro.winType[WT_Perfect+WT_NumTypes].step(ro.snd)
							index = index - 1
						}
					}
					ro.winType[index].step(ro.snd)
				}
			}
		} else {
			return ro.current > 0
		}
	}
	return sys.tickNextFrame()
}

func (ro *LifeBarRound) reset() {
	ro.current = 0
	// Round animations
	ro.round_default.Reset()
	ro.round_default_top.Reset()
	for i := range ro.round_default_bg {
		ro.round_default_bg[i].Reset()
	}
	for i := range ro.round {
		ro.round[i].Reset()
	}
	// Single round animations
	ro.round_single.Reset()
	ro.round_single_top.Reset()
	for i := range ro.round_single_bg {
		ro.round_single_bg[i].Reset()
	}
	// Final round animations
	ro.round_final.Reset()
	ro.round_final_top.Reset()
	for i := range ro.round_final_bg {
		ro.round_final_bg[i].Reset()
	}
	// Fight call animations
	ro.fight.Reset()
	ro.fight_top.Reset()
	for i := range ro.fight_bg {
		ro.fight_bg[i].Reset()
	}
	// KO animations
	ro.ko.Reset()
	ro.ko_top.Reset()
	for i := range ro.ko_bg {
		ro.ko_bg[i].Reset()
	}
	ro.dko.Reset()
	ro.dko_top.Reset()
	for i := range ro.dko_bg {
		ro.dko_bg[i].Reset()
	}
	// Time Over animations
	ro.to.Reset()
	ro.to_top.Reset()
	for i := range ro.to_bg {
		ro.to_bg[i].Reset()
	}
	for i := range ro.win {
		ro.win[i].Reset()
	}
	for i := range ro.win_top {
		ro.win_top[i].Reset()
	}
	for i := range ro.win_bg {
		for j := range ro.win_bg[i] {
			ro.win_bg[i][j].Reset()
		}
	}
	for i := range ro.win2 {
		ro.win2[i].Reset()
	}
	for i := range ro.win2_top {
		ro.win2_top[i].Reset()
	}
	for i := range ro.win2_bg {
		for j := range ro.win2_bg[i] {
			ro.win2_bg[i][j].Reset()
		}
	}
	for i := range ro.win3 {
		ro.win3[i].Reset()
	}
	for i := range ro.win3_top {
		ro.win3_top[i].Reset()
	}
	for i := range ro.win3_bg {
		for j := range ro.win3_bg[i] {
			ro.win3_bg[i][j].Reset()
		}
	}
	for i := range ro.win4 {
		ro.win4[i].Reset()
	}
	for i := range ro.win4_top {
		ro.win4_top[i].Reset()
	}
	for i := range ro.win4_bg {
		for j := range ro.win4_bg[i] {
			ro.win4_bg[i][j].Reset()
		}
	}
	// Draw game
	ro.drawgame.Reset()
	ro.drawgame_top.Reset()
	for i := range ro.drawgame_bg {
		ro.drawgame_bg[i].Reset()
	}
	// Win types
	for i := range ro.winType {
		ro.winType[i].reset()
	}
	// Reset action timers
	ro.waitTimer = [4]int32{}
	ro.waitSoundTimer = [4]int32{}
	ro.drawTimer = [4]int32{}
	ro.roundCallOver = false
	ro.fightCallOver = false
}

func (ro *LifeBarRound) draw(layerno int16, f []*Fnt) {
	ob := sys.brightness
	sys.brightness = 256

	// Round call animations
	if !ro.roundCallOver && ro.waitTimer[0] < 0 && sys.intro <= ro.ctrl_time {

		// Draw default round background
		for i := range ro.round_default_bg {
			ro.round_default_bg[i].Draw(
				float32(ro.pos[0])+sys.lifebarOffsetX,
				float32(ro.pos[1]),
				layerno,
				sys.lifebarScale,
			)
		}

		// Check round number
		var round_ref AnimTextSnd
		roundNum := sys.round
		if sys.consecutiveRounds {
			roundNum = sys.consecutiveWins[0] + 1
		}

		// Draw background
		if ro.isSingleRound() &&
			(ro.round_single.text.font[0] != -1 || len(ro.round_single.anim.anim.frames) > 0 || len(ro.round_single_bg[0].anim.frames) > 0) {
			// Single round
			for i := range ro.round_single_bg {
				ro.round_single_bg[i].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
			}
			round_ref = ro.round_single
		} else if ro.isFinalRound() &&
			(ro.round_final.text.font[0] != -1 || len(ro.round_final.anim.anim.frames) > 0 || len(ro.round_final_bg[0].anim.frames) > 0) {
			// Final round
			for i := range ro.round_final_bg {
				ro.round_final_bg[i].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
			}
			round_ref = ro.round_final
		} else if int(roundNum) <= len(ro.round) {
			// Otherwise, use the appropriate round reference
			round_ref = ro.round[roundNum-1]
		}

		// Backup default text
		tmp := ro.round_default.text.text

		// If round_ref text is empty, format the default round text
		if round_ref.text.text == "" {
			ro.round_default.text.text = OldSprintf(tmp, roundNum)
		} else {
			ro.round_default.text.text = ""
		}

		// Draw default round
		ro.round_default.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, f, sys.lifebarScale)

		// Restore default text
		ro.round_default.text.text = tmp

		// Backup round_ref text
		tmp = round_ref.text.text

		// Format the round_ref text with the round number
		round_ref.text.text = OldSprintf(tmp, roundNum)

		// Draw round-specific elements
		round_ref.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, f, sys.lifebarScale)

		// Restore round_ref text
		round_ref.text.text = tmp

		// Draw the single or final top layer if appropriate, otherwise draw the default top layer
		if ro.isSingleRound() && len(ro.round_single_top.anim.frames) > 0 {
			ro.round_single_top.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
		} else if ro.isFinalRound() && len(ro.round_final_top.anim.frames) > 0 {
			ro.round_final_top.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
		} else {
			ro.round_default_top.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
		}
	}

	// "Fight!" animations
	if !ro.fightCallOver && ro.waitTimer[1] < 0 {
		for i := range ro.fight_bg {
			ro.fight_bg[i].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
		}
		ro.fight.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, f, sys.lifebarScale)
		ro.fight_top.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
	}

	if ro.current == 2 {
		// KO animations
		if ro.waitTimer[2] < 0 {
			switch sys.finishType {
			case FT_KO:
				for i := range ro.ko_bg {
					ro.ko_bg[i].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
				}
				ro.ko.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, f, sys.lifebarScale)
				ro.ko_top.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
			case FT_DKO:
				for i := range ro.dko_bg {
					ro.dko_bg[i].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
				}
				ro.dko.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, f, sys.lifebarScale)
				ro.dko_top.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
			default:
				for i := range ro.to_bg {
					ro.to_bg[i].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
				}
				ro.to.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, f, sys.lifebarScale)
				ro.to_top.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
			}
		}
		// Winner announcement
		if ro.waitTimer[3] < 0 {
			wt := sys.winTeam
			if wt < 0 {
				wt = 0
			}
			if sys.finishType == FT_TODraw {
				for i := range ro.drawgame_bg {
					ro.drawgame_bg[i].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
				}
				ro.drawgame.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, f, sys.lifebarScale)
				ro.drawgame_top.Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
			} else if sys.winTeam >= 0 { // Skip if draw game (double KO)
				if sys.tmode[sys.winTeam] == TM_Simul || sys.tmode[sys.winTeam] == TM_Tag {
					var inter []interface{}
					for i := sys.winTeam; i < len(sys.chars); i += 2 {
						if len(sys.chars[i]) > 0 {
							inter = append(inter, sys.cgi[i].displayname)
						}
					}
					if sys.numSimul[sys.winTeam] == 2 {
						tmp := ro.win2[wt].text.text
						for i := range ro.win2_bg[wt] {
							ro.win2_bg[wt][i].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
						}
						ro.win2[wt].text.text = OldSprintf(tmp, inter...)
						ro.win2[wt].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, f, sys.lifebarScale)
						ro.win2[wt].text.text = tmp
						ro.win2_top[wt].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
					} else if sys.numSimul[sys.winTeam] == 3 {
						tmp := ro.win3[wt].text.text
						for i := range ro.win3_bg[wt] {
							ro.win3_bg[wt][i].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
						}
						ro.win3[wt].text.text = OldSprintf(tmp, inter...)
						ro.win3[wt].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, f, sys.lifebarScale)
						ro.win3[wt].text.text = tmp
						ro.win3_top[wt].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
					} else {
						tmp := ro.win4[wt].text.text
						for i := range ro.win4_bg[wt] {
							ro.win4_bg[wt][i].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
						}
						ro.win4[wt].text.text = OldSprintf(tmp, inter...)
						ro.win4[wt].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, f, sys.lifebarScale)
						ro.win4[wt].text.text = tmp
						ro.win4_top[wt].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
					}
				} else if sys.winTeam >= 0 {
					tmp := ro.win[wt].text.text
					for i := range ro.win_bg[wt] {
						ro.win_bg[wt][i].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
					}
					ro.win[wt].text.text = OldSprintf(tmp, sys.cgi[sys.winTeam].displayname)
					ro.win[wt].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, f, sys.lifebarScale)
					ro.win[wt].text.text = tmp
					ro.win_top[wt].Draw(float32(ro.pos[0])+sys.lifebarOffsetX, float32(ro.pos[1]), layerno, sys.lifebarScale)
				}
			}
			// Perfect and other special win types
			if sys.winTeam >= 0 {
				index := sys.winType[sys.winTeam]
				perfect := false
				if index > WT_NumTypes {
					if sys.winTeam == 0 {
						index = index - WT_NumTypes - 1
					} else {
						index = index - 1
					}
					perfect = true
				}
				if perfect {
					if sys.winTeam == 0 {
						ro.winType[WT_Perfect].bgDraw(layerno)
						ro.winType[WT_Perfect].draw(layerno, f)
					} else {
						ro.winType[WT_Perfect+WT_NumTypes].bgDraw(layerno)
						ro.winType[WT_Perfect+WT_NumTypes].draw(layerno, f)
					}
				}
				ro.winType[index].bgDraw(layerno)
				ro.winType[index].draw(layerno, f)
			}
		}
	}
	sys.brightness = ob
}

type LifeBarRatio struct {
	pos  [2]int32
	icon [4]AnimLayout
	bg   AnimLayout
	top  AnimLayout
}

func newLifeBarRatio() *LifeBarRatio {
	return &LifeBarRatio{}
}

func readLifeBarRatio(pre string, is IniSection,
	sff *Sff, at AnimationTable) *LifeBarRatio {
	ra := newLifeBarRatio()
	is.ReadI32(pre+"pos", &ra.pos[0], &ra.pos[1])
	ra.icon[0] = *ReadAnimLayout(pre+"level1.", is, sff, at, 0)
	ra.icon[1] = *ReadAnimLayout(pre+"level2.", is, sff, at, 0)
	ra.icon[2] = *ReadAnimLayout(pre+"level3.", is, sff, at, 0)
	ra.icon[3] = *ReadAnimLayout(pre+"level4.", is, sff, at, 0)
	ra.bg = *ReadAnimLayout(pre+"bg.", is, sff, at, 0)
	return ra
}

func (ra *LifeBarRatio) step(num int32) {
	ra.icon[num].Action()
	ra.bg.Action()
}

func (ra *LifeBarRatio) reset() {
	for i := range ra.icon {
		ra.icon[i].Reset()
	}
	ra.bg.Reset()
}

func (ra *LifeBarRatio) bgDraw(layerno int16) {
	ra.bg.Draw(float32(ra.pos[0])+sys.lifebarOffsetX, float32(ra.pos[1]), layerno, sys.lifebarScale)
}

func (ra *LifeBarRatio) draw(layerno int16, num int32) {
	ra.icon[num].Draw(float32(ra.pos[0])+sys.lifebarOffsetX,
		float32(ra.pos[1]), layerno, sys.lifebarScale)
	ra.top.Draw(float32(ra.pos[0])+sys.lifebarOffsetX, float32(ra.pos[1]), layerno, sys.lifebarScale)
}

type LifeBarTimer struct {
	pos     [2]int32
	text    LbText
	bg      AnimLayout
	top     AnimLayout
	enabled map[string]bool
	active  bool
}

func newLifeBarTimer() *LifeBarTimer {
	return &LifeBarTimer{enabled: make(map[string]bool)}
}

func readLifeBarTimer(is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt) *LifeBarTimer {
	tr := newLifeBarTimer()
	is.ReadI32("pos", &tr.pos[0], &tr.pos[1])
	tr.text = *readLbText("text.", is, "", 0, f, 0)
	tr.bg = *ReadAnimLayout("bg.", is, sff, at, 0)
	tr.top = *ReadAnimLayout("top.", is, sff, at, 0)
	for k := range is {
		sp := strings.Split(k, ".")
		if len(sp) == 2 && sp[0] == "enabled" {
			var b bool
			if is.ReadBool(k, &b) {
				tr.enabled[sp[1]] = b
			}
		}
	}
	return tr
}

func (tr *LifeBarTimer) step() {
	tr.bg.Action()
	tr.top.Action()
	tr.text.step()
}

func (tr *LifeBarTimer) reset() {
	tr.bg.Reset()
	tr.top.Reset()
}

func (tr *LifeBarTimer) bgDraw(layerno int16) {
	if tr.active {
		tr.bg.Draw(float32(tr.pos[0])+sys.lifebarOffsetX, float32(tr.pos[1]), layerno, sys.lifebarScale)
	}
}

func (tr *LifeBarTimer) draw(layerno int16, f []*Fnt) {
	if tr.active && sys.lifebar.ti.framespercount > 0 &&
		tr.text.font[0] >= 0 && int(tr.text.font[0]) < len(f) && f[tr.text.font[0]] != nil && sys.time >= 0 {
		text := tr.text.text
		totalSec := float64(timeTotal()) / 60
		h := math.Floor(totalSec / 3600)
		m := math.Floor((totalSec/3600 - h) * 60)
		s := math.Floor(((totalSec/3600-h)*60 - m) * 60)
		x := math.Floor((((totalSec/3600-h)*60-m)*60 - s) * 100)
		ms, ss, xs := fmt.Sprintf("%.0f", m), fmt.Sprintf("%.0f", s), fmt.Sprintf("%.0f", x)
		if len(ms) < 2 {
			ms = "0" + ms
		}
		if len(ss) < 2 {
			ss = "0" + ss
		}
		if len(xs) < 2 {
			xs = "0" + xs
		}
		text = strings.Replace(text, "%m", ms, 1)
		text = strings.Replace(text, "%s", ss, 1)
		text = strings.Replace(text, "%x", xs, 1)
		tr.text.lay.DrawText(float32(tr.pos[0])+sys.lifebarOffsetX, float32(tr.pos[1]), sys.lifebarScale, layerno,
			text, f[tr.text.font[0]], tr.text.font[1], tr.text.font[2], tr.text.palfx, tr.text.frgba)
		tr.top.Draw(float32(tr.pos[0])+sys.lifebarOffsetX, float32(tr.pos[1]), layerno, sys.lifebarScale)
	}
}

func timeElapsed() int32 {
	return sys.roundTime - sys.time
}

func timeRemaining() int32 {
	if sys.time >= 0 {
		return sys.time
	}
	return -1
}

func timeTotal() int32 {
	t := sys.timerStart
	for _, v := range sys.timerRounds {
		t += v
	}
	if sys.lifebar.ro.timerActive {
		t += timeElapsed()
	}
	return t
}

type LifeBarScore struct {
	pos         [2]int32
	text        LbText
	bg          AnimLayout
	top         AnimLayout
	separator   [2]string
	pad         int32
	places      int32
	min         float32
	max         float32
	scorePoints float32
	enabled     map[string]bool
	active      bool
}

func newLifeBarScore() *LifeBarScore {
	return &LifeBarScore{separator: [2]string{"", "."}, enabled: make(map[string]bool)}
}

func readLifeBarScore(pre string, is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt) *LifeBarScore {
	sc := newLifeBarScore()
	is.ReadI32(pre+"pos", &sc.pos[0], &sc.pos[1])
	sc.text = *readLbText(pre+"text.", is, "", 0, f, 0)
	sc.separator[0], _, _ = is.getText("format.integer.separator")
	sc.separator[1], _, _ = is.getText("format.decimal.separator")
	is.ReadI32("format.integer.pad", &sc.pad)
	is.ReadI32("format.decimal.places", &sc.places)
	is.ReadF32("score.min", &sc.min)
	is.ReadF32("score.max", &sc.max)
	sc.bg = *ReadAnimLayout(pre+"bg.", is, sff, at, 0)
	sc.top = *ReadAnimLayout(pre+"top.", is, sff, at, 0)
	for k := range is {
		sp := strings.Split(k, ".")
		if len(sp) == 3 && pre == fmt.Sprintf("%v.", sp[0]) && sp[1] == "enabled" {
			var b bool
			if is.ReadBool(k, &b) {
				sc.enabled[sp[2]] = b
			}
		}
	}
	return sc
}

func (sc *LifeBarScore) step() {
	sc.bg.Action()
	sc.top.Action()
	sc.text.step()
}

func (sc *LifeBarScore) reset() {
	sc.bg.Reset()
	sc.top.Reset()
	sc.scorePoints = 0
}

func (sc *LifeBarScore) bgDraw(layerno int16) {
	if sc.active {
		sc.bg.Draw(float32(sc.pos[0])+sys.lifebarOffsetX, float32(sc.pos[1]), layerno, sys.lifebarScale)
	}
}

func (sc *LifeBarScore) draw(layerno int16, f []*Fnt, side int) {
	if sc.active && sc.text.font[0] >= 0 && int(sc.text.font[0]) < len(f) && f[sc.text.font[0]] != nil {
		text := sc.text.text
		total := sys.chars[side][0].scoreTotal()
		if total == 0 && sc.pad == 0 {
			return
		}
		if total > sc.max {
			total = sc.max
		} else if total < sc.min {
			total = sc.min
		}
		// split float value
		s := strings.Split(fmt.Sprintf("%f", total), ".")
		// integer left padding (add leading zeros)
		for i := int(sc.pad) - len(s[0]); i > 0; i-- {
			s[0] = "0" + s[0]
		}
		// integer thousands separator
		for i := len(s[0]) - 3; i > 0; i -= 3 {
			s[0] = s[0][:i] + sc.separator[0] + s[0][i:]
		}
		// decimal places (trim trailing numbers)
		if int(sc.places) < len(s[1]) {
			s[1] = s[1][:sc.places]
		}
		// decimal separator
		ds := ""
		if sc.places > 0 {
			ds = sc.separator[1]
		}
		// replace %s with formatted string
		text = strings.Replace(text, "%s", s[0]+ds+s[1], 1)
		sc.text.lay.DrawText(float32(sc.pos[0])+sys.lifebarOffsetX, float32(sc.pos[1]), sys.lifebarScale, layerno,
			text, f[sc.text.font[0]], sc.text.font[1], sc.text.font[2], sc.text.palfx, sc.text.frgba)
		sc.top.Draw(float32(sc.pos[0])+sys.lifebarOffsetX, float32(sc.pos[1]), layerno, sys.lifebarScale)
	}
}

type LifeBarMatch struct {
	pos     [2]int32
	text    LbText
	bg      AnimLayout
	top     AnimLayout
	enabled map[string]bool
	active  bool
}

func newLifeBarMatch() *LifeBarMatch {
	return &LifeBarMatch{enabled: make(map[string]bool)}
}

func readLifeBarMatch(is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt) *LifeBarMatch {
	ma := newLifeBarMatch()
	is.ReadI32("pos", &ma.pos[0], &ma.pos[1])
	ma.text = *readLbText("text.", is, "", 0, f, 0)
	ma.bg = *ReadAnimLayout("bg.", is, sff, at, 0)
	ma.top = *ReadAnimLayout("top.", is, sff, at, 0)
	for k := range is {
		sp := strings.Split(k, ".")
		if len(sp) == 2 && sp[0] == "enabled" {
			var b bool
			if is.ReadBool(k, &b) {
				ma.enabled[sp[1]] = b
			}
		}
	}
	return ma
}

func (ma *LifeBarMatch) step() {
	ma.bg.Action()
	ma.top.Action()
	ma.text.step()
}

func (ma *LifeBarMatch) reset() {
	ma.bg.Reset()
	ma.top.Reset()
}

func (ma *LifeBarMatch) bgDraw(layerno int16) {
	if ma.active {
		ma.bg.Draw(float32(ma.pos[0])+sys.lifebarOffsetX, float32(ma.pos[1]), layerno, sys.lifebarScale)
	}
}

func (ma *LifeBarMatch) draw(layerno int16, f []*Fnt) {
	if ma.active && ma.text.font[0] >= 0 && int(ma.text.font[0]) < len(f) && f[ma.text.font[0]] != nil {
		text := ma.text.text
		text = strings.Replace(text, "%s", fmt.Sprintf("%v", sys.match), 1)
		ma.text.lay.DrawText(float32(ma.pos[0])+sys.lifebarOffsetX, float32(ma.pos[1]), sys.lifebarScale, layerno,
			text, f[ma.text.font[0]], ma.text.font[1], ma.text.font[2], ma.text.palfx, ma.text.frgba)
		ma.top.Draw(float32(ma.pos[0])+sys.lifebarOffsetX, float32(ma.pos[1]), layerno, sys.lifebarScale)
	}
}

type LifeBarAiLevel struct {
	pos       [2]int32
	text      LbText
	bg        AnimLayout
	top       AnimLayout
	separator string
	places    int32
	enabled   map[string]bool
	active    bool
}

func newLifeBarAiLevel() *LifeBarAiLevel {
	return &LifeBarAiLevel{separator: ".", enabled: make(map[string]bool)}
}

func readLifeBarAiLevel(pre string, is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt) *LifeBarAiLevel {
	ai := newLifeBarAiLevel()
	is.ReadI32(pre+"pos", &ai.pos[0], &ai.pos[1])
	ai.text = *readLbText(pre+"text.", is, "", 0, f, 0)
	ai.separator, _, _ = is.getText("format.decimal.separator")
	is.ReadI32("format.decimal.places", &ai.places)
	ai.bg = *ReadAnimLayout(pre+"bg.", is, sff, at, 0)
	ai.top = *ReadAnimLayout(pre+"top.", is, sff, at, 0)
	for k := range is {
		sp := strings.Split(k, ".")
		if len(sp) == 3 && pre == fmt.Sprintf("%v.", sp[0]) && sp[1] == "enabled" {
			var b bool
			if is.ReadBool(k, &b) {
				ai.enabled[sp[2]] = b
			}
		}
	}
	return ai
}

func (ai *LifeBarAiLevel) step() {
	ai.bg.Action()
	ai.top.Action()
	ai.text.step()
}

func (ai *LifeBarAiLevel) reset() {
	ai.bg.Reset()
	ai.top.Reset()
}

func (ai *LifeBarAiLevel) bgDraw(layerno int16) {
	if ai.active {
		ai.bg.Draw(float32(ai.pos[0])+sys.lifebarOffsetX, float32(ai.pos[1]), layerno, sys.lifebarScale)
	}
}

func (ai *LifeBarAiLevel) draw(layerno int16, f []*Fnt, ailv float32) {
	if ai.active && ailv > 0 && ai.text.font[0] >= 0 && int(ai.text.font[0]) < len(f) && f[ai.text.font[0]] != nil {
		text := ai.text.text
		// split float value
		s := strings.Split(fmt.Sprintf("%f", ailv), ".")
		// decimal places (trim trailing numbers)
		if int(ai.places) < len(s[1]) {
			s[1] = s[1][:ai.places]
		}
		// decimal separator
		ds := ""
		if ai.places > 0 {
			ds = ai.separator
		}
		// replace %s with formatted string
		text = strings.Replace(text, "%s", s[0]+ds+s[1], 1)
		// percentage value
		p := ailv / 8 * 100
		text = strings.Replace(text, "%p", fmt.Sprintf("%.0f", p), 1)
		ai.text.lay.DrawText(float32(ai.pos[0])+sys.lifebarOffsetX, float32(ai.pos[1]), sys.lifebarScale, layerno,
			text, f[ai.text.font[0]], ai.text.font[1], ai.text.font[2], ai.text.palfx, ai.text.frgba)
		ai.top.Draw(float32(ai.pos[0])+sys.lifebarOffsetX, float32(ai.pos[1]), layerno, sys.lifebarScale)
	}
}

type LifeBarWinCount struct {
	pos     [2]int32
	text    LbText
	bg      AnimLayout
	top     AnimLayout
	wins    int32
	enabled map[string]bool
	active  bool
}

func newLifeBarWinCount() *LifeBarWinCount {
	return &LifeBarWinCount{enabled: make(map[string]bool)}
}

func readLifeBarWinCount(pre string, is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt) *LifeBarWinCount {
	wc := newLifeBarWinCount()
	is.ReadI32(pre+"pos", &wc.pos[0], &wc.pos[1])
	wc.text = *readLbText(pre+"text.", is, "", 0, f, 0)
	wc.bg = *ReadAnimLayout(pre+"bg.", is, sff, at, 0)
	wc.top = *ReadAnimLayout(pre+"top.", is, sff, at, 0)
	for k := range is {
		sp := strings.Split(k, ".")
		if len(sp) == 3 && pre == fmt.Sprintf("%v.", sp[0]) && sp[1] == "enabled" {
			var b bool
			if is.ReadBool(k, &b) {
				wc.enabled[sp[2]] = b
			}
		}
	}
	return wc
}

func (wc *LifeBarWinCount) step() {
	wc.bg.Action()
	wc.top.Action()
	wc.text.step()
}

func (wc *LifeBarWinCount) reset() {
	wc.bg.Reset()
	wc.top.Reset()
}

func (wc *LifeBarWinCount) bgDraw(layerno int16) {
	if wc.active {
		wc.bg.Draw(float32(wc.pos[0])+sys.lifebarOffsetX, float32(wc.pos[1]), layerno, sys.lifebarScale)
	}
}

func (wc *LifeBarWinCount) draw(layerno int16, f []*Fnt, side int) {
	if wc.active && wc.text.font[0] >= 0 && int(wc.text.font[0]) < len(f) && f[wc.text.font[0]] != nil {
		text := wc.text.text
		text = strings.Replace(text, "%s", fmt.Sprintf("%v", wc.wins), 1)
		wc.text.lay.DrawText(float32(wc.pos[0])+sys.lifebarOffsetX, float32(wc.pos[1]), sys.lifebarScale, layerno,
			text, f[wc.text.font[0]], wc.text.font[1], wc.text.font[2], wc.text.palfx, wc.text.frgba)
		wc.top.Draw(float32(wc.pos[0])+sys.lifebarOffsetX, float32(wc.pos[1]), layerno, sys.lifebarScale)
	}
}

type LifeBarMode struct {
	pos  [2]int32
	text LbText
	bg   AnimLayout
	top  AnimLayout
}

func newLifeBarMode() *LifeBarMode {
	return &LifeBarMode{}
}

func readLifeBarMode(is IniSection,
	sff *Sff, at AnimationTable, f []*Fnt) map[string]*LifeBarMode {
	mo := make(map[string]*LifeBarMode)
	for k := range is {
		sp := strings.Split(k, ".")
		if _, ok := mo[sp[0]]; !ok {
			mo[sp[0]] = newLifeBarMode()
			is.ReadI32(sp[0]+".pos", &mo[sp[0]].pos[0], &mo[sp[0]].pos[1])
			mo[sp[0]].text = *readLbText(sp[0]+".text.", is, "", 0, f, 0)
			mo[sp[0]].bg = *ReadAnimLayout(sp[0]+".bg.", is, sff, at, 0)
			mo[sp[0]].top = *ReadAnimLayout(sp[0]+".top.", is, sff, at, 0)
		}
	}
	return mo
}

func (mo *LifeBarMode) step() {
	mo.bg.Action()
	mo.top.Action()
	mo.text.step()
}

func (mo *LifeBarMode) reset() {
	mo.bg.Reset()
	mo.top.Reset()
}

func (mo *LifeBarMode) bgDraw(layerno int16) {
	if sys.lifebar.mode {
		mo.bg.Draw(float32(mo.pos[0])+sys.lifebarOffsetX, float32(mo.pos[1]), layerno, sys.lifebarScale)
	}
}

func (mo *LifeBarMode) draw(layerno int16, f []*Fnt) {
	if sys.lifebar.mode && mo.text.font[0] >= 0 && int(mo.text.font[0]) < len(f) && f[mo.text.font[0]] != nil {
		mo.text.lay.DrawText(float32(mo.pos[0])+sys.lifebarOffsetX, float32(mo.pos[1]), sys.lifebarScale, layerno,
			mo.text.text, f[mo.text.font[0]], mo.text.font[1], mo.text.font[2], mo.text.palfx, mo.text.frgba)
		mo.top.Draw(float32(mo.pos[0])+sys.lifebarOffsetX, float32(mo.pos[1]), layerno, sys.lifebarScale)
	}
}

type Lifebar struct {
	def        string
	name       string
	nameLow    string
	author     string
	authorLow  string
	at         AnimationTable
	sff        *Sff
	snd        *Snd
	fnt        [10]*Fnt
	ref        [2]int
	order      [2][]int
	hb         [8][]*HealthBar
	pb         [8][]*PowerBar
	gb         [8][]*GuardBar
	sb         [8][]*StunBar
	fa         [8][]*LifeBarFace
	nm         [8][]*LifeBarName
	wi         [2]*LifeBarWinIcon
	ti         *LifeBarTime
	co         [2]*LifeBarCombo
	ac         [2]*LifeBarAction
	ro         *LifeBarRound
	ra         [2]*LifeBarRatio
	tr         *LifeBarTimer
	sc         [2]*LifeBarScore
	ma         *LifeBarMatch
	ai         [2]*LifeBarAiLevel
	wc         [2]*LifeBarWinCount
	mo         map[string]*LifeBarMode
	missing    map[string]int
	active     bool
	bars       bool
	mode       bool
	redlifebar bool
	guardbar   bool
	stunbar    bool
	hidebars   bool
	fnt_scale  float32
	fx_limit   int
	textsprite []*TextSprite
}

func loadLifebar(def string) (*Lifebar, error) {
	str, err := LoadText(def)
	if err != nil {
		return nil, err
	}
	l := &Lifebar{sff: &Sff{}, snd: &Snd{},
		hb: [...][]*HealthBar{make([]*HealthBar, 2), make([]*HealthBar, 8),
			make([]*HealthBar, 2), make([]*HealthBar, 8), make([]*HealthBar, 6),
			make([]*HealthBar, 8), make([]*HealthBar, 6), make([]*HealthBar, 8)},
		pb: [...][]*PowerBar{make([]*PowerBar, 2), make([]*PowerBar, 8),
			make([]*PowerBar, 2), make([]*PowerBar, 8), make([]*PowerBar, 6),
			make([]*PowerBar, 8), make([]*PowerBar, 6), make([]*PowerBar, 8)},
		gb: [...][]*GuardBar{make([]*GuardBar, 2), make([]*GuardBar, 8),
			make([]*GuardBar, 2), make([]*GuardBar, 8), make([]*GuardBar, 6),
			make([]*GuardBar, 8), make([]*GuardBar, 6), make([]*GuardBar, 8)},
		sb: [...][]*StunBar{make([]*StunBar, 2), make([]*StunBar, 8),
			make([]*StunBar, 2), make([]*StunBar, 8), make([]*StunBar, 6),
			make([]*StunBar, 8), make([]*StunBar, 6), make([]*StunBar, 8)},
		fa: [...][]*LifeBarFace{make([]*LifeBarFace, 2), make([]*LifeBarFace, 8),
			make([]*LifeBarFace, 2), make([]*LifeBarFace, 8), make([]*LifeBarFace, 6),
			make([]*LifeBarFace, 8), make([]*LifeBarFace, 6), make([]*LifeBarFace, 8)},
		nm: [...][]*LifeBarName{make([]*LifeBarName, 2), make([]*LifeBarName, 8),
			make([]*LifeBarName, 2), make([]*LifeBarName, 8), make([]*LifeBarName, 6),
			make([]*LifeBarName, 8), make([]*LifeBarName, 6), make([]*LifeBarName, 8)},
		active: true, bars: true, mode: true, fnt_scale: 1, fx_limit: 3}
	l.missing = map[string]int{
		"[tag lifebar]": 3, "[simul_3p lifebar]": 4, "[simul_4p lifebar]": 5,
		"[tag_3p lifebar]": 6, "[tag_4p lifebar]": 7, "[simul powerbar]": 1,
		"[turns powerbar]": 2, "[tag powerbar]": 3, "[simul_3p powerbar]": 4,
		"[simul_4p powerbar]": 5, "[tag_3p powerbar]": 6, "[tag_4p powerbar]": 7,
		"[guardbar]": 0, "[simul guardbar]": 1, "[turns guardbar]": 2,
		"[tag guardbar]": 3, "[simul_3p guardbar]": 4, "[simul_4p guardbar]": 5,
		"[tag_3p guardbar]": 6, "[tag_4p guardbar]": 7, "[stunbar]": 0,
		"[simul stunbar]": 1, "[turns stunbar]": 2, "[tag stunbar]": 3,
		"[simul_3p stunbar]": 4, "[simul_4p stunbar]": 5, "[tag_3p stunbar]": 6,
		"[tag_4p stunbar]": 7, "[tag face]": 3, "[simul_3p face]": 4,
		"[simul_4p face]": 5, "[tag_3p face]": 6, "[tag_4p face]": 7,
		"[tag name]": 3, "[simul_3p name]": 4, "[simul_4p name]": 5,
		"[tag_3p name]": 6, "[tag_4p name]": 7, "[action]": -1, "[ratio]": -1,
		"[timer]": -1, "[score]": -1, "[match]": -1, "[ailevel]": -1,
		"[wincount]": -1, "[mode]": -1,
	}
	strc := strings.ToLower(strings.TrimSpace(str))
	for k := range l.missing {
		strc = strings.Replace(strc, ";"+k, "", -1)
		if strings.Contains(strc, k) {
			delete(l.missing, k)
		} else {
			str += "\n" + k
		}
	}
	lines, i := SplitAndTrim(str, "\n"), 0
	l.at = ReadAnimationTable(l.sff, &l.sff.palList, lines, &i)
	i = 0
	filesflg := true
	ffx := newFightFx()
	// Load Common FX first
	for _, key := range SortedKeys(sys.cfg.Common.Fx) {
		for _, v := range sys.cfg.Common.Fx[key] {
			if err := loadFightFx(v); err != nil {
				return nil, err
			}
		}
	}
	for i < len(lines) {
		is, name, subname := ReadIniSection(lines, &i)
		switch name {
		case "info":
			var b bool
			if is.ReadBool("doubleres", &b) {
				l.fnt_scale = 0.5
			}
			l.name, _, _ = is.getText("name")
			l.nameLow = strings.ToLower(l.name)
			l.author, _, _ = is.getText("author")
			l.authorLow = strings.ToLower(l.author)
		case "files":
			if filesflg {
				filesflg = false
				if is.LoadFile("sff", []string{def, sys.motifDir, "", "data/"},
					func(filename string) error {
						s, err := loadSff(filename, false)
						if err != nil {
							return err
						}
						*l.sff = *s
						return nil
					}); err != nil {
					return nil, err
				}
				if is.LoadFile("snd", []string{def, sys.motifDir, "", "data/"},
					func(filename string) error {
						s, err := LoadSnd(filename)
						if err != nil {
							return err
						}
						*l.snd = *s
						return nil
					}); err != nil {
					return nil, err
				}
				if is.LoadFile("fightfx.sff", []string{def, sys.motifDir, "", "data/"},
					func(filename string) error {
						s, err := loadSff(filename, false)
						if err != nil {
							return err
						}
						*ffx.fsff = *s
						return nil
					}); err != nil {
					return nil, err
				}
				if is.LoadFile("fightfx.air", []string{def, sys.motifDir, "", "data/"},
					func(filename string) error {
						str, err := LoadText(filename)
						if err != nil {
							return err
						}
						lines, i := SplitAndTrim(str, "\n"), 0
						ffx.fat = ReadAnimationTable(ffx.fsff, &ffx.fsff.palList, lines, &i)
						return nil
					}); err != nil {
					return nil, err
				}
				if is.LoadFile("common.snd", []string{def, sys.motifDir, "", "data/"},
					func(filename string) error {
						ffx.fsnd, err = LoadSnd(filename)
						return err
					}); err != nil {
					return nil, err
				}
				for i := 1; i <= l.fx_limit; i++ {
					if err := is.LoadFile(fmt.Sprintf("fx%v", i), []string{def, sys.motifDir, "", "data/"},
						func(filename string) error {
							if err := loadFightFx(filename); err != nil {
								return err
							}
							return nil
						}); err != nil {
						return nil, err
					}
				}
				for i := range l.fnt {
					/*if*/
					is.LoadFile(fmt.Sprintf("font%v", i), []string{def, sys.motifDir, "", "data/", "font/"},
						func(filename string) error {
							var height int32 = -1
							if len(is[fmt.Sprintf("font%v.height", i)]) > 0 {
								height = Atoi(is[fmt.Sprintf("font%v.height", i)])
							}
							if l.fnt[i], err = loadFnt(filename, height); err != nil {
								sys.errLog.Printf("failed to load %v (lifebar font): %v", filename, err)
								l.fnt[i] = newFnt()
							}
							return err
						},
					)
					/*err != nil {
						//return nil, err
					}*/
				}
			}
		case "fightfx":
			is.ReadF32("scale", &ffx.fx_scale)
		case "lifebar":
			if l.hb[0][0] == nil {
				l.hb[0][0] = readHealthBar("p1.", is, l.sff, l.at, l.fnt[:])
			}
			if l.hb[0][1] == nil {
				l.hb[0][1] = readHealthBar("p2.", is, l.sff, l.at, l.fnt[:])
			}
		case "powerbar":
			if l.pb[0][0] == nil {
				l.pb[0][0] = readPowerBar("p1.", is, l.sff, l.at, l.fnt[:])
			}
			if l.pb[0][1] == nil {
				l.pb[0][1] = readPowerBar("p2.", is, l.sff, l.at, l.fnt[:])
			}
		case "guardbar":
			if l.gb[0][0] == nil {
				l.gb[0][0] = readGuardBar("p1.", is, l.sff, l.at, l.fnt[:])
			}
			if l.gb[0][1] == nil {
				l.gb[0][1] = readGuardBar("p2.", is, l.sff, l.at, l.fnt[:])
			}
		case "stunbar":
			if l.sb[0][0] == nil {
				l.sb[0][0] = readStunBar("p1.", is, l.sff, l.at, l.fnt[:])
			}
			if l.sb[0][1] == nil {
				l.sb[0][1] = readStunBar("p2.", is, l.sff, l.at, l.fnt[:])
			}
		case "face":
			if l.fa[0][0] == nil {
				l.fa[0][0] = readLifeBarFace("p1.", is, l.sff, l.at)
			}
			if l.fa[0][1] == nil {
				l.fa[0][1] = readLifeBarFace("p2.", is, l.sff, l.at)
			}
		case "name":
			if l.nm[0][0] == nil {
				l.nm[0][0] = readLifeBarName("p1.", is, l.sff, l.at, l.fnt[:])
			}
			if l.nm[0][1] == nil {
				l.nm[0][1] = readLifeBarName("p2.", is, l.sff, l.at, l.fnt[:])
			}
		case "turns ":
			subname = strings.ToLower(subname)
			switch {
			case len(subname) >= 7 && subname[:7] == "lifebar":
				if l.hb[2][0] == nil {
					l.hb[2][0] = readHealthBar("p1.", is, l.sff, l.at, l.fnt[:])
				}
				if l.hb[2][1] == nil {
					l.hb[2][1] = readHealthBar("p2.", is, l.sff, l.at, l.fnt[:])
				}
			case len(subname) >= 8 && subname[:8] == "powerbar":
				if l.pb[2][0] == nil {
					l.pb[2][0] = readPowerBar("p1.", is, l.sff, l.at, l.fnt[:])
				}
				if l.pb[2][1] == nil {
					l.pb[2][1] = readPowerBar("p2.", is, l.sff, l.at, l.fnt[:])
				}
			case len(subname) >= 8 && subname[:8] == "guardbar":
				if l.gb[2][0] == nil {
					l.gb[2][0] = readGuardBar("p1.", is, l.sff, l.at, l.fnt[:])
				}
				if l.gb[2][1] == nil {
					l.gb[2][1] = readGuardBar("p2.", is, l.sff, l.at, l.fnt[:])
				}
			case len(subname) >= 7 && subname[:7] == "stunbar":
				if l.sb[2][0] == nil {
					l.sb[2][0] = readStunBar("p1.", is, l.sff, l.at, l.fnt[:])
				}
				if l.sb[2][1] == nil {
					l.sb[2][1] = readStunBar("p2.", is, l.sff, l.at, l.fnt[:])
				}
			case len(subname) >= 4 && subname[:4] == "face":
				if l.fa[2][0] == nil {
					l.fa[2][0] = readLifeBarFace("p1.", is, l.sff, l.at)
				}
				if l.fa[2][1] == nil {
					l.fa[2][1] = readLifeBarFace("p2.", is, l.sff, l.at)
				}
			case len(subname) >= 4 && subname[:4] == "name":
				if l.nm[2][0] == nil {
					l.nm[2][0] = readLifeBarName("p1.", is, l.sff, l.at, l.fnt[:])
				}
				if l.nm[2][1] == nil {
					l.nm[2][1] = readLifeBarName("p2.", is, l.sff, l.at, l.fnt[:])
				}
			}
		case "simul ", "simul_3p ", "simul_4p ", "tag ", "tag_3p ", "tag_4p ":
			i := 1 //"simul "
			switch name {
			case "tag ":
				i = 3
			case "simul_3p ":
				i = 4
			case "simul_4p ":
				i = 5
			case "tag_3p ":
				i = 6
			case "tag_4p ":
				i = 7
			}
			subname = strings.ToLower(subname)
			switch {
			case len(subname) >= 7 && subname[:7] == "lifebar":
				if l.hb[i][0] == nil {
					l.hb[i][0] = readHealthBar("p1.", is, l.sff, l.at, l.fnt[:])
				}
				if l.hb[i][1] == nil {
					l.hb[i][1] = readHealthBar("p2.", is, l.sff, l.at, l.fnt[:])
				}
				if l.hb[i][2] == nil {
					l.hb[i][2] = readHealthBar("p3.", is, l.sff, l.at, l.fnt[:])
				}
				if l.hb[i][3] == nil {
					l.hb[i][3] = readHealthBar("p4.", is, l.sff, l.at, l.fnt[:])
				}
				if l.hb[i][4] == nil {
					l.hb[i][4] = readHealthBar("p5.", is, l.sff, l.at, l.fnt[:])
				}
				if l.hb[i][5] == nil {
					l.hb[i][5] = readHealthBar("p6.", is, l.sff, l.at, l.fnt[:])
				}
				if i != 4 && i != 6 {
					if l.hb[i][6] == nil {
						l.hb[i][6] = readHealthBar("p7.", is, l.sff, l.at, l.fnt[:])
					}
					if l.hb[i][7] == nil {
						l.hb[i][7] = readHealthBar("p8.", is, l.sff, l.at, l.fnt[:])
					}
				}
			case len(subname) >= 8 && subname[:8] == "powerbar":
				if l.pb[i][0] == nil {
					l.pb[i][0] = readPowerBar("p1.", is, l.sff, l.at, l.fnt[:])
				}
				if l.pb[i][1] == nil {
					l.pb[i][1] = readPowerBar("p2.", is, l.sff, l.at, l.fnt[:])
				}
				if l.pb[i][2] == nil {
					l.pb[i][2] = readPowerBar("p3.", is, l.sff, l.at, l.fnt[:])
				}
				if l.pb[i][3] == nil {
					l.pb[i][3] = readPowerBar("p4.", is, l.sff, l.at, l.fnt[:])
				}
				if l.pb[i][4] == nil {
					l.pb[i][4] = readPowerBar("p5.", is, l.sff, l.at, l.fnt[:])
				}
				if l.pb[i][5] == nil {
					l.pb[i][5] = readPowerBar("p6.", is, l.sff, l.at, l.fnt[:])
				}
				if i != 4 && i != 6 {
					if l.pb[i][6] == nil {
						l.pb[i][6] = readPowerBar("p7.", is, l.sff, l.at, l.fnt[:])
					}
					if l.pb[i][7] == nil {
						l.pb[i][7] = readPowerBar("p8.", is, l.sff, l.at, l.fnt[:])
					}
				}
			case len(subname) >= 8 && subname[:8] == "guardbar":
				if l.gb[i][0] == nil {
					l.gb[i][0] = readGuardBar("p1.", is, l.sff, l.at, l.fnt[:])
				}
				if l.gb[i][1] == nil {
					l.gb[i][1] = readGuardBar("p2.", is, l.sff, l.at, l.fnt[:])
				}
				if l.gb[i][2] == nil {
					l.gb[i][2] = readGuardBar("p3.", is, l.sff, l.at, l.fnt[:])
				}
				if l.gb[i][3] == nil {
					l.gb[i][3] = readGuardBar("p4.", is, l.sff, l.at, l.fnt[:])
				}
				if l.gb[i][4] == nil {
					l.gb[i][4] = readGuardBar("p5.", is, l.sff, l.at, l.fnt[:])
				}
				if l.gb[i][5] == nil {
					l.gb[i][5] = readGuardBar("p6.", is, l.sff, l.at, l.fnt[:])
				}
				if i != 4 && i != 6 {
					if l.gb[i][6] == nil {
						l.gb[i][6] = readGuardBar("p7.", is, l.sff, l.at, l.fnt[:])
					}
					if l.gb[i][7] == nil {
						l.gb[i][7] = readGuardBar("p8.", is, l.sff, l.at, l.fnt[:])
					}
				}
			case len(subname) >= 7 && subname[:7] == "stunbar":
				if l.sb[i][0] == nil {
					l.sb[i][0] = readStunBar("p1.", is, l.sff, l.at, l.fnt[:])
				}
				if l.sb[i][1] == nil {
					l.sb[i][1] = readStunBar("p2.", is, l.sff, l.at, l.fnt[:])
				}
				if l.sb[i][2] == nil {
					l.sb[i][2] = readStunBar("p3.", is, l.sff, l.at, l.fnt[:])
				}
				if l.sb[i][3] == nil {
					l.sb[i][3] = readStunBar("p4.", is, l.sff, l.at, l.fnt[:])
				}
				if l.sb[i][4] == nil {
					l.sb[i][4] = readStunBar("p5.", is, l.sff, l.at, l.fnt[:])
				}
				if l.sb[i][5] == nil {
					l.sb[i][5] = readStunBar("p6.", is, l.sff, l.at, l.fnt[:])
				}
				if i != 4 && i != 6 {
					if l.sb[i][6] == nil {
						l.sb[i][6] = readStunBar("p7.", is, l.sff, l.at, l.fnt[:])
					}
					if l.sb[i][7] == nil {
						l.sb[i][7] = readStunBar("p8.", is, l.sff, l.at, l.fnt[:])
					}
				}
			case len(subname) >= 4 && subname[:4] == "face":
				if l.fa[i][0] == nil {
					l.fa[i][0] = readLifeBarFace("p1.", is, l.sff, l.at)
				}
				if l.fa[i][1] == nil {
					l.fa[i][1] = readLifeBarFace("p2.", is, l.sff, l.at)
				}
				if l.fa[i][2] == nil {
					l.fa[i][2] = readLifeBarFace("p3.", is, l.sff, l.at)
				}
				if l.fa[i][3] == nil {
					l.fa[i][3] = readLifeBarFace("p4.", is, l.sff, l.at)
				}
				if l.fa[i][4] == nil {
					l.fa[i][4] = readLifeBarFace("p5.", is, l.sff, l.at)
				}
				if l.fa[i][5] == nil {
					l.fa[i][5] = readLifeBarFace("p6.", is, l.sff, l.at)
				}
				if i != 4 && i != 6 {
					if l.fa[i][6] == nil {
						l.fa[i][6] = readLifeBarFace("p7.", is, l.sff, l.at)
					}
					if l.fa[i][7] == nil {
						l.fa[i][7] = readLifeBarFace("p8.", is, l.sff, l.at)
					}
				}
			case len(subname) >= 4 && subname[:4] == "name":
				if l.nm[i][0] == nil {
					l.nm[i][0] = readLifeBarName("p1.", is, l.sff, l.at, l.fnt[:])
				}
				if l.nm[i][1] == nil {
					l.nm[i][1] = readLifeBarName("p2.", is, l.sff, l.at, l.fnt[:])
				}
				if l.nm[i][2] == nil {
					l.nm[i][2] = readLifeBarName("p3.", is, l.sff, l.at, l.fnt[:])
				}
				if l.nm[i][3] == nil {
					l.nm[i][3] = readLifeBarName("p4.", is, l.sff, l.at, l.fnt[:])
				}
				if l.nm[i][4] == nil {
					l.nm[i][4] = readLifeBarName("p5.", is, l.sff, l.at, l.fnt[:])
				}
				if l.nm[i][5] == nil {
					l.nm[i][5] = readLifeBarName("p6.", is, l.sff, l.at, l.fnt[:])
				}
				if i != 4 && i != 6 {
					if l.nm[i][6] == nil {
						l.nm[i][6] = readLifeBarName("p7.", is, l.sff, l.at, l.fnt[:])
					}
					if l.nm[i][7] == nil {
						l.nm[i][7] = readLifeBarName("p8.", is, l.sff, l.at, l.fnt[:])
					}
				}
			}
		case "winicon":
			if l.wi[0] == nil {
				l.wi[0] = readLifeBarWinIcon("p1.", is, l.sff, l.at, l.fnt[:])
			}
			if l.wi[1] == nil {
				l.wi[1] = readLifeBarWinIcon("p2.", is, l.sff, l.at, l.fnt[:])
			}
		case "time":
			if l.ti == nil {
				l.ti = readLifeBarTime(is, l.sff, l.at, l.fnt[:])
			}
		case "combo":
			if l.co[0] == nil {
				if _, ok := is["team1.pos"]; ok {
					l.co[0] = readLifeBarCombo("team1.", is, l.sff, l.at, l.fnt[:], 0)
				} else {
					l.co[0] = readLifeBarCombo("", is, l.sff, l.at, l.fnt[:], 0)
				}
			}
			if l.co[1] == nil {
				if _, ok := is["team2.pos"]; ok {
					l.co[1] = readLifeBarCombo("team2.", is, l.sff, l.at, l.fnt[:], 1)
				} else {
					l.co[1] = readLifeBarCombo("", is, l.sff, l.at, l.fnt[:], 1)
				}
			}
		case "action":
			if l.ac[0] == nil {
				l.ac[0] = readLifeBarAction("team1.", is, l.fnt[:])
			}
			if l.ac[1] == nil {
				l.ac[1] = readLifeBarAction("team2.", is, l.fnt[:])
			}
		case "round":
			if l.ro == nil {
				l.ro = readLifeBarRound(is, l.sff, l.at, l.snd, l.fnt[:])
			}
		case "ratio":
			if l.ra[0] == nil {
				l.ra[0] = readLifeBarRatio("p1.", is, l.sff, l.at)
			}
			if l.ra[1] == nil {
				l.ra[1] = readLifeBarRatio("p2.", is, l.sff, l.at)
			}
		case "timer":
			if l.tr == nil {
				l.tr = readLifeBarTimer(is, l.sff, l.at, l.fnt[:])
			}
		case "score":
			if l.sc[0] == nil {
				l.sc[0] = readLifeBarScore("p1.", is, l.sff, l.at, l.fnt[:])
			}
			if l.sc[1] == nil {
				l.sc[1] = readLifeBarScore("p2.", is, l.sff, l.at, l.fnt[:])
			}
		case "match":
			if l.ma == nil {
				l.ma = readLifeBarMatch(is, l.sff, l.at, l.fnt[:])
			}
		case "ailevel":
			if l.ai[0] == nil {
				l.ai[0] = readLifeBarAiLevel("p1.", is, l.sff, l.at, l.fnt[:])
			}
			if l.ai[1] == nil {
				l.ai[1] = readLifeBarAiLevel("p2.", is, l.sff, l.at, l.fnt[:])
			}
		case "wincount":
			if l.wc[0] == nil {
				l.wc[0] = readLifeBarWinCount("p1.", is, l.sff, l.at, l.fnt[:])
			}
			if l.wc[1] == nil {
				l.wc[1] = readLifeBarWinCount("p2.", is, l.sff, l.at, l.fnt[:])
			}
		case "mode":
			if l.mo == nil {
				l.mo = readLifeBarMode(is, l.sff, l.at, l.fnt[:])
			}
		}
	}
	sys.ffx["f"] = ffx
	// fightfx scale
	//if math.IsNaN(float64(sys.ffx["f"].fx_scale)) {
	//	sys.ffx["f"].fx_scale = float32(sys.lifebarLocalcoord[0]) / 320
	//}
	for _, a := range sys.ffx["f"].fat {
		a.start_scale = [...]float32{sys.lifebarScale * sys.ffx["f"].fx_scale,
			sys.lifebarScale * sys.ffx["f"].fx_scale}
	}
	// Iterate over map in a stable iteration order
	keys := make([]string, 0, len(l.missing))
	for k := range l.missing {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if strings.Contains(k, " lifebar") {
			for i := 3; i < len(l.hb); i++ {
				if i == l.missing[k] {
					for j := 0; j < len(l.hb[i]); j++ {
						if i == 6 || i == 7 {
							l.hb[i][j] = l.hb[3][j]
						} else {
							l.hb[i][j] = l.hb[1][j]
						}
					}
				}
			}
		} else if strings.Contains(k, " powerbar") {
			for i := 1; i < len(l.pb); i++ {
				if i == l.missing[k] {
					for j := 0; j < 2; j++ {
						switch i {
						case 4, 5:
							l.pb[i][j] = l.pb[1][j]
						case 6, 7:
							l.pb[i][j] = l.pb[3][j]
						default:
							l.pb[i][j] = l.pb[0][j]
						}
					}
				}
			}
		} else if strings.Contains(k, " guardbar") {
			for i := 1; i < len(l.gb); i++ {
				if i == l.missing[k] {
					for j := 0; j < 2; j++ {
						switch i {
						case 4, 5:
							l.gb[i][j] = l.gb[1][j]
						case 6, 7:
							l.gb[i][j] = l.gb[3][j]
						default:
							l.gb[i][j] = l.gb[0][j]
						}
					}
				}
			}
		} else if strings.Contains(k, " stunbar") {
			for i := 1; i < len(l.sb); i++ {
				if i == l.missing[k] {
					for j := 0; j < 2; j++ {
						switch i {
						case 4, 5:
							l.sb[i][j] = l.sb[1][j]
						case 6, 7:
							l.sb[i][j] = l.sb[3][j]
						default:
							l.sb[i][j] = l.sb[0][j]
						}
					}
				}
			}
		} else if strings.Contains(k, " face") {
			for i := 3; i < len(l.fa); i++ {
				if i == l.missing[k] {
					for j := 0; j < len(l.fa[i]); j++ {
						if i == 6 || i == 7 {
							l.fa[i][j] = l.fa[3][j]
						} else {
							l.fa[i][j] = l.fa[1][j]
						}
					}
				}
			}
		} else if strings.Contains(k, " name") {
			for i := 3; i < len(l.nm); i++ {
				if i == l.missing[k] {
					for j := 0; j < len(l.nm[i]); j++ {
						if i == 6 || i == 7 {
							l.nm[i][j] = l.nm[3][j]
						} else {
							l.nm[i][j] = l.nm[1][j]
						}
					}
				}
			}
		}
	}
	l.def = def
	return l, nil
}

func (l *Lifebar) reloadLifebar() error {
	lb, err := loadLifebar(l.def)
	if err != nil {
		return err
	}
	lb.ti.framespercount = l.ti.framespercount
	lb.ro.match_maxdrawgames = l.ro.match_maxdrawgames
	lb.ro.match_wins = l.ro.match_wins
	lb.tr.active = l.tr.active
	lb.sc[0].active = l.sc[0].active
	lb.sc[1].active = l.sc[1].active
	lb.ma.active = l.ma.active
	lb.ai[0].active = l.ai[0].active
	lb.ai[1].active = l.ai[1].active
	lb.wc[0].active = l.wc[0].active
	lb.wc[1].active = l.wc[1].active
	lb.active = l.active
	lb.bars = l.bars
	lb.mode = l.mode
	lb.redlifebar = l.redlifebar
	lb.guardbar = l.guardbar
	lb.stunbar = l.stunbar
	// lb.fx_scale = l.fx_scale
	sys.lifebar = *lb
	return nil
}

func (l *Lifebar) step() {
	if sys.paused && !sys.step {
		return
	}
	for ti, tm := range sys.tmode {
		if tm == TM_Tag {
			for i, v := range l.order[ti] {
				if sys.teamLeader[sys.chars[v][0].teamside] == sys.chars[v][0].playerNo && sys.chars[v][0].alive() {
					if i != 0 {
						if i == len(l.order[ti])-1 {
							l.order[ti] = sliceMoveInt(l.order[ti], i, 0)
						} else {
							last := len(l.order[ti]) - 1
							for n := last; n > 0; n-- {
								if !sys.chars[l.order[ti][n]][0].alive() {
									last -= 1
								}
							}
							l.order[ti] = sliceMoveInt(l.order[ti], 0, last)
						}
					}
					break
				}
			}
		}
	}
	for ti := range sys.tmode {
		for i, v := range l.order[ti] {
			// HealthBar
			l.hb[l.ref[ti]][i*2+ti].step(v, l.hb[l.ref[ti]][v])
			// PowerBar
			l.pb[l.ref[ti]][i*2+ti].step(v, l.pb[l.ref[ti]][v], l.snd)
			// GuardBar
			l.gb[l.ref[ti]][i*2+ti].step(v, l.gb[l.ref[ti]][v], l.snd)
			// StunBar
			l.sb[l.ref[ti]][i*2+ti].step(v, l.sb[l.ref[ti]][v], l.snd)
			// LifeBarFace
			l.fa[l.ref[ti]][i*2+ti].step(v, l.fa[l.ref[ti]][v])
			// LifeBarName
			l.nm[l.ref[ti]][i*2+ti].step()
		}
	}
	// LifeBarWinIcon
	for i := range l.wi {
		l.wi[i].step(sys.wins[i])
	}
	// LifeBarTime
	l.ti.step()
	// LifeBarCombo
	cb, cd, cp, dz := [2]int32{}, [2]int32{}, [2]float32{}, [2]bool{}
	targets := [2]int32{}
	for _, ch := range sys.chars { // Iterate through all players to see the combo status of each team
		for _, c := range ch {
			if c.receivedHits > 0 && (c.teamside == 0 || c.teamside == 1) && (c.alive() || !c.scf(SCF_over_ko)) { // If alive or not alive but not in state 5150 yet
				side := 1 - c.teamside
				cb[side] += c.receivedHits
				cd[side] += c.receivedDmg
				// Perhaps helper percentages shouldn't be tracked, but ignoring them creates scenarios where the lifebars show 0% damage which looks wrong
				cp[side] += float32(c.receivedDmg) / float32(c.lifeMax) * 100
				targets[side]++
				if c.scf(SCF_dizzy) {
					dz[side] = true
				}
			}
		}
	}
	for side := 0; side < 2; side++ {
		if targets[side] > 0 {
			cp[side] /= float32(targets[side]) // Divide damage percentage by number of valid enemies
		}
	}
	for i := range l.co {
		l.co[i].step(cb[i], cd[i], cp[i], dz[i]) // Combo hits, combo damage, combo damage percentage, dizzy flag
	}
	// LifeBarAction
	for i := range l.ac {
		l.ac[i].step(l.order[i][0])
	}
	// LifeBarRatio
	for ti, tm := range sys.tmode {
		if tm == TM_Turns {
			rl := sys.chars[ti][0].ocd().ratioLevel
			if rl > 0 {
				l.ra[ti].step(rl - 1)
			}
		}
	}
	// LifeBarTimer
	l.tr.step()
	// LifeBarScore
	for i := range l.sc {
		l.sc[i].step()
	}
	// LifeBarMatch
	l.ma.step()
	// LifeBarAiLevel
	for i := range l.ai {
		l.ai[i].step()
	}
	// LifeBarWinCount
	for i := range l.wc {
		l.wc[i].step()
	}
	// LifeBarMode
	if _, ok := l.mo[sys.gameMode]; ok {
		l.mo[sys.gameMode].step()
	}
	// Text sctrl
	for i := 0; i < len(l.textsprite); i++ {
		if l.textsprite[i].removetime == 0 {
			l.textsprite = append(l.textsprite[:i], l.textsprite[i+1:]...)
			i-- // -1 as the slice just got shorter
		} else {
			l.textsprite[i].Draw()
			if sys.tickNextFrame() {
				if l.textsprite[i].removetime > 0 {
					l.textsprite[i].removetime--
				}
			}
		}
	}
}

func (l *Lifebar) RemoveText(id, ownerid int32) {
	for i := len(l.textsprite) - 1; i >= 0; i-- {
		if (id == -1 && l.textsprite[i].ownerid == ownerid) ||
			(id != -1 && l.textsprite[i].id == id && l.textsprite[i].ownerid == ownerid) {
			l.textsprite = append(l.textsprite[:i], l.textsprite[i+1:]...)
		}
	}
}

func (l *Lifebar) reset() {
	var num [2]int
	for ti, tm := range sys.tmode {
		l.ref[ti] = int(tm)
		if tm == TM_Simul {
			if sys.numSimul[ti] == 3 {
				l.ref[ti] = 4 // Simul_3P (6)
			} else if sys.numSimul[ti] >= 4 {
				l.ref[ti] = 5 // Simul_4P (8)
			} else {
				l.ref[ti] = 1 // Simul (8)
			}
		} else if tm == TM_Tag {
			if sys.numSimul[ti] == 3 {
				l.ref[ti] = 6 // Tag_3P (6)
			} else if sys.numSimul[ti] >= 4 {
				l.ref[ti] = 7 // Tag_4P (8)
			} else {
				l.ref[ti] = 3 // Tag (8)
			}
		} else if tm == TM_Turns {
			l.ref[ti] = 2 // Turns (2)
		} else {
			l.ref[ti] = 0 // Single (2)
		}
		if tm == TM_Simul || tm == TM_Tag {
			num[ti] = int(math.Min(8, float64(sys.numSimul[ti])*2))
		} else {
			num[ti] = len(l.hb[l.ref[ti]])
		}
		l.order[ti] = []int{}
		for i := ti; i < num[ti]; i += 2 {
			l.order[ti] = append(l.order[ti], i)
		}
	}
	for i := range l.hb {
		for j := range l.hb[i] {
			l.hb[i][j].reset()
		}
	}
	for i := range l.pb {
		for j := range l.pb[i] {
			l.pb[i][j].reset()
		}
	}
	for i := range l.gb {
		for j := range l.gb[i] {
			l.gb[i][j].reset()
		}
	}
	for i := range l.sb {
		for j := range l.sb[i] {
			l.sb[i][j].reset()
		}
	}
	for i := range l.fa {
		for j := range l.fa[i] {
			l.fa[i][j].reset()
		}
	}
	for i := range l.nm {
		for j := range l.nm[i] {
			l.nm[i][j].reset()
		}
	}
	for i := range l.wi {
		l.wi[i].reset()
	}
	l.ti.reset()
	for i := range l.co {
		l.co[i].reset()
	}
	for i := range l.ac {
		l.ac[i].reset(l.order[i][0])
	}
	l.ro.reset()
	for i := range l.ra {
		l.ra[i].reset()
	}
	l.tr.reset()
	for i := range l.sc {
		l.sc[i].reset()
	}
	l.ma.reset()
	for i := range l.ai {
		l.ai[i].reset()
	}
	for i := range l.wc {
		l.wc[i].reset()
	}
	if _, ok := l.mo[sys.gameMode]; ok {
		l.mo[sys.gameMode].reset()
	}
	l.textsprite = []*TextSprite{}
}

func (l *Lifebar) draw(layerno int16) {
	if sys.postMatchFlg || sys.dialogueBarsFlg {
		return
	}
	if sys.lifebarDisplay && l.active {
		if !sys.gsf(GSF_nobardisplay) && l.bars {
			// HealthBar
			for ti := range sys.tmode {
				for i, v := range l.order[ti] {
					index := i*2 + ti
					if !sys.chars[v][0].asf(ASF_nolifebardisplay) {
						l.hb[l.ref[ti]][index].bgDraw(layerno)
						l.hb[l.ref[ti]][index].draw(layerno, v, l.hb[l.ref[ti]][v], l.fnt[:])
					}
				}
			}
			// PowerBar
			for ti, tm := range sys.tmode {
				for i, v := range l.order[ti] {
					index := i*2 + ti
					if sys.cfg.Options.Team.PowerShare && (tm == TM_Simul || tm == TM_Tag) { // Draw player 1 or 2 bars
						if i == 0 && !sys.chars[v][0].asf(ASF_nopowerbardisplay) {
							l.pb[l.ref[ti]][index].bgDraw(layerno, index)
							l.pb[l.ref[ti]][index].draw(layerno, index, l.pb[l.ref[ti]][index], l.fnt[:])
						}
					} else { // Draw everyone's bars
						if !sys.chars[v][0].asf(ASF_nopowerbardisplay) {
							l.pb[l.ref[ti]][index].bgDraw(layerno, index)
							l.pb[l.ref[ti]][index].draw(layerno, v, l.pb[l.ref[ti]][v], l.fnt[:])
						}
					}
				}
			}
			// GuardBar
			for ti := range sys.tmode {
				for i, v := range l.order[ti] {
					index := i*2 + ti
					if !sys.chars[v][0].asf(ASF_noguardbardisplay) {
						l.gb[l.ref[ti]][index].bgDraw(layerno)
						l.gb[l.ref[ti]][index].draw(layerno, v, l.gb[l.ref[ti]][v], l.fnt[:])
					}
				}
			}
			// StunBar
			for ti := range sys.tmode {
				for i, v := range l.order[ti] {
					index := i*2 + ti
					if !sys.chars[v][0].asf(ASF_nostunbardisplay) {
						l.sb[l.ref[ti]][index].bgDraw(layerno)
						l.sb[l.ref[ti]][index].draw(layerno, v, l.sb[l.ref[ti]][v], l.fnt[:])
					}
				}
			}
			// LifeBarFace
			for ti := range sys.tmode {
				for i, v := range l.order[ti] {
					index := i*2 + ti
					if !sys.chars[v][0].asf(ASF_nofacedisplay) {
						l.fa[l.ref[ti]][index].bgDraw(layerno)
						l.fa[l.ref[ti]][index].draw(layerno, v, l.fa[l.ref[ti]][v])
					}
				}
			}
			// LifeBarName
			for ti := range sys.tmode {
				for i, v := range l.order[ti] {
					index := i*2 + ti
					if !sys.chars[v][0].asf(ASF_nonamedisplay) {
						l.nm[l.ref[ti]][index].bgDraw(layerno)
						l.nm[l.ref[ti]][index].draw(layerno, v, l.fnt[:], ti)
					}
				}
			}
			// LifeBarTime
			l.ti.bgDraw(layerno)
			l.ti.draw(layerno, l.fnt[:])
			// LifeBarWinIcon
			for i := range l.wi {
				if !sys.chars[i][0].asf(ASF_nowinicondisplay) {
					l.wi[i].draw(layerno, l.fnt[:], i)
				}
			}
			// LifeBarRatio
			for ti, tm := range sys.tmode {
				if tm == TM_Turns {
					if rl := sys.chars[ti][0].ocd().ratioLevel; rl > 0 && !sys.chars[ti][0].asf(ASF_nofacedisplay) {
						l.ra[ti].bgDraw(layerno)
						l.ra[ti].draw(layerno, rl-1)
					}
				}
			}
			// LifeBarTimer
			l.tr.bgDraw(layerno)
			l.tr.draw(layerno, l.fnt[:])
			// LifeBarScore
			for i := range l.sc {
				l.sc[i].bgDraw(layerno)
				l.sc[i].draw(layerno, l.fnt[:], i)
			}
			// LifeBarMatch
			l.ma.bgDraw(layerno)
			l.ma.draw(layerno, l.fnt[:])
			// LifeBarAiLevel
			for i := range l.ai {
				l.ai[i].bgDraw(layerno)
				l.ai[i].draw(layerno, l.fnt[:], sys.aiLevel[sys.chars[i][0].playerNo])
			}
			// LifeBarWinCount
			for i := range l.wc {
				l.wc[i].bgDraw(layerno)
				l.wc[i].draw(layerno, l.fnt[:], i)
			}
		}
		// LifeBarCombo
		for i := range l.co {
			if !sys.chars[i][0].asf(ASF_nocombodisplay) {
				l.co[i].draw(layerno, l.fnt[:], i)
			}
		}
		// LifeBarAction
		for i := range l.ac {
			if !sys.chars[i][0].asf(ASF_nolifebaraction) {
				l.ac[i].draw(layerno, l.fnt[:], i)
			}
		}
		// LifeBarMode
		if _, ok := l.mo[sys.gameMode]; ok {
			l.mo[sys.gameMode].bgDraw(layerno)
			l.mo[sys.gameMode].draw(layerno, l.fnt[:])
		}
	}
	if l.active {
		// LifeBarRound
		l.ro.draw(layerno, l.fnt[:])
	}
	// Text sctrl
	for _, v := range l.textsprite {
		if v.layerno == layerno {
			v.Draw()
		}
	}
	BlendReset()
}
