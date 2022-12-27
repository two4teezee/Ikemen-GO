package main

type Copyable[T any] interface {
	Clone() T
}

func CopySlice[T any](src, dst *[]T) {
	*(dst) = (*dst)[:0]
	for i := 0; i < len(*src); i++ {
		*(dst) = append(*(dst), (*src)[i])
	}
}

func CopyMap[T comparable, E any](src, dst *map[T]E) {
	for k := range *src {
		(*dst)[k] = (*src)[k]
	}

	for k := range *dst {
		if _, ok := (*src)[k]; !ok {
			delete(*dst, k)
		}
	}
}

// func Copy2DSlice[T any](src, dst *[][]T) {
// 	if len(*dst) < len(*src) {
// 		i := 0
// 		for ; i < len(*dst); i++ {
// 			CopySlice(&(*src)[i], &(*dst)[i])
// 		}
// 		for ; i < len(*src); i++ {
// 			slice := PoolGet((*src)[i]).(*[]T)
// 			*dst = append(*dst, *slice)
// 			CopySlice(&(*src)[i], &(*dst)[i])
// 		}
// 	} else {
// 		(*dst) = (*dst)[0:len(*src)]
// 		for i := 0; i < len(*src); i++ {
// 			CopySlice(&(*src)[i], &(*dst)[i])
// 		}
// 	}
// }

// func DeepCopy2DSlice[T Copyable[T]](src, dst *[][]T) {
// 	if len(*dst) < len(*src) {
// 		i := 0
// 		for ; i < len(*dst); i++ {
// 			DeepCopySlice(&(*src)[i], &(*dst)[i])
// 		}
// 		for ; i < len(*src); i++ {
// 			slice := PoolGet((*src)[i]).(*[]T)
// 			*dst = append(*dst, *slice)
// 			DeepCopySlice(&(*src)[i], &(*dst)[i])
// 		}
// 	} else {
// 		(*dst) = (*dst)[0:len(*src)]
// 		for i := 0; i < len(*src); i++ {
// 			DeepCopySlice(&(*src)[i], &(*dst)[i])
// 		}
// 	}
// }

func DeepCopySlice[T Copyable[T]](src, dst *[]T) {
	if len(*dst) >= len(*src) {
		*(dst) = (*dst)[0:len(*src)]
		for i := 0; i < len(*src); i++ {
			(*dst)[i] = (*src)[i].Clone()
		}
	} else {
		i := 0
		for ; i < len(*dst); i++ {
			(*dst)[i] = (*src)[i].Clone()
		}
		for ; i < len(*src); i++ {
			(*dst) = append(*dst, (*src)[i].Clone())
		}
	}
}

func (a Animation) Clone() (result Animation) {
	result = a

	result.frames = make([]AnimFrame, len(a.frames))
	for i := 0; i < len(a.frames); i++ {
		result.frames[i] = *a.frames[i].Clone()
	}

	result.interpolate_offset = make([]int32, len(a.interpolate_offset))
	copy(result.interpolate_offset, a.interpolate_offset)

	result.interpolate_scale = make([]int32, len(a.interpolate_scale))
	copy(result.interpolate_scale, a.interpolate_scale)

	result.interpolate_angle = make([]int32, len(a.interpolate_angle))
	copy(result.interpolate_angle, a.interpolate_angle)

	result.interpolate_blend = make([]int32, len(a.interpolate_blend))
	copy(result.interpolate_blend, a.interpolate_blend)

	return
}

func (af *AnimFrame) Clone() (result *AnimFrame) {
	result = &AnimFrame{}
	*result = *af
	result.Ex = make([][]float32, len(af.Ex))
	for i := 0; i < len(af.Ex); i++ {
		result.Ex[i] = make([]float32, len(af.Ex[i]))
		copy(result.Ex[i], af.Ex[i])
	}
	return
}

func (sp StringPool) Clone() (result StringPool) {
	result = sp
	result.List = make([]string, len(sp.List))
	copy(result.List, sp.List)
	result.Map = make(map[string]int)
	for k, v := range sp.Map {
		result.Map[k] = v
	}
	return
}

func (b *StateBlock) Clone() (result StateBlock) {
	result = *b
	result.trigger = make(BytecodeExp, len(b.trigger))
	copy(result.trigger, b.trigger)
	if b.elseBlock != nil {
		eb := b.elseBlock.Clone()
		result.elseBlock = &eb
	}
	result.ctrls = make([]StateController, len(b.ctrls))
	copy(result.ctrls, b.ctrls)
	return result
}

func (sb *StateBytecode) Clone() (result StateBytecode) {
	result = *sb
	result.stateDef = make(stateDef, len(sb.stateDef))
	copy(result.stateDef, sb.stateDef)

	result.ctrlsps = make([]int32, len(sb.ctrlsps))
	copy(result.ctrlsps, sb.ctrlsps)
	result.block = sb.block.Clone()
	return result
}

func (ghv GetHitVar) Clone() (result GetHitVar) {
	result = ghv

	// Manually copy references that shallow copy poorly, as needed
	// Pointers, slices, maps, functions, channels etc
	result.hitBy = make([][2]int32, len(ghv.hitBy))
	copy(result.hitBy, ghv.hitBy)

	return
}

func (ai AfterImage) Clone() (result AfterImage) {
	result = ai
	result.palfx = make([]PalFX, len(ai.palfx))
	for i := 0; i < len(ai.palfx); i++ {
		result.palfx[i] = ai.palfx[i].Clone()
	}
	return
}

func (e Explod) Clone() (result Explod) {
	result = e
	if e.anim != nil {
		anim := e.anim.Clone()
		result.anim = &anim
	}
	palfx := e.palfx.Clone()
	result.palfx = &palfx
	return
}

func (p Projectile) Clone() (result Projectile) {
	result = p
	if p.ani != nil {
		*result.ani = p.ani.Clone()
	}
	result.aimg.palfx = make([]PalFX, len(p.aimg.palfx))
	for i := 0; i < len(p.aimg.palfx); i++ {
		result.aimg.palfx[i] = p.aimg.palfx[i].Clone()
	}

	palfx := p.palfx.Clone()
	result.palfx = &palfx
	return
}

func (ss *StateState) Clone() (result StateState) {
	result = *ss
	result.ps = make([]int32, len(ss.ps))
	copy(result.ps, ss.ps)
	for i := 0; i < len(ss.wakegawakaranai); i++ {
		result.wakegawakaranai[i] = make([]bool, len(ss.wakegawakaranai[i]))
		copy(result.wakegawakaranai[i], ss.wakegawakaranai[i])
	}
	result.sb = ss.sb.Clone()
	return result
}

func (c *Char) Clone() (result Char) {
	result = Char{}
	result = *c

	result.aimg = c.aimg.Clone()

	result.nextHitScale = make(map[int32][3]*HitScale)
	for i, v := range c.nextHitScale {
		hitScale := [3]*HitScale{}
		for i := 0; i < len(v); i++ {
			if v[i] != nil {
				*hitScale[i] = *v[i]
			}
		}
		result.nextHitScale[i] = hitScale
	}

	result.activeHitScale = make(map[int32][3]*HitScale)
	for i, v := range c.activeHitScale {
		hitScale := [3]*HitScale{}
		for i := 0; i < len(v); i++ {
			if v[i] != nil {
				// *hitScale[i] = *v[i] causes bugs
				hitScale[i] = v[i]
				*hitScale[i] = *v[i]
			}
		}
		result.activeHitScale[i] = hitScale
	}

	// todo, find the curFrame index and set result.curFrame as the pointer at
	// that index
	if c.anim != nil {
		anim := c.anim.Clone()
		result.anim = &anim
	}
	if c.curFrame != nil {
		result.curFrame = c.curFrame.Clone()
	}

	// Manually copy references that shallow copy poorly, as needed
	// Pointers, slices, maps, functions, channels etc
	result.ghv = c.ghv.Clone()

	result.children = make([]*Char, len(c.children))
	copy(result.children, c.children)
	// for i := 0; i < len(c.children); i++ {
	// 	//result.children[i] = c.children[i].Clone()
	// }

	result.targets = make([]int32, len(c.targets))
	copy(result.targets, c.targets)

	result.targetsOfHitdef = make([]int32, len(c.targetsOfHitdef))
	copy(result.targetsOfHitdef, c.targetsOfHitdef)

	for i := range c.enemynear {
		result.enemynear[i] = make([]*Char, len(c.enemynear[i]))
		copy(result.enemynear[i], c.enemynear[i])
		//for j := 0; j < len(c.enemynear[i]); j++ {
		//	result.enemynear[i][j] = c.enemynear[i][j].Clone()
		//}
	}

	result.clipboardText = make([]string, len(c.clipboardText))
	copy(result.clipboardText, c.clipboardText)

	result.cmd = make([]CommandList, len(c.cmd))
	for i, c := range c.cmd {
		result.cmd[i] = c.Clone()
	}
	for i := range result.cmd {
		result.cmd[i].Buffer = result.cmd[0].Buffer
	}

	result.ss = c.ss.Clone()

	result.mapArray = make(map[string]float32)
	for k, v := range c.mapArray {
		result.mapArray[k] = v
	}
	return
}

func (pf PalFX) Clone() (result PalFX) {
	result = pf
	result.remap = make([]int, len(pf.remap))
	copy(result.remap, pf.remap)
	return
}

func (ce *cmdElem) Clone() (result cmdElem) {
	result = *ce
	result.key = make([]CommandKey, len(ce.key))
	copy(result.key, ce.key)
	return
}

func (c *Command) Clone() (result Command) {
	result = *c

	result.cmd = make([]cmdElem, len(c.cmd))
	for i := 0; i < len(c.cmd); i++ {
		result.cmd[i] = c.cmd[i].Clone()
	}

	result.held = make([]bool, len(c.held))
	copy(result.held, c.held)

	result.hold = make([][]CommandKey, len(c.hold))
	for i := 0; i < len(c.hold); i++ {
		result.hold[i] = make([]CommandKey, len(c.hold[i]))
		for j := 0; j < len(c.hold[i]); j++ {
			result.hold[i][j] = c.hold[i][j]
		}
	}
	return
}

func (cl *CommandList) Clone() (result CommandList) {
	result = *cl
	result.Buffer = &CommandBuffer{}
	*result.Buffer = *cl.Buffer
	result.Commands = make([][]Command, len(cl.Commands))
	for i := 0; i < len(cl.Commands); i++ {
		result.Commands[i] = make([]Command, len(cl.Commands[i]))
		for j := 0; j < len(cl.Commands[i]); j++ {
			result.Commands[i][j] = cl.Commands[i][j].Clone()
		}
	}
	result.Names = make(map[string]int)
	for k, v := range cl.Names {
		result.Names[k] = v
	}
	return
}

func (l *Lifebar) Clone() (result Lifebar) {
	result = *l

	if l.ro != nil {
		round := *l.ro
		result.ro = &round
	}

	//UIT
	for i := 0; i < len(l.sc); i++ {
		if l.sc[i] != nil {
			score := *l.sc[i]
			result.sc[i] = &score
		}
	}
	if l.ti != nil {
		time := *l.ti
		result.ti = &time
	}
	for i := 0; i < len(l.co); i++ {
		if l.co[i] != nil {
			combo := *l.co[i]
			result.co[i] = &combo
		}
	}
	//

	// Not UIT adding amyway
	for i := 0; i < len(l.wc); i++ {
		wins := *l.wc[i]
		result.wc[i] = &wins
	}

	if l.ma != nil {
		match := *l.ma
		result.ma = &match
	}

	for i := 0; i < len(l.ai); i++ {
		ai := *l.ai[i]
		result.ai[i] = &ai
	}

	if l.tr != nil {
		timer := *l.tr
		result.tr = &timer
	}
	//

	for i := range result.order {
		result.order[i] = make([]int, len(l.order[i]))
		copy(result.order[i], l.order[i])
	}

	for i := range result.hb {
		result.hb[i] = make([]*HealthBar, len(l.hb[i]))
		for j := 0; j < len(l.hb[i]); j++ {
			health := *l.hb[i][j]
			result.hb[i][j] = &health
		}
	}

	for i := range result.pb {
		result.pb[i] = make([]*PowerBar, len(l.pb[i]))
		for j := 0; j < len(l.pb[i]); j++ {
			power := *l.pb[i][j]
			result.pb[i][j] = &power
		}
	}

	for i := range result.gb {
		result.gb[i] = make([]*GuardBar, len(l.gb[i]))
		for j := 0; j < len(l.gb[i]); j++ {
			gaurd := *l.gb[i][j]
			result.gb[i][j] = &gaurd
		}
	}

	for i := range result.sb {
		result.sb[i] = make([]*StunBar, len(l.sb[i]))
		for j := 0; j < len(l.sb[i]); j++ {
			stun := *l.sb[i][j]
			result.sb[i][j] = &stun
		}
	}

	for i := range result.fa {
		result.fa[i] = make([]*LifeBarFace, len(l.fa[i]))
		for j := 0; j < len(l.fa[i]); j++ {
			face := *l.fa[i][j]
			result.fa[i][j] = &face
		}
	}

	for i := range result.nm {
		result.nm[i] = make([]*LifeBarName, len(l.nm[i]))
		for j := 0; j < len(l.nm[i]); j++ {
			name := *l.nm[i][j]
			result.nm[i][j] = &name
		}
	}

	return
}

func (bg backGround) Clone() (result backGround) {
	result = bg
	result.anim = bg.anim.Clone()
	return
}

func (bgc bgCtrl) Clone() (result bgCtrl) {
	result = bgc
	result.bg = make([]*backGround, len(bgc.bg))
	for i := 0; i < len(bgc.bg); i++ {
		bg := bgc.bg[i].Clone()
		result.bg[i] = &bg
	}
	return
}

func (bgctn bgctNode) Clone() (result bgctNode) {
	result = bgctNode{}
	result = bgctn
	result.bgc = make([]*bgCtrl, len(bgctn.bgc))
	for i := 0; i < len(bgctn.bgc); i++ {
		bgc := bgctn.bgc[i].Clone()
		result.bgc[i] = &bgc
	}
	return
}

func (bgct bgcTimeLine) Clone() (result bgcTimeLine) {
	result = bgct
	result.line = make([]bgctNode, len(bgct.line))
	for i := 0; i < len(bgct.line); i++ {
		result.line[i] = bgct.line[i].Clone()
	}
	result.al = make([]*bgCtrl, len(bgct.al))
	for i := 0; i < len(bgct.al); i++ {
		bgCtrl := bgct.al[i].Clone()
		result.al[i] = &bgCtrl
	}
	return
}

func (s Stage) Clone() (result Stage) {
	result = s

	result.attachedchardef = make([]string, len(s.attachedchardef))
	copy(result.attachedchardef, s.attachedchardef)

	result.constants = make(map[string]float32)
	for k, v := range s.constants {
		result.constants[k] = v
	}

	result.at = make(AnimationTable)
	for k, v := range s.at {
		anim := v.Clone()
		result.at[k] = &anim
	}

	result.bg = make([]*backGround, len(s.bg))
	for i := 0; i < len(s.bg); i++ {
		bg := s.bg[i].Clone()
		result.bg[i] = &bg
	}

	result.bgc = make([]bgCtrl, len(s.bgc))
	for i := 0; i < len(s.bgc); i++ {
		result.bgc[i] = s.bgc[i].Clone()
	}

	result.bgct = s.bgct.Clone()
	return
}

// other things can be copied, only focusing on OCD right now
func (s Select) Clone() (result Select) {
	result = s
	for i := 0; i < len(s.ocd); i++ {
		result.ocd[i] = make([]OverrideCharData, len(s.ocd[i]))
		copy(result.ocd[i], s.ocd[i])
	}

	result.stageAnimPreload = make([]int32, len(s.stageAnimPreload))
	copy(result.stageAnimPreload, s.stageAnimPreload)

	return
}

func (f Fight) Clone() (result Fight) {
	result = f
	result.oldStageVars = f.oldStageVars.Clone()
	result.level = make([]int32, len(f.level))
	copy(result.level, f.level)

	result.life = make([]int32, len(f.life))
	copy(result.life, f.life)
	result.pow = make([]int32, len(f.pow))
	copy(result.pow, f.pow)
	result.gpow = make([]int32, len(f.gpow))
	copy(result.gpow, f.gpow)
	result.spow = make([]int32, len(f.spow))
	copy(result.spow, f.spow)
	result.rlife = make([]int32, len(f.rlife))
	copy(result.rlife, f.rlife)

	result.ivar = make([][]int32, len(f.ivar))
	for i := 0; i < len(f.ivar); i++ {
		result.ivar[i] = make([]int32, len(f.ivar[i]))
		copy(result.ivar[i], f.ivar[i])
	}

	result.fvar = make([][]float32, len(f.fvar))
	for i := 0; i < len(f.ivar); i++ {
		result.fvar[i] = make([]float32, len(f.fvar[i]))
		copy(result.fvar[i], f.fvar[i])
	}

	result.dialogue = make([][]string, len(f.dialogue))
	for i := 0; i < len(result.dialogue); i++ {
		result.dialogue[i] = make([]string, len(f.dialogue[i]))
		copy(result.dialogue[i], f.dialogue[i])
	}

	result.mapArray = make([]map[string]float32, len(f.mapArray))
	for i := 0; i < len(f.mapArray); i++ {
		result.mapArray[i] = make(map[string]float32)
		for k, v := range f.mapArray[i] {
			result.mapArray[i][k] = v
		}
	}
	return
}
