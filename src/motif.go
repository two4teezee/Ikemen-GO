package main

import (
	_ "embed" // Support for go:embed resources
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"gopkg.in/ini.v1"
)

//go:embed resources/defaultMotif.ini
var defaultMotif []byte

type PalFxProperties struct {
	Time        int32    `ini:"time" default:"-1"`
	Color       float32  `ini:"color" default:"256"`
	Hue         float32  `ini:"hue"`
	Add         [3]int32 `ini:"add"`
	Mul         [3]int32 `ini:"mul" default:"256,256,256"`
	SinAdd      [4]int32 `ini:"sinadd"`
	SinMul      [4]int32 `ini:"sinmul"`
	SinColor    [2]int32 `ini:"sincolor"`
	SinHue      [2]int32 `ini:"sinhue"`
	InvertAll   bool     `ini:"invertall"`
	InvertBlend int32    `ini:"invertblend"`
	PalFxData   *PalFX
}

type InfoProperties struct {
	Name         string   `ini:"name"`
	Author       string   `ini:"author"`
	VersionDate  string   `ini:"versiondate"`
	MugenVersion string   `ini:"mugenversion"`
	Localcoord   [2]int32 `ini:"localcoord" default:"320,240"`
}

type FontProperties struct {
	Font    string    `ini:"" lua:"font" lookup:"def,font/,,data/"`
	Height  int32     `ini:"height" default:"-1"`
	Type    string    `ini:"type"`
	Size    [2]uint16 `ini:"size"`
	Spacing [2]int32  `ini:"spacing"`
	Offset  [2]int32  `ini:"offset"`
}

type FilesProperties struct {
	Spr     string `ini:"spr" lookup:"def,,data/"`
	Snd     string `ini:"snd" lookup:"def,,data/"`
	Loading struct {
		Storyboard string `ini:"storyboard" lookup:"def,,data/"`
	} `ini:"loading"`
	Logo struct {
		Storyboard string `ini:"storyboard" lookup:"def,,data/"`
	} `ini:"logo"`
	Intro struct {
		Storyboard string `ini:"storyboard" lookup:"def,,data/"`
	} `ini:"intro"`
	Select string                     `ini:"select" default:"select.def" lookup:"def,,data/"`
	Fight  string                     `ini:"fight" default:"fight.def" lookup:"def,,data/"`
	Font   map[string]*FontProperties `ini:"map:^(?i)font[0-9]+$" lua:"font"`
	Glyphs string                     `ini:"glyphs" lookup:"def,,data/"`
	Module string                     `ini:"module" lookup:"def,,data/"`
	Model  string                     `ini:"model" lookup:"def,,data/"`
}

type BgmProperties struct {
	Bgm           []string  `ini:"" lua:"bgm" lookup:"def,,data/,sound/"`
	Loop          []int32   `ini:"loop" default:"1"`
	Volume        []int32   `ini:"volume" default:"100"`
	LoopStart     []int32   `ini:"loopstart"`
	LoopEnd       []int32   `ini:"loopend"`
	StartPosition []int32   `ini:"startposition"`
	FreqMul       []float32 `ini:"freqmul" default:"1"`
	LoopCount     []int32   `ini:"loopcount" default:"-1"`
}

type FadeProperties struct {
	Time       int32    `ini:"time"`
	Col        [3]int32 `ini:"col"`
	Anim       int32    `ini:"anim" default:"-1"`
	Localcoord [2]int32 `ini:"localcoord"`
	AnimData   *Anim
	Snd        [2]int32 `ini:"snd" default:"-1,0"`
	FadeData   *Fade
}

type AnimationProperties struct {
	Anim       int32      `ini:"anim" default:"-1"`
	Spr        [2]int32   `ini:"spr" default:"-1,0"`
	Offset     [2]float32 `ini:"offset"`
	Facing     int32      `ini:"facing" default:"1"`
	Scale      [2]float32 `ini:"scale" default:"1,1"`
	Xshear     float32    `ini:"xshear"`
	Angle      float32    `ini:"angle"`
	Layerno    int16      `ini:"layerno" default:"1"`
	Window     [4]int32   `ini:"window"`
	Localcoord [2]int32   `ini:"localcoord"`
	AnimData   *Anim
}

type TextProperties struct {
	Font           [8]int32   `ini:"font" default:"-1,0,0,255,255,255,255,-1"`
	Offset         [2]float32 `ini:"offset"`
	Scale          [2]float32 `ini:"scale" default:"1,1"`
	Xshear         float32    `ini:"xshear"`
	Angle          float32    `ini:"angle"`
	Text           string     `ini:"text"`
	Layerno        int16      `ini:"layerno" default:"1"`
	Window         [4]int32   `ini:"window"`
	Localcoord     [2]int32   `ini:"localcoord"`
	TextSpriteData *TextSprite
}

type TextMapProperties struct {
	Font           [8]int32          `ini:"font" default:"-1,0,0,255,255,255,255,-1"`
	Offset         [2]float32        `ini:"offset"`
	Scale          [2]float32        `ini:"scale" default:"1,1"`
	Xshear         float32           `ini:"xshear"`
	Angle          float32           `ini:"angle"`
	Text           map[string]string `ini:"text"`
	Layerno        int16             `ini:"layerno" default:"1"`
	Window         [4]int32          `ini:"window"`
	Localcoord     [2]int32          `ini:"localcoord"`
	TextSpriteData *TextSprite
}

type BoxBgProperties struct {
	Visible    bool     `ini:"visible"`
	Col        [3]int32 `ini:"col"`
	Alpha      [2]int32 `ini:"alpha" default:"0,255"`
	Layerno    int16    `ini:"layerno" default:"1"`
	Localcoord [2]int32 `ini:"localcoord"`
	RectData   *Rect
}

type TweenProperties struct {
	Factor [2]float32 `ini:"factor"`
	Snap   int32      `ini:"snap"`
	Wrap   struct {
		Snap int32 `ini:"snap"`
	} `ini:"wrap"`
}

type BoxCursorProperties struct {
	Visible    bool     `ini:"visible"`
	Coords     [4]int32 `ini:"coords"`
	Col        [3]int32 `ini:"col"`
	Layerno    int16    `ini:"layerno" default:"1"`
	Localcoord [2]int32 `ini:"localcoord"`
	Pulse      [3]int32 `ini:"pulse"`
	//Alpha      [2]int32 `ini:"alpha" default:"0,255"`
	//Palfx      PalFxProperties `ini:"palfx"`
	RectData *Rect
	Tween    TweenProperties `ini:"tween"`
}

type MenuProperties struct {
	Pos   [2]float32      `ini:"pos" default:"1000,900"`
	Tween TweenProperties `ini:"tween"`
	Item  struct {
		TextProperties
		Spacing [2]float32                      `ini:"spacing"`
		Tween   TweenProperties                 `ini:"tween"`
		Bg      map[string]*AnimationProperties `ini:"bg"`
		Active  struct {
			TextProperties
			Bg map[string]*AnimationProperties `ini:"bg"`
		} `ini:"active"`
		Selected struct { // not used by [Title Info], [Option Info].keymenu, [Replay Info], [Attract Mode]
			TextProperties
			Active TextProperties `ini:"active"`
		} `ini:"selected"`
		Value struct { // not used by [Title Info], [Replay Info], [Attract Mode]
			TextProperties
			Active   TextProperties `ini:"active"`
			Conflict TextProperties `ini:"conflict"`
		} `ini:"value"`
		Info struct { // not used by [Title Info], [Replay Info], [Menu Info], [Attract Mode]
			TextProperties
			Active TextProperties `ini:"active"`
		} `ini:"info"`
	} `ini:"item"`
	Window struct { // not used by [Option Info].keymenu
		Margins struct {
			Y [2]float32 `ini:"y"`
		} `ini:"margins"`
		VisibleItems int32 `ini:"visibleitems"`
	} `ini:"window"`
	BoxCursor BoxCursorProperties `ini:"boxcursor"`
	BoxBg     BoxBgProperties     `ini:"boxbg"`
	Arrow     struct {            // not used by [Option Info].keymenu
		Up   AnimationProperties `ini:"up"`
		Down AnimationProperties `ini:"down"`
	} `ini:"arrow"`
	Title struct {
		Uppercase bool `ini:"uppercase"`
	} `ini:"title"`
	Next struct {
		Key []string `ini:"key"`
	} `ini:"next"`
	Previous struct {
		Key []string `ini:"key"`
	} `ini:"previous"`
	Done struct {
		Key []string `ini:"key"`
	} `ini:"done"`
	Hiscore struct { // not used by [Option Info], [Option Info].keymenu, [Replay Info], [Menu Info]
		Key []string `ini:"key"`
	} `ini:"hiscore"`
	Unlock    map[string]string `ini:"unlock"`    // not used by [Option Info].keymenu, [Replay Info], [Menu Info]
	Valuename map[string]string `ini:"valuename"` // not used by [Title Info], [Option Info].keymenu, [Replay Info], [Menu Info], [Attract Mode]
	Itemname  map[string]string `ini:"itemname"`
}

type OverlayProperties struct {
	Col        [3]int32 `ini:"col"`
	Alpha      [2]int32 `ini:"alpha" default:"0,255"`
	Layerno    int16    `ini:"layerno" default:"0"`
	Window     [4]int32 `ini:"window"`
	Localcoord [2]int32 `ini:"localcoord"`
	RectData   *Rect
}

type TitleInfoProperties struct {
	FadeIn  FadeProperties `ini:"fadein"`
	FadeOut FadeProperties `ini:"fadeout"`
	Title   TextProperties `ini:"title"`
	Menu    MenuProperties `ini:"menu"`
	Cursor  struct {
		Move struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"move"`
		Done struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"done"`
		Snd map[string][2]int32 `ini:"snd"`
	} `ini:"cursor"`
	Cancel struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"cancel"`
	Loading TextProperties `ini:"loading"`
	Footer  struct {
		Title   TextProperties    `ini:"title"`
		Info    TextProperties    `ini:"info"`
		Version TextProperties    `ini:"version"`
		Overlay OverlayProperties `ini:"overlay"`
	} `ini:"footer"`
	Connecting struct {
		Host    TextProperties    `ini:"host"`
		Join    TextProperties    `ini:"join"`
		Overlay OverlayProperties `ini:"overlay"`
	} `ini:"connecting"`
	TextInput struct {
		TextMapProperties
		Overlay OverlayProperties `ini:"overlay"`
	} `ini:"textinput"`
}

type BgDefProperties struct {
	Sff            *Sff
	BGDef          *BGDef
	Spr            string   `ini:"spr"`
	BgClearColor   [3]int32 `ini:"bgclearcolor" default:"-1,0,0"`
	BgClearAlpha   [2]int32 `ini:"bgclearalpha" default:"255,0"`
	BgClearLayerno int16    `ini:"bgclearlayerno" default:"0"`
	DefaultLayer   int32    `ini:"defaultlayer" default:"0"`
	Localcoord     [2]int32 `ini:"localcoord"`
	RectData       *Rect
}

type InfoBoxProperties struct {
	Title   TextProperties    `ini:"title"`
	Text    TextProperties    `ini:"text"`
	Overlay OverlayProperties `ini:"overlay"`
}

type CellOverrideProperties struct {
	Offset [2]float32 `ini:"offset"`
	Facing int32      `ini:"facing"`
	Skip   bool       `ini:"skip"`
}

type TimerProperties struct {
	TextProperties
	Count          int32 `ini:"count"`
	Framespercount int32 `ini:"framespercount" default:"60"`
	Displaytime    int32 `ini:"displaytime"`
}

type PlayerCursorDoneProperties struct {
	AnimationProperties
	Snd [2]int32 `ini:"snd" default:"-1,0"`
}

type PlayerCursorProperties struct {
	StartCell [2]int32                               `ini:"startcell"`
	Active    map[string]*AnimationProperties        `ini:"active"`
	Done      map[string]*PlayerCursorDoneProperties `ini:"done"`
	Move      struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"move"`
	Blink      bool            `ini:"blink"`      // only used by P2
	SwitchTime int32           `ini:"switchtime"` // only used by P2
	Tween      TweenProperties `ini:"tween"`
	Reset      bool            `ini:"reset"`
}

type ItemProperties struct {
	TextMapProperties
	Spacing [2]float32      `ini:"spacing"`
	Tween   TweenProperties `ini:"tween"`
	Active  struct {
		Font       [8]int32 `ini:"font" default:"-1,0,0,255,255,255,255,-1"`
		Switchtime int32    `ini:"switchtime"`
	} `ini:"active"`
	Active2 struct {
		Font [8]int32 `ini:"font" default:"-1,0,0,255,255,255,255,-1"`
	} `ini:"active2"`
	Cursor    AnimationProperties `ini:"cursor"`    // only used by [Select Info].pX.teammenu.item
	Uppercase bool                `ini:"uppercase"` // only used by [Hiscore Info].item.name
}

type AnimationTextProperties struct {
	Anim           int32      `ini:"anim" default:"-1"`
	Spr            [2]int32   `ini:"spr" default:"-1,0"`
	Offset         [2]float32 `ini:"offset"`
	Facing         int32      `ini:"facing" default:"1"`
	Scale          [2]float32 `ini:"scale" default:"1,1"`
	Xshear         float32    `ini:"xshear"`
	Angle          float32    `ini:"angle"`
	Layerno        int16      `ini:"layerno" default:"1"`
	Window         [4]int32   `ini:"window"`
	Localcoord     [2]int32   `ini:"localcoord"`
	AnimData       *Anim
	Font           [8]int32 `ini:"font" default:"-1,0,0,255,255,255,255,-1"`
	Text           string   `ini:"text"`
	TextSpriteData *TextSprite
}

type AnimationCharPreloadProperties struct {
	Anim       int32      `ini:"anim" default:"-1" preload:"char"`
	Spr        [2]int32   `ini:"spr" default:"-1,0" preload:"char"`
	Offset     [2]float32 `ini:"offset"`
	Facing     int32      `ini:"facing" default:"1"`
	Scale      [2]float32 `ini:"scale" default:"1,1"`
	Xshear     float32    `ini:"xshear"`
	Angle      float32    `ini:"angle"`
	Layerno    int16      `ini:"layerno" default:"1"`
	Window     [4]int32   `ini:"window"`
	Localcoord [2]int32   `ini:"localcoord"`
	AnimData   *Anim
	ApplyPal   bool `ini:"applypal" preload:"pal"`
}

type FaceProperties struct {
	AnimationCharPreloadProperties
	Done struct { // not used by [Victory Screen]
		AnimationCharPreloadProperties
		Key []string `ini:"key"` // only used by [VS Screen]
	} `ini:"done"`
	Random   AnimationProperties `ini:"random"` // only used by [Select Info]
	Velocity [2]float32          `ini:"velocity"`
	Accel    [2]float32          `ini:"accel"`
	Friction [2]float32          `ini:"friction" default:"1,1"`
	Pos      [2]float32          `ini:"pos"`
	Num      int32               `ini:"num"`     // only used by P1-P2
	Spacing  [2]float32          `ini:"spacing"` // only used by P1-P2
	Padding  bool                `ini:"padding"` // only used by P1-P2
}

type PlayerSelectProperties struct {
	Cursor PlayerCursorProperties `ini:"cursor"`
	Random struct {
		Move struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"move"`
	} `ini:"random"`
	Face  FaceProperties `ini:"face"`
	Face2 FaceProperties `ini:"face2"`
	Name  struct {
		TextProperties
		Num     int32      `ini:"num"`     // only used by P1-P2
		Spacing [2]float32 `ini:"spacing"` // only used by P1-P2
		Random  struct {
			Text string `ini:"text"`
		} `ini:"random"`
	} `ini:"name"`
	Swap struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"swap"`
	Select struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"select"`
	TeamMenu struct { // only used by P1-P2
		Pos    [2]float32                      `ini:"pos"`
		Bg     map[string]*AnimationProperties `ini:"bg"`
		Active struct {
			Bg map[string]*AnimationProperties `ini:"bg"`
		} `ini:"active"`
		SelfTitle struct {
			AnimationTextProperties
		} `ini:"selftitle"`
		EnemyTitle struct {
			AnimationTextProperties
		} `ini:"enemytitle"`
		Move struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"move"`
		Value struct {
			Snd   [2]int32            `ini:"snd" default:"-1,0"`
			Icon  AnimationProperties `ini:"icon"`
			Empty struct {
				Icon AnimationProperties `ini:"icon"`
			} `ini:"empty"`
			Spacing [2]float32 `ini:"spacing"`
		} `ini:"value"`
		Done struct {
			Key []string `ini:"key"`
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"done"`
		Next struct {
			Key []string `ini:"key"`
		} `ini:"next"`
		Previous struct {
			Key []string `ini:"key"`
		} `ini:"previous"`
		Add struct {
			Key []string `ini:"key"`
		} `ini:"add"`
		Subtract struct {
			Key []string `ini:"key"`
		} `ini:"subtract"`
		Item   ItemProperties `ini:"item"`
		Ratio1 struct {
			Icon AnimationProperties `ini:"icon"`
		} `ini:"ratio1"`
		Ratio2 struct {
			Icon AnimationProperties `ini:"icon"`
		} `ini:"ratio2"`
		Ratio3 struct {
			Icon AnimationProperties `ini:"icon"`
		} `ini:"ratio3"`
		Ratio4 struct {
			Icon AnimationProperties `ini:"icon"`
		} `ini:"ratio4"`
		Ratio5 struct {
			Icon AnimationProperties `ini:"icon"`
		} `ini:"ratio5"`
		Ratio6 struct {
			Icon AnimationProperties `ini:"icon"`
		} `ini:"ratio6"`
		Ratio7 struct {
			Icon AnimationProperties `ini:"icon"`
		} `ini:"ratio7"`
	} `ini:"teammenu"`
	PalMenu struct {
		Pos  [2]float32 `ini:"pos"`
		Next struct {
			Key []string `ini:"key"`
		} `ini:"next"`
		Previous struct {
			Key []string `ini:"key"`
		} `ini:"previous"`
		Random struct {
			Key  []string `ini:"key"`
			Text string   `ini:"text"`
		} `ini:"random"`
		Value struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"value"`
		Cancel struct {
			Key []string `ini:"key"`
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"cancel"`
		Done struct {
			Key []string `ini:"key"`
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"done"`
		Preview struct {
			Snd  [2]int32 `ini:"snd" default:"-1,0"`
			Anim int32    `ini:"anim" default:"-1" preload:"char"`
			Spr  [2]int32 `ini:"spr" default:"-1,0" preload:"char"`
		} `ini:"preview"`
		Number TextProperties      `ini:"number"`
		Text   TextProperties      `ini:"text"`
		Bg     AnimationProperties `ini:"bg"`
	} `ini:"palmenu"`
}

type TeamModesProperties struct {
	Single string `ini:"single"`
	Simul  string `ini:"simul"`
	Turns  string `ini:"turns"`
	Tag    string `ini:"tag"`
	Ratio  string `ini:"ratio"`
}

type SelectInfoProperties struct {
	FadeIn               FadeProperties `ini:"fadein"`
	FadeOut              FadeProperties `ini:"fadeout"`
	Rows                 int32          `ini:"rows" default:"2"`
	Columns              int32          `ini:"columns" default:"5"`
	Wrapping             bool           `ini:"wrapping"`
	Pos                  [2]float32     `ini:"pos"`
	ShowEmptyBoxes       bool           `ini:"showemptyboxes"`
	MoveOverEmptyBoxes   bool           `ini:"moveoveremptyboxes"`
	SearchEmptyBoxesUp   bool           `ini:"searchemptyboxesup"`
	SearchEmptyBoxesDown bool           `ini:"searchemptyboxesdown"`
	Cell                 struct {
		Size    [2]int32            `ini:"size"`
		Spacing [2]float32          `ini:"spacing"`
		Bg      AnimationProperties `ini:"bg"`
		Random  struct {
			AnimationProperties
			SwitchTime int32 `ini:"switchtime"`
		} `ini:"random"`
		MapCell map[string]*CellOverrideProperties `ini:"map:^[0-9]+-[0-9]+$" lua:""`
	} `ini:"cell"`
	P1      PlayerSelectProperties `ini:"p1"`
	P2      PlayerSelectProperties `ini:"p2"`
	P3      PlayerSelectProperties `ini:"p3"`
	P4      PlayerSelectProperties `ini:"p4"`
	P5      PlayerSelectProperties `ini:"p5"`
	P6      PlayerSelectProperties `ini:"p6"`
	P7      PlayerSelectProperties `ini:"p7"`
	P8      PlayerSelectProperties `ini:"p8"`
	PalMenu struct {
		Random struct {
			SwitchTime int32 `ini:"switchtime"`
			ApplyPal   bool  `ini:"applypal" preload:"pal"`
		} `ini:"random"`
	} `ini:"palmenu"`
	Random struct {
		Move struct {
			Snd struct {
				Cancel bool `ini:"cancel"`
			} `ini:"snd"`
		} `ini:"move"`
	} `ini:"random"`
	Stage struct {
		Pos [2]float32 `ini:"pos"`
		TextProperties
		Random struct {
			Text string `ini:"text"`
		} `ini:"random"`
		Randomselect int32 `ini:"randomselect"`
		Active       struct {
			Font       [8]int32 `ini:"font" default:"-1,0,0,255,255,255,255,-1"`
			Switchtime int32    `ini:"switchtime"`
		} `ini:"active"`
		Active2 struct {
			Font [8]int32 `ini:"font" default:"-1,0,0,255,255,255,255,-1"`
		} `ini:"active2"`
		Done struct {
			Font [8]int32 `ini:"font" default:"-1,0,0,255,255,255,255,-1"`
			Snd  [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"done"`
		Move struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"move"`
		Portrait struct {
			AnimationStagePreloadProperties
			Bg     AnimationProperties `ini:"bg"`
			Random AnimationProperties `ini:"random"`
		} `ini:"portrait"`
	} `ini:"stage"`
	Cancel struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"cancel"`
	Portrait AnimationProperties `ini:"portrait"`
	Title    TextMapProperties   `ini:"title"`
	TeamMenu struct {
		Move struct {
			Wrapping bool `ini:"wrapping"`
		} `ini:"move"`
		Itemname map[string]*TeamModesProperties `ini:"itemname"`
	} `ini:"teammenu"`
	Timer         TimerProperties `ini:"timer"`
	Record        TextProperties  `ini:"record"`
	PaletteSelect int32           `ini:"paletteselect"`
}

type ValueIconVsProperties struct {
	AnimationProperties
	Spacing [2]float32 `ini:"spacing"` // not used by value.empty.icon, only used by P1-P2
}

type PlayerVsProperties struct {
	FaceProperties
	Face2 FaceProperties `ini:"face2"`
	Skip  struct {
		Key []string `ini:"key"`
	} `ini:"skip"`
	Select struct {
		Key []string `ini:"key"`
	} `ini:"select"`
	Name struct {
		TextProperties
		Pos     [2]float32 `ini:"pos"`
		Num     int32      `ini:"num"`     // only used by P1-P2
		Spacing [2]float32 `ini:"spacing"` // only used by P1-P2
	} `ini:"name"`
	Icon struct {
		AnimationProperties
		Done AnimationProperties `ini:"done"`
	} `ini:"icon"`
	Value struct {
		Icon  ValueIconVsProperties `ini:"icon"`
		Empty struct {
			Icon ValueIconVsProperties `ini:"icon"`
		} `ini:"empty"`
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"value"`
}

type AnimationStagePreloadProperties struct {
	Anim       int32      `ini:"anim" default:"-1" preload:"stage"`
	Spr        [2]int32   `ini:"spr" default:"-1,0" preload:"stage"`
	Offset     [2]float32 `ini:"offset"`
	Facing     int32      `ini:"facing" default:"1"`
	Scale      [2]float32 `ini:"scale" default:"1,1"`
	Xshear     float32    `ini:"xshear"`
	Angle      float32    `ini:"angle"`
	Layerno    int16      `ini:"layerno" default:"1"`
	Window     [4]int32   `ini:"window"`
	Localcoord [2]int32   `ini:"localcoord"`
	AnimData   *Anim
}

type VsScreenProperties struct {
	FadeIn      FadeProperties `ini:"fadein"`
	FadeOut     FadeProperties `ini:"fadeout"`
	Time        int32          `ini:"time"`
	Match       TextProperties `ini:"match"`
	OrderSelect struct {
		Enabled bool `ini:"enabled"`
	} `ini:"orderselect"`
	P1    PlayerVsProperties `ini:"p1"`
	P2    PlayerVsProperties `ini:"p2"`
	P3    PlayerVsProperties `ini:"p3"`
	P4    PlayerVsProperties `ini:"p4"`
	P5    PlayerVsProperties `ini:"p5"`
	P6    PlayerVsProperties `ini:"p6"`
	P7    PlayerVsProperties `ini:"p7"`
	P8    PlayerVsProperties `ini:"p8"`
	Timer TimerProperties    `ini:"timer"`
	Done  struct {
		Time int32 `ini:"time"`
	} `ini:"done"`
	Stage struct {
		Pos [2]float32 `ini:"pos"`
		TextProperties
		Portrait struct {
			AnimationStagePreloadProperties
			Bg AnimationProperties `ini:"bg"`
		} `ini:"portrait"`
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"stage"`
}

type DemoModeProperties struct {
	Enabled bool           `ini:"enabled"`
	FadeIn  FadeProperties `ini:"fadein"`
	FadeOut FadeProperties `ini:"fadeout"`
	Title   struct {
		WaitTime int32 `ini:"waittime"`
	} `ini:"title"`
	Fight struct {
		EndTime int32 `ini:"endtime"`
		PlayBgm bool  `ini:"playbgm"`
		StopBgm bool  `ini:"stopbgm"`
		Bars    struct {
			Display bool `ini:"display"`
		} `ini:"bars"`
	} `ini:"fight"`
	Intro struct {
		WaitCycles int32 `ini:"waitcycles"`
	} `ini:"intro"`
	DebugInfo bool `ini:"debuginfo"`
	Select    struct {
		Enabled bool `ini:"enabled"`
	} `ini:"select"`
	VsScreen struct {
		Enabled bool `ini:"enabled"`
	} `ini:"vsscreen"`
}

type ContinueCounterProperties struct {
	SkipTime int32    `ini:"skiptime"`
	Snd      [2]int32 `ini:"snd" default:"-1,0"`
}

type ContinueScreenProperties struct {
	Enabled  bool           `ini:"enabled"`
	FadeIn   FadeProperties `ini:"fadein"`
	FadeOut  FadeProperties `ini:"fadeout"`
	Pos      [2]float32     `ini:"pos"`
	Continue TextProperties `ini:"continue"`
	Yes      struct {
		TextProperties
		Active TextProperties `ini:"active"`
	} `ini:"yes"`
	No struct {
		TextProperties
		Active TextProperties `ini:"active"`
	} `ini:"no"`
	Sounds struct {
		Enabled bool `ini:"enabled"`
	} `ini:"sounds"`
	LegacyMode struct {
		Enabled bool `ini:"enabled" default:"true"`
	} `ini:"legacymode"`
	GameOver struct {
		Enabled bool `ini:"enabled"`
	} `ini:"gameover"`
	Move struct {
		Key []string `ini:"key"`
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"move"`
	Cancel struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"cancel"`
	Done struct {
		Key []string `ini:"key"`
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"done"`
	Overlay OverlayProperties `ini:"overlay"`
	P1      struct {
		State []int32 `ini:"state"`
		Yes   struct {
			State []int32 `ini:"state"`
		} `ini:"yes"`
		No struct {
			State []int32 `ini:"state"`
		} `ini:"no"`
		Teammate struct {
			State []int32 `ini:"state"`
			Yes   struct {
				State []int32 `ini:"state"`
			} `ini:"yes"`
			No struct {
				State []int32 `ini:"state"`
			} `ini:"no"`
		} `ini:"teammate"`
	} `ini:"p1"`
	P2 struct {
		State []int32 `ini:"state"`
		Yes   struct {
			State []int32 `ini:"state"`
		} `ini:"yes"`
		No struct {
			State []int32 `ini:"state"`
		} `ini:"no"`
		Teammate struct {
			State []int32 `ini:"state"`
			Yes   struct {
				State []int32 `ini:"state"`
			} `ini:"yes"`
			No struct {
				State []int32 `ini:"state"`
			} `ini:"no"`
		} `ini:"teammate"`
	} `ini:"p2"`
	Credits TextProperties `ini:"credits"`
	Counter struct {
		AnimationProperties
		StartTime int32 `ini:"starttime"`
		EndTime   int32 `ini:"endtime"`
		Default   struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"default"`
		SkipStart int32                                 `ini:"skipstart"`
		MapCounts map[string]*ContinueCounterProperties `ini:"map:^[0-9]+$" lua:""`
		End       ContinueCounterProperties             `ini:"end"`
	} `ini:"counter"`
}

type StoryboardProperties struct {
	Enabled    bool   `ini:"enabled"`
	Storyboard string `ini:"storyboard" lookup:"def,,data/"`
}

type PlayerVictoryProperties struct {
	FaceProperties
	Face2    FaceProperties `ini:"face2"`
	Name     TextProperties `ini:"name"`
	State    []int32        `ini:"state"`
	Teammate struct {
		State []int32 `ini:"state"`
	} `ini:"teammate"`
}

type VictoryScreenProperties struct {
	Enabled bool `ini:"enabled"`
	Sounds  struct {
		Enabled bool `ini:"enabled"`
	} `ini:"sounds"`
	Cpu struct {
		Enabled bool `ini:"enabled"`
	} `ini:"cpu"`
	Winner struct {
		TeamKo struct {
			Enabled bool `ini:"enabled"`
		} `ini:"teamko"`
	} `ini:"winner"`
	FadeIn   FadeProperties `ini:"fadein"`
	FadeOut  FadeProperties `ini:"fadeout"`
	Time     int32          `ini:"time"`
	WinQuote struct {
		TextProperties
		TextSpacing float32 `ini:"textspacing"`
		TextDelay   int32   `ini:"textdelay"`
		TextWrap    string  `ini:"textwrap"`
		DisplayTime int32   `ini:"displaytime"`
	} `ini:"winquote"`
	Overlay OverlayProperties `ini:"overlay"`
	Skip    struct {
		Key []string `ini:"key"`
	} `ini:"skip"`
	Cancel struct {
		Key []string `ini:"key"`
	} `ini:"cancel"`
	P1 PlayerVictoryProperties `ini:"p1"`
	P2 PlayerVictoryProperties `ini:"p2"`
	P3 PlayerVictoryProperties `ini:"p3"`
	P4 PlayerVictoryProperties `ini:"p4"`
	P5 PlayerVictoryProperties `ini:"p5"`
	P6 PlayerVictoryProperties `ini:"p6"`
	P7 PlayerVictoryProperties `ini:"p7"`
	P8 PlayerVictoryProperties `ini:"p8"`
}

type PlayerResultsProperties struct {
	State []int32  `ini:"state"`
	Win   struct { // not used by [Win Screen], [Time Attack Result Screen]
		State []int32 `ini:"state"`
	} `ini:"win"`
	Teammate struct {
		State []int32  `ini:"state"`
		Win   struct { // not used by [Win Screen], [Time Attack Result Screen]
			State []int32 `ini:"state"`
		} `ini:"win"`
	} `ini:"teammate"`
}

type WinScreenProperties struct {
	Enabled bool `ini:"enabled"`
	Sounds  struct {
		Enabled bool `ini:"enabled"`
	} `ini:"sounds"`
	FadeIn  FadeProperties `ini:"fadein"`
	FadeOut FadeProperties `ini:"fadeout"`
	Pose    struct {
		Time int32 `ini:"time"`
	} `ini:"pose"`
	State struct {
		Time int32 `ini:"time"`
	} `ini:"state"`
	WinText struct {
		TextProperties
		DisplayTime int32 `ini:"displaytime"`
	} `ini:"wintext"`
	Overlay OverlayProperties `ini:"overlay"`
	Cancel  struct {
		Key []string `ini:"key"`
	} `ini:"cancel"`
	P1 PlayerResultsProperties `ini:"p1"`
	P2 PlayerResultsProperties `ini:"p2"`
}

type ResultsScreenProperties struct {
	Enabled bool `ini:"enabled"`
	Sounds  struct {
		Enabled bool `ini:"enabled"`
	} `ini:"sounds"`
	RoundsToWin int32          `ini:"roundstowin"` // used only by [Survival Results Screen]
	FadeIn      FadeProperties `ini:"fadein"`
	FadeOut     FadeProperties `ini:"fadeout"`
	Show        struct {
		Time int32 `ini:"time"`
	} `ini:"show"`
	State struct {
		Time int32 `ini:"time"`
	} `ini:"state"`
	WinsText struct {
		TextProperties
		DisplayTime int32 `ini:"displaytime"`
	} `ini:"winstext"`
	Overlay OverlayProperties `ini:"overlay"`
	Cancel  struct {
		Key []string `ini:"key"`
	} `ini:"cancel"`
	P1 PlayerResultsProperties `ini:"p1"`
	P2 PlayerResultsProperties `ini:"p2"`
}

type OptionInfoProperties struct {
	FadeIn  FadeProperties `ini:"fadein"`
	FadeOut FadeProperties `ini:"fadeout"`
	Title   TextProperties `ini:"title"`
	Menu    MenuProperties `ini:"menu"`
	Cursor  struct {
		Move struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"move"`
		Done struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"done"`
	} `ini:"cursor"`
	Cancel struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"cancel"`
	TextInput struct {
		TextMapProperties
		Overlay OverlayProperties `ini:"overlay"`
	} `ini:"textinput"`
	KeyMenu struct {
		P1 struct {
			MenuOffset [2]float32     `ini:"menuoffset"`
			Playerno   TextProperties `ini:"playerno"`
		} `ini:"p1"`
		P2 struct {
			MenuOffset [2]float32     `ini:"menuoffset"`
			Playerno   TextProperties `ini:"playerno"`
		} `ini:"p2"`
		Menu     MenuProperties    `ini:"menu"`
		Itemname map[string]string `ini:"itemname"`
	} `ini:"keymenu"`
	Itemname map[string]string `ini:"itemname"` // not used by [Option Info]
}

type ReplayInfoProperties struct {
	FadeIn  FadeProperties `ini:"fadein"`
	FadeOut FadeProperties `ini:"fadeout"`
	Title   TextProperties `ini:"title"`
	Menu    MenuProperties `ini:"menu"`
	Cursor  struct {
		Move struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"move"`
		Done struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"done"`
	} `ini:"cursor"`
	Cancel struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"cancel"`
}

type MenuInfoProperties struct {
	Enabled bool           `ini:"enabled"`
	FadeIn  FadeProperties `ini:"fadein"`
	FadeOut FadeProperties `ini:"fadeout"`
	Title   TextProperties `ini:"title"`
	Menu    MenuProperties `ini:"menu"`
	Cursor  struct {
		Move struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"move"`
		Done struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"done"`
	} `ini:"cursor"`
	Cancel struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"cancel"`
	Enter struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"enter"`
	Overlay  OverlayProperties `ini:"overlay"`
	Movelist struct {
		Pos   [2]float32 `ini:"pos"`
		Title struct {
			TextProperties
			Uppercase bool `ini:"uppercase"`
		} `ini:"title"`
		Text struct {
			TextProperties
			Spacing [2]float32 `ini:"spacing"`
		} `ini:"text"`
		Glyphs struct {
			Offset     [2]float32 `ini:"offset"`
			Scale      [2]float32 `ini:"scale" default:"1,1"`
			Layerno    int16      `ini:"layerno" default:"1"`
			Localcoord [2]int32   `ini:"localcoord"`
			Spacing    [2]float32 `ini:"spacing"`
		} `ini:"glyphs"`
		Window struct {
			Margins struct {
				Y [2]float32 `ini:"y"`
			} `ini:"margins"`
			VisibleItems int32 `ini:"visibleitems"`
			Width        int32 `ini:"width"`
		} `ini:"window"`
		Overlay OverlayProperties `ini:"overlay"`
		Arrow   struct {
			Up   AnimationProperties `ini:"up"`
			Down AnimationProperties `ini:"down"`
		} `ini:"arrow"`
		Itemname map[string]string `ini:"itemname"`
	} `ini:"movelist"`
}

type AttractModeProperties struct {
	Enabled bool           `ini:"enabled"`
	FadeIn  FadeProperties `ini:"fadein"`
	FadeOut FadeProperties `ini:"fadeout"`
	Title   TextProperties `ini:"title"`
	Menu    MenuProperties `ini:"menu"`
	Cursor  struct {
		Move struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"move"`
		Done struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"done"`
		Snd map[string][2]int32 `ini:"snd"`
	} `ini:"cursor"`
	Cancel struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"cancel"`
	Credits struct {
		TextProperties
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"credits"`
	Logo struct {
		Storyboard string `ini:"storyboard" lookup:"def,,data/"`
	} `ini:"logo"`
	Intro struct {
		Storyboard string `ini:"storyboard" lookup:"def,,data/"`
	} `ini:"intro"`
	Start struct {
		Storyboard string `ini:"storyboard" lookup:"def,,data/"`
		Done       struct {
			Snd [2]int32 `ini:"snd" default:"-1,0"`
		} `ini:"done"`
		Time   int32 `ini:"time"`
		Insert struct {
			TextProperties
			Blinktime int32 `ini:"blinktime"`
		} `ini:"insert"`
		Press struct {
			TextProperties
			Blinktime int32 `ini:"blinktime"`
		} `ini:"press"`
		Timer TimerProperties `ini:"timer"`
	} `ini:"start"`
	Options struct {
		KeyCode string `ini:"keycode"`
	} `ini:"options"`
}

type ChallengerInfoProperties struct {
	Enabled bool           `ini:"enabled"`
	FadeIn  FadeProperties `ini:"fadein"`
	FadeOut FadeProperties `ini:"fadeout"`
	Time    int32          `ini:"time"`
	Pause   struct {
		Time int32 `ini:"time"`
	} `ini:"pause"`
	Snd struct {
		Snd  [2]int32 `ini:"" default:"-1,0"`
		Time int32    `ini:"time"`
	} `ini:"snd"`
	Text struct {
		TextProperties
		Displaytime int32 `ini:"displaytime"`
	} `ini:"text"`
	Bg struct {
		AnimationProperties
		Displaytime int32 `ini:"displaytime"`
	} `ini:"bg"`
	Overlay OverlayProperties `ini:"overlay"`
}

type PlayerDialogueProperties struct {
	Bg   AnimationProperties `ini:"bg"`
	Face AnimationProperties `ini:"face"`
	Name TextProperties      `ini:"name"`
	Text struct {
		TextProperties
		TextSpacing float32 `ini:"textspacing"`
		TextDelay   int32   `ini:"textdelay"`
		TextWrap    string  `ini:"textwrap"`
	} `ini:"text"`
	Active AnimationProperties `ini:"active"`
}

type DialogueInfoProperties struct {
	Enabled    bool  `ini:"enabled"`
	StartTime  int32 `ini:"starttime"`
	EndTime    int32 `ini:"endtime"`
	SwitchTime int32 `ini:"switchtime"`
	SkipTime   int32 `ini:"skiptime"`
	Skip       struct {
		Key []string `ini:"key"`
	} `ini:"skip"`
	Cancel struct {
		Key []string `ini:"key"`
	} `ini:"cancel"`
	P1 PlayerDialogueProperties `ini:"p1"`
	P2 PlayerDialogueProperties `ini:"p2"`
}

type HiscoreInfoProperties struct {
	Enabled bool              `ini:"enabled"`
	FadeIn  FadeProperties    `ini:"fadein"`
	FadeOut FadeProperties    `ini:"fadeout"`
	Time    int32             `ini:"time"`
	Pos     [2]float32        `ini:"pos"`
	Ranking map[string]string `ini:"ranking"`
	Title   struct {
		TextMapProperties
		Uppercase bool           `ini:"uppercase"`
		Rank      TextProperties `ini:"rank"`
		Result    TextProperties `ini:"result"`
		Name      TextProperties `ini:"name"`
		Face      TextProperties `ini:"face"`
	} `ini:"title"`
	Item struct {
		Offset  [2]float32     `ini:"offset"`
		Spacing [2]float32     `ini:"spacing"`
		Rank    ItemProperties `ini:"rank"`
		Result  ItemProperties `ini:"result"`
		Name    ItemProperties `ini:"name"`
		Face    struct {
			AnimationCharPreloadProperties
			Num     int32               `ini:"num"`
			Spacing [2]float32          `ini:"spacing"`
			Bg      AnimationProperties `ini:"bg"`
			Unknown AnimationProperties `ini:"unknown"`
		} `ini:"face"`
	} `ini:"item"`
	Timer  TimerProperties `ini:"timer"`
	Window struct {
		Margins struct {
			Y [2]float32 `ini:"y"`
		} `ini:"margins"`
		VisibleItems int32 `ini:"visibleitems"`
		Width        int32 `ini:"width"`
	} `ini:"window"`
	Overlay OverlayProperties `ini:"overlay"`
	Next    struct {
		Key []string `ini:"key"`
	} `ini:"next"`
	Previous struct {
		Key []string `ini:"key"`
	} `ini:"previous"`
	Done struct {
		Key  []string `ini:"key"`
		Snd  [2]int32 `ini:"snd" default:"-1,0"`
		Time int32    `ini:"time"`
	} `ini:"done"`
	Move struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"move"`
	Cancel struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"cancel"`
	Glyphs []string `ini:"glyphs"`
}

type WarningInfoProperties struct {
	Title   TextProperties    `ini:"title"`
	Text    TextMapProperties `ini:"text"`
	Overlay OverlayProperties `ini:"overlay"`
	Cancel  struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"cancel"`
	Done struct {
		Snd [2]int32 `ini:"snd" default:"-1,0"`
	} `ini:"done"`
}

type GlyphProperties struct {
	Spr    [2]int32   `ini:""` // default ini target: e.g. [Glyphs] ^3K = 63,0 -> Glyphs["^3K"].Spr = [63,0]
	Offset [2]float32 `ini:"offset"`
	//Facing     int32      `ini:"facing" default:"1"`
	Scale [2]float32 `ini:"scale" default:"1,1"`
	//Xshear     float32    `ini:"xshear"`
	//Angle      float32    `ini:"angle"`
	Layerno int16 `ini:"layerno" default:"1"`
	//Window     [4]int32   `ini:"window"`
	Localcoord [2]int32 `ini:"localcoord"`
	AnimData   *Anim
	Size       [2]int32 `lua:"Size"`
}

type Motif struct {
	IniFile                 *ini.File
	UserIniFile             *ini.File
	DefaultOnlyIni          *ini.File
	At                      AnimationTable
	Sff                     *Sff
	Snd                     *Snd
	Fnt                     map[int]*Fnt
	GlyphsSff               *Sff
	Model                   *Model
	Music                   Music
	Def                     string                      `ini:"def"`
	Info                    InfoProperties              `ini:"info"`
	Files                   FilesProperties             `ini:"files"`
	Languages               map[string]string           `ini:"languages"`
	TitleInfo               TitleInfoProperties         `ini:"title_info"`
	TitleBgDef              BgDefProperties             `ini:"titlebgdef"`
	InfoBox                 InfoBoxProperties           `ini:"infobox"`
	SelectInfo              SelectInfoProperties        `ini:"select_info"`
	SelectBgDef             BgDefProperties             `ini:"selectbgdef"`
	VsScreen                VsScreenProperties          `ini:"vs_screen"`
	VersusBgDef             BgDefProperties             `ini:"versusbgdef"`
	DemoMode                DemoModeProperties          `ini:"demo_mode"`
	ContinueScreen          ContinueScreenProperties    `ini:"continue_screen"`
	ContinueBgDef           BgDefProperties             `ini:"continuebgdef"`
	GameOverScreen          StoryboardProperties        `ini:"game_over_screen"`
	VictoryScreen           VictoryScreenProperties     `ini:"victory_screen"`
	VictoryBgDef            BgDefProperties             `ini:"victorybgdef"`
	WinScreen               WinScreenProperties         `ini:"win_screen"`
	WinBgDef                BgDefProperties             `ini:"winbgdef"`
	DefaultEnding           StoryboardProperties        `ini:"default_ending"`
	EndCredits              StoryboardProperties        `ini:"end_credits"`
	SurvivalResultsScreen   ResultsScreenProperties     `ini:"survival_results_screen"`
	SurvivalResultsBgDef    BgDefProperties             `ini:"survivalresultsbgdef"`
	TimeAttackResultsScreen ResultsScreenProperties     `ini:"time_attack_results_screen"`
	TimeAttackResultsBgDef  BgDefProperties             `ini:"timeattackresultsbgdef"`
	OptionInfo              OptionInfoProperties        `ini:"option_info"`
	OptionBgDef             BgDefProperties             `ini:"optionbgdef"`
	ReplayInfo              ReplayInfoProperties        `ini:"replay_info"`
	ReplayBgDef             BgDefProperties             `ini:"replaybgdef"`
	MenuInfo                MenuInfoProperties          `ini:"menu_info"`
	MenuBgDef               BgDefProperties             `ini:"menubgdef"`
	TrainingInfo            MenuInfoProperties          `ini:"training_info"`
	TrainingBgDef           BgDefProperties             `ini:"trainingbgdef"`
	AttractMode             AttractModeProperties       `ini:"attract_mode"`
	AttractBgDef            BgDefProperties             `ini:"attractbgdef"`
	ChallengerInfo          ChallengerInfoProperties    `ini:"challenger_info"`
	ChallengerBgDef         BgDefProperties             `ini:"challengerbgdef"`
	DialogueInfo            DialogueInfoProperties      `ini:"dialogue_info"`
	HiscoreInfo             HiscoreInfoProperties       `ini:"hiscore_info"`
	HiscoreBgDef            BgDefProperties             `ini:"hiscorebgdef"`
	WarningInfo             WarningInfoProperties       `ini:"warning_info"`
	Glyphs                  map[string]*GlyphProperties `ini:"glyphs" literal:"true" insensitivekeys:"false" sff:"GlyphsSff"`
	fntIndexByKey           map[string]int              // filepath|height -> index
	ch                      MotifChallenger
	co                      MotifContinue
	de                      MotifDemo
	di                      MotifDialogue
	vi                      MotifVictory
	wi                      MotifWin
	hi                      MotifHiscore
	me                      MotifMenu
	fadeIn                  *Fade
	fadeOut                 *Fade
	fadePolicy              FadeStartPolicy
	btnPressedFlag          bool
	textsprite              []*TextSprite
}

// hasUserKey returns true if the given key exists in `section` of the INI.
func hasUserKey(iniFile *ini.File, section, key string) bool {
	sec, err := iniFile.GetSection(section)
	if err != nil {
		return false
	}
	return sec.HasKey(key)
}

// applyCustomDefaults injects custom defaults.
func applyCustomDefaults(m *Motif, iniFile *ini.File) {
	// Copy the first spacing value into the second one when only a single value is defined in system.def.
	sec := "Select Info"
	// Only act if the user actually set the key in their system.def
	s, err := iniFile.GetSection(sec)
	if err != nil || s == nil {
		return
	}
	k, err := s.GetKey("cell.spacing")
	if err != nil {
		return
	}
	raw := strings.TrimSpace(k.Value())
	// If user specified a single value (no comma/&), duplicate it into the second slot.
	if raw != "" && !strings.Contains(raw, ",") && !strings.Contains(raw, "&") {
		// At this point, the first component has already been parsed into the struct.
		// Just mirror it to the second component.
		m.SelectInfo.Cell.Spacing[1] = m.SelectInfo.Cell.Spacing[0]
	}

	// Inject computed defaults based on system.def localcoord when specific keys in system.def are not defined.
	w := int(m.Info.Localcoord[0])
	h := int(m.Info.Localcoord[1])
	sec = "Title Info"

	// loading.offset = (W - 1 - 10*H/320, H - 8)
	if !hasUserKey(iniFile, sec, "loading.offset") {
		x := w - 1 - (10*h)/320
		y := h - 8
		_ = SetValue(m, "title_info.loading.offset", fmt.Sprintf("%d, %d", x, y))
	}

	// footer.title.offset = (2*W/320, H)
	if !hasUserKey(iniFile, sec, "footer.title.offset") {
		x := (2 * w) / 320
		y := h
		_ = SetValue(m, "title_info.footer.title.offset", fmt.Sprintf("%d, %d", x, y))
	}

	// footer.info.offset = (W/2, H)
	if !hasUserKey(iniFile, sec, "footer.info.offset") {
		x := w / 2
		y := h
		_ = SetValue(m, "title_info.footer.info.offset", fmt.Sprintf("%d, %d", x, y))
	}

	// footer.version.offset = (W - 1 - 2*W/320, H)
	if !hasUserKey(iniFile, sec, "footer.version.offset") {
		x := w - 1 - (2*w)/320
		y := h
		_ = SetValue(m, "title_info.footer.version.offset", fmt.Sprintf("%d, %d", x, y))
	}

	// footer.overlay.window = (0, H - 7, W, H)
	if !hasUserKey(iniFile, sec, "footer.overlay.window") {
		x1, y1, x2, y2 := 0, h-7, w, h
		_ = SetValue(m, "title_info.footer.overlay.window", fmt.Sprintf("%d, %d, %d, %d", x1, y1, x2, y2))
	}
}

// reserveUserFontSlots marks user-specified [Files] font indices as taken so
// resolveInlineFonts won't auto-assign inline fonts to those slots.
func reserveUserFontSlots(m *Motif) {
	if m == nil || m.UserIniFile == nil {
		return
	}
	sec, err := m.UserIniFile.GetSection("Files")
	if err != nil || sec == nil {
		return
	}
	re := regexp.MustCompile(`(?i)^font(\d+)$`)
	for _, k := range sec.Keys() {
		name := k.Name()
		match := re.FindStringSubmatch(name)
		if len(match) == 2 {
			idx := int(Atoi(match[1]))
			if _, exists := m.Fnt[idx]; !exists {
				m.Fnt[idx] = nil // placeholder to block ensureFontIndex from using this slot
			}
		}
	}
}

// preprocessINIContent removes or modifies specific sections before parsing.
func preprocessINIContent(input string) string {
	// Define a regex to find the [Infobox Text] section
	infoboxRegex := regexp.MustCompile(`(?s)\[Infobox Text\]\n(.*?)(\n\[|$)`)
	// Extract the content of [Infobox Text]
	matches := infoboxRegex.FindStringSubmatch(input)
	if len(matches) < 3 {
		// If the section is not found, return the original input
		return input
	}
	infoboxTextContent := matches[1]
	// Process the extracted text
	processedText := strings.TrimSpace(infoboxTextContent)
	processedText = strings.ReplaceAll(processedText, "\n", `\n`)
	// Resolve first two %s placeholders to Version and BuildTime
	processedText = strings.Replace(processedText, "%s", Version, 1)
	processedText = strings.Replace(processedText, "%s", BuildTime, 1)
	// Create the new text.text line with an added newline at the end
	newTextLine := fmt.Sprintf("\ttext.text = %s\n\n", processedText)
	// Remove the [Infobox Text] section from the input
	output := infoboxRegex.ReplaceAllString(input, "$2")
	// Define a regex to find the [InfoBox] section header
	infoBoxHeaderRegex := regexp.MustCompile(`(?m)(\[InfoBox\]\n)`)
	// Insert the new text.text line right after the [InfoBox] header
	output = infoBoxHeaderRegex.ReplaceAllString(output, "${1}"+newTextLine)
	return output
}

// loadMotif loads and parses the INI file into a Motif struct.
func loadMotif(def string) (*Motif, error) {
	if def == "" {
		sys.motif.resolvePath()
		def = sys.motif.Def
	}
	// Define load options if needed
	// https://github.com/go-ini/ini/blob/main/ini.go
	options := ini.LoadOptions{
		Insensitive:             false,
		InsensitiveSections:     true,
		InsensitiveKeys:         false,
		IgnoreInlineComment:     false,
		SkipUnrecognizableLines: true,
		//AllowBooleanKeys: true,
		AllowShadows: false,
		//AllowNestedValues: true,
		UnparseableSections:        []string{"Infobox Text"},
		AllowPythonMultilineValues: false,
		//KeyValueDelimiters: "=:",
		//KeyValueDelimiterOnWrite: "=",
		//ChildSectionDelimiter: ".",
		//AllowNonUniqueSections: true,
		//AllowDuplicateShadowValues: true,
	}

	// Load the INI file
	var iniFile *ini.File
	var userIniFile *ini.File
	var defaultOnlyIni *ini.File

	if err := LoadFile(&def, []string{def, "", "data/"}, func(filename string) error {
		inputBytes, err := LoadText(filename)
		if err != nil {
			return fmt.Errorf("Failed to load text from %s: %w", filename, err)
		}

		createTempFile := func(content string) (*os.File, error) {
			tmp, err := os.CreateTemp("", "temp_*.ini")
			if err != nil {
				return nil, fmt.Errorf("could not create temporary file: %w", err)
			}
			// Ensure the temporary file is removed when the function exits
			defer os.Remove(tmp.Name())

			if _, err := tmp.WriteString(content); err != nil {
				tmp.Close()
				return nil, fmt.Errorf("failed to write to temporary file %s: %w", tmp.Name(), err)
			}

			return tmp, nil
		}

		normalizedInput := preprocessINIContent(NormalizeNewlines(string(inputBytes)))
		tempDefFile, err := createTempFile(normalizedInput)
		if err != nil {
			return err
		}

		normalizedDefault := preprocessINIContent(NormalizeNewlines(string(defaultMotif)))
		tempDefaultFile, err := createTempFile(normalizedDefault)
		if err != nil {
			return err
		}

		// Load the INI file using the temporary files
		iniFile, err = ini.LoadSources(options, tempDefaultFile.Name(), tempDefFile.Name())
		if err != nil {
			return fmt.Errorf("Failed to load INI sources from %s and %s: %w", tempDefaultFile.Name(), tempDefFile.Name(), err)
		}

		// Also keep a defaults-only INI, so we can apply it before user overrides.
		defaultOnlyIni, err = ini.LoadSources(options, tempDefaultFile.Name())
		if err != nil {
			return fmt.Errorf("Failed to load defaults-only INI from %s: %w", tempDefaultFile.Name(), err)
		}

		// Load user-only INI to know which keys the user actually provided in system.def
		userIniFile, err = ini.LoadSources(options, tempDefFile.Name())
		if err != nil {
			return fmt.Errorf("Failed to load user INI source from %s: %w", tempDefFile.Name(), err)
		}

		tempDefaultFile.Close()
		tempDefFile.Close()

		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading definition file: %v\n", err)
		return nil, err
	}

	var m Motif
	m.Def = def
	m.initStruct()

	assignFrom := func(src *ini.File) {
		if src == nil {
			return
		}
		type secPair struct {
			sec  *ini.Section
			name string // logical (language-stripped) name
		}
		var baseSecs, langSecs []secPair
		curLang := SelectedLanguage()

		for _, s := range src.Sections() {
			raw := s.Name()
			if raw == ini.DEFAULT_SECTION {
				continue
			}
			lang, base, has := splitLangPrefix(raw)
			logical := base
			// Backgrounds and [Begin Action] blocks are skipped (case-insensitive).
			lb := strings.ToLower(logical)
			for _, p := range []string{
				"begin ", "titlebg ", "selectbg ", "versusbg ", "continuebg ",
				"victorybg ", "winbg ", "survivalresultsbg ", "timeattackresultsbg ",
				"optionbg ", "replaybg ", "menubg ", "trainingbg ", "attractbg ",
				"challengerbg ", "hiscorebg ",
			} {
				if strings.HasPrefix(lb, p) {
					goto nextSection
				}
			}
			// "music" is handled separately later.
			if strings.EqualFold(logical, "music") {
				goto nextSection
			}
			// Route by language.
			if has {
				if lang == "en" {
					baseSecs = append(baseSecs, secPair{s, logical})
				} else if lang == curLang {
					langSecs = append(langSecs, secPair{s, logical})
				}
			} else {
				baseSecs = append(baseSecs, secPair{s, logical})
			}
		nextSection:
		}

		process := func(pair secPair) {
			section := pair.sec
			sectionName := pair.name
			for _, key := range section.Keys() {
				keyName := key.Name()
				if strings.HasPrefix(keyName, "menu.itemname.") {
					// handled in script.go to preserve order
					continue
				}
				value := key.Value()

				// Normalize spaces
				secNorm := strings.ReplaceAll(sectionName, " ", "_")
				keyNorm := strings.ReplaceAll(keyName, " ", "_")

				var keyParts []queryPart
				// Literal sections keep dots in keys
				if isLiteralSectionFor(&m, sectionName) {
					keyParts = []queryPart{
						{name: strings.ToLower(secNorm)},
						{name: keyNorm},
					}
				} else {
					fullKey := strings.ToLower(secNorm) + "." + keyNorm
					keyParts = parseQueryPath(fullKey)
				}

				if err := assignField(&m, keyParts, value, def); err != nil {
					fmt.Printf("Warning: Failed to assign key [%s.%s]: %v\n", sectionName, keyName, err)
				}
			}
		}
		for _, sp := range baseSecs {
			process(sp)
		}
		for _, sp := range langSecs {
			process(sp)
		}
	}

	// Apply precedence: struct defaults < defaultMotif.ini < user motif
	assignFrom(defaultOnlyIni)
	assignFrom(userIniFile)
	sys.keepAlive()

	// Localcoord is used during loading (before the final sys.motif assignment)
	sys.motif.Info.Localcoord = m.Info.Localcoord

	m.IniFile = iniFile
	m.UserIniFile = userIniFile
	m.DefaultOnlyIni = defaultOnlyIni
	if userIniFile != nil {
		applyCustomDefaults(&m, userIniFile)
	}

	// Resolve inline fonts early, so TitleInfo.Loading can get a real font index.
	reserveUserFontSlots(&m)
	resolveInlineFonts(m.IniFile, m.Def, m.Fnt, m.fntIndexByKey, m.SetValueUpdate)
	syncFontsMap(&m.Files.Font, m.Fnt, m.fntIndexByKey)
	sys.keepAlive()

	// Build the loading TextSprite and draw it before we proceed to heavier asset loads (SFF, BG, etc).
	m.drawLoading()
	sys.keepAlive()

	// Proceed with the regular heavyweight loads.
	m.loadFiles()
	sys.keepAlive()

	str, err := LoadText(def)
	if err != nil {
		return nil, err
	}
	lines, i := SplitAndTrim(str, "\n"), 0
	m.At = ReadAnimationTable(m.Sff, &m.Sff.palList, lines, &i)
	i = 0

	m.overrideParams()
	m.fixLocalcoordOverrides()
	m.applyGlyphDefaultsFromMovelist()
	m.populateDataPointers()
	m.applyPostParsePosAdjustments()

	m.Music = parseMusicSection(pickLangSection(iniFile, "Music"))

	return &m, nil
}

// InheritSpec describes how to inherit keys from one prefix to another inside the given INI sections.
// Example: srcSec="Option Info", srcPrefix="menu.", dstSec="Option Info", dstPrefix="keymenu.menu."
type InheritSpec struct {
	SrcSec    string
	SrcPrefix string
	DstSec    string
	DstPrefix string
}

// Propagates movelist.glyphs defaults
func (m *Motif) applyGlyphDefaultsFromMovelist() {
	if m == nil {
		return
	}
	if len(m.Glyphs) == 0 {
		return
	}

	mg := m.MenuInfo.Movelist.Glyphs
	for _, g := range m.Glyphs {
		if g == nil {
			continue
		}
		g.Offset = mg.Offset
		g.Scale = mg.Scale
		g.Layerno = mg.Layerno
		g.Localcoord = mg.Localcoord
	}
}

// If an element is customized in userIniFile but its localcoord is missing/empty,
// use Info.Localcoord instead of the defaultMotif value; otherwise keep the default.
func (m *Motif) fixLocalcoordOverrides() {
	if m == nil || m.UserIniFile == nil || m.DefaultOnlyIni == nil || m.IniFile == nil {
		return
	}

	for _, mergedSec := range m.IniFile.Sections() {
		secName := mergedSec.Name()
		if secName == ini.DEFAULT_SECTION {
			continue
		}

		userSec, _ := m.UserIniFile.GetSection(secName)
		defSec, _ := m.DefaultOnlyIni.GetSection(secName)

		var userKeys []*ini.Key
		if userSec != nil {
			userKeys = userSec.Keys()
		}

		for _, key := range mergedSec.Keys() {
			keyName := key.Name()
			lowerKey := strings.ToLower(keyName)

			if !strings.HasSuffix(lowerKey, ".localcoord") {
				continue
			}

			lastDot := strings.LastIndex(keyName, ".")
			if lastDot < 0 {
				continue
			}
			prefix := keyName[:lastDot+1]
			lowerPrefix := strings.ToLower(prefix)

			// 1) If user explicitly sets this .localcoord (non-empty) in their motif, always respect it.
			if userSec != nil {
				if uk, err := userSec.GetKey(keyName); err == nil && uk != nil &&
					strings.TrimSpace(uk.Value()) != "" {
					continue
				}
			}

			// 2) Is this exact .localcoord key present in the embedded defaults?
			inDefaults := false
			if defSec != nil {
				if dk, err := defSec.GetKey(keyName); err == nil && dk != nil {
					inDefaults = true
				}
			}

			// 3) Did the user (or inheritance that wrote back into UserIniFile) touch this element?
			// Check if user customized this element (any other key with same prefix).
			elementTouched := false
			for _, uk := range userKeys {
				n := uk.Name()
				ln := strings.ToLower(n)
				if strings.HasPrefix(ln, lowerPrefix) &&
					ln != lowerKey &&
					!strings.HasSuffix(ln, ".key") &&
					!strings.HasSuffix(ln, ".snd") &&
					strings.TrimSpace(uk.Value()) != "" {
					elementTouched = true
					break
				}
			}

			// 4) If missing from defaults, any same-prefix non-localcoord key in merged INI marks override.
			if !inDefaults && !elementTouched {
				for _, mk := range mergedSec.Keys() {
					n := mk.Name()
					ln := strings.ToLower(n)
					if strings.HasPrefix(ln, lowerPrefix) &&
						ln != lowerKey &&
						strings.TrimSpace(mk.Value()) != "" {
						elementTouched = true
						break
					}
				}
			}
			if !elementTouched {
				// Pure untouched default (or truly isolated entry): keep its default localcoord.
				continue
			}

			// Let PopulateDataPointers re-fill this from motif localcoord:
			// set struct (and ini) value to 0,0 so it is treated as "unspecified".
			secNorm := strings.ReplaceAll(secName, " ", "_")
			keyNorm := strings.ReplaceAll(keyName, " ", "_")
			query := strings.ToLower(secNorm + "." + keyNorm)

			if err := m.SetValueUpdate(query, "0, 0"); err != nil {
				fmt.Printf("Warning: failed to reset localcoord for %s: %v\n", query, err)
			}
		}
	}
}

// mergeWithInheritance applies "user overrides default" with intra-file inheritance.
func (m *Motif) mergeWithInheritance(specs []InheritSpec) {
	if m == nil || m.IniFile == nil {
		return
	}
	user := m.UserIniFile
	defs := m.DefaultOnlyIni
	merged := m.IniFile

	get := func(f *ini.File, sec, key string) (string, bool) {
		if f == nil {
			return "", false
		}
		s, err := f.GetSection(sec)
		if err != nil || s == nil {
			return "", false
		}
		if !s.HasKey(key) {
			return "", false
		}
		k, _ := s.GetKey(key)
		if k == nil {
			return "", false
		}
		return k.Value(), true
	}

	shouldSkip := func(fullKeyLower string) bool {
		// Avoid inheriting dotted item/valuename lists which are consumed elsewhere.
		return strings.Contains(fullKeyLower, ".itemname.") || strings.Contains(fullKeyLower, ".valuename.")
	}

	// Ensure a section exists in the user ini when we need to mirror
	// a user-originated inherited key into it.
	ensureUserSection := func(name string) *ini.Section {
		if user == nil {
			return nil
		}
		if s, err := user.GetSection(name); err == nil && s != nil {
			return s
		}
		s, err := user.NewSection(name)
		if err != nil {
			fmt.Printf("Warning: failed to create section %s in user ini: %v\n", name, err)
			return nil
		}
		return s
	}

	type valueSource int
	const (
		srcNone valueSource = iota
		srcUserDst
		srcUserSrc
		srcDefDst
		srcDefSrc
	)

	for _, sp := range specs {
		// Build a set of suffixes present under either src/dst in user or default.
		suffixes := map[string]struct{}{}
		collect := func(f *ini.File, sec, prefix string) {
			if f == nil {
				return
			}
			s, err := f.GetSection(sec)
			if err != nil || s == nil {
				return
			}
			lp := strings.ToLower(prefix)
			for _, k := range s.Keys() {
				kn := k.Name()
				lkn := strings.ToLower(kn)
				if strings.HasPrefix(lkn, lp) {
					if shouldSkip(lkn) {
						continue
					}
					// same-length lowercasing keeps indices aligned (ASCII)
					suf := kn[len(prefix):]
					suffixes[suf] = struct{}{}
				}
			}
		}
		collect(user, sp.SrcSec, sp.SrcPrefix)
		collect(user, sp.DstSec, sp.DstPrefix)
		collect(defs, sp.SrcSec, sp.SrcPrefix)
		collect(defs, sp.DstSec, sp.DstPrefix)

		// Resolve and set final values following the precedence.
		for suf := range suffixes {
			dstKey := sp.DstPrefix + suf
			srcKey := sp.SrcPrefix + suf
			lowerFull := strings.ToLower(dstKey)
			if shouldSkip(lowerFull) {
				continue
			}

			// If the user didn't set dst/src, and the merged INI already has dst,
			// skip touching it to avoid clobbering a resolved (e.g. font-index) value.
			if _, uDst := get(user, sp.DstSec, dstKey); !uDst {
				if _, uSrc := get(user, sp.SrcSec, srcKey); !uSrc {
					if _, haveMerged := get(merged, sp.DstSec, dstKey); haveMerged {
						// Nothing to inherit here; keep the existing (possibly resolved) value.
						continue
					}
				}
			}

			var (
				val string
				src valueSource
			)

			// Priority: user dst > user src > default dst > default src
			if v, ok := get(user, sp.DstSec, dstKey); ok {
				val, src = v, srcUserDst
			} else if v, ok := get(user, sp.SrcSec, srcKey); ok {
				val, src = v, srcUserSrc
			} else if v, ok := get(defs, sp.DstSec, dstKey); ok {
				val, src = v, srcDefDst
			} else if v, ok := get(defs, sp.SrcSec, srcKey); ok {
				val, src = v, srcDefSrc
			} else {
				continue
			}

			// Write into the merged ini & struct.
			secPath := strings.ReplaceAll(sp.DstSec, " ", "_")
			query := strings.ToLower(secPath + "." + dstKey)
			if err := m.SetValueUpdate(query, val); err != nil {
				//fmt.Printf("Warning: inheritance set failed for %s = %q: %v\n", query, val, err)
			}

			// If a value comes from the user INI (directly or via src), copy it into m.UserIniFile
			// so fixLocalcoordOverrides treats the element as user-touched, including inherited keys.
			if user != nil {
				switch src {
				case srcUserDst:
					// Already present in user INI at the destination; nothing to do.
				case srcUserSrc:
					if sec := ensureUserSection(sp.DstSec); sec != nil && !sec.HasKey(dstKey) {
						if _, err := sec.NewKey(dstKey, val); err != nil {
							//fmt.Printf("Warning: failed to write inherited key %s.%s into user ini: %v\n", sp.DstSec, dstKey, err)
						}
					}
				default:
					// Default-sourced inheritance: do NOT mirror into userIniFile.
				}
			}
		}
	}
}

func (m *Motif) overrideParams() {
	// Define inheritance rules (section/prefix based).
	specs := []InheritSpec{
		// [Training Info]
		{SrcSec: "Menu Info", SrcPrefix: "", DstSec: "Training Info", DstPrefix: ""},
		// [Option Info]
		{SrcSec: "Option Info", SrcPrefix: "menu.", DstSec: "Option Info", DstPrefix: "keymenu.menu."},
		// [Select Info]
		{SrcSec: "Select Info", SrcPrefix: "p1.", DstSec: "Select Info", DstPrefix: "p3."},
		{SrcSec: "Select Info", SrcPrefix: "p1.", DstSec: "Select Info", DstPrefix: "p5."},
		{SrcSec: "Select Info", SrcPrefix: "p1.", DstSec: "Select Info", DstPrefix: "p7."},
		{SrcSec: "Select Info", SrcPrefix: "p2.", DstSec: "Select Info", DstPrefix: "p4."},
		{SrcSec: "Select Info", SrcPrefix: "p2.", DstSec: "Select Info", DstPrefix: "p6."},
		{SrcSec: "Select Info", SrcPrefix: "p2.", DstSec: "Select Info", DstPrefix: "p8."},
		// [VS Screen]
		{SrcSec: "VS Screen", SrcPrefix: "p1.", DstSec: "VS Screen", DstPrefix: "p3."},
		{SrcSec: "VS Screen", SrcPrefix: "p1.", DstSec: "VS Screen", DstPrefix: "p5."},
		{SrcSec: "VS Screen", SrcPrefix: "p1.", DstSec: "VS Screen", DstPrefix: "p7."},
		{SrcSec: "VS Screen", SrcPrefix: "p2.", DstSec: "VS Screen", DstPrefix: "p4."},
		{SrcSec: "VS Screen", SrcPrefix: "p2.", DstSec: "VS Screen", DstPrefix: "p6."},
		{SrcSec: "VS Screen", SrcPrefix: "p2.", DstSec: "VS Screen", DstPrefix: "p8."},
		// [Victory Screen]
		{SrcSec: "Victory Screen", SrcPrefix: "p1.", DstSec: "Victory Screen", DstPrefix: "p3."},
		{SrcSec: "Victory Screen", SrcPrefix: "p1.", DstSec: "Victory Screen", DstPrefix: "p5."},
		{SrcSec: "Victory Screen", SrcPrefix: "p1.", DstSec: "Victory Screen", DstPrefix: "p7."},
		{SrcSec: "Victory Screen", SrcPrefix: "p2.", DstSec: "Victory Screen", DstPrefix: "p4."},
		{SrcSec: "Victory Screen", SrcPrefix: "p2.", DstSec: "Victory Screen", DstPrefix: "p6."},
		{SrcSec: "Victory Screen", SrcPrefix: "p2.", DstSec: "Victory Screen", DstPrefix: "p8."},
	}
	m.mergeWithInheritance(specs)
	// Inheritance may add new font filenames; rerun resolver to map them to deduped font indices.
	//resolveInlineFonts(m.IniFile, m.Def, m.Fnt, m.fntIndexByKey, m.SetValueUpdate)
}

func (m *Motif) loadBgDefProperties(bgDef *BgDefProperties, bgname, spr string) {
	if bgDef.Spr == "" || bgDef.Spr == spr || bgDef.Spr == m.Files.Spr {
		bgDef.Sff = m.Sff
	} else {
		LoadFile(&bgDef.Spr, []string{bgDef.Spr, m.Def, "", "data/"}, func(filename string) error {
			if filename != "" {
				var err error
				bgDef.Sff, err = loadSff(filename, false)
				if err != nil {
					sys.errLog.Printf("Failed to load %v: %v", filename, err)
				}
			}
			if bgDef.Sff == nil {
				bgDef.Sff = m.Sff
			}
			return nil
		})
	}
	if bgname != "" {
		var err error
		bgDef.BGDef, err = loadBGDef(bgDef.Sff, m.Model, m.Def, bgname, bgDef.DefaultLayer)
		if err != nil {
			sys.errLog.Printf("Failed to load %v (%v): %v\n", bgname, m.Def, err.Error())
		}
	}
	if bgDef.BGDef == nil {
		bgDef.BGDef = newBGDef(m.Def)
	}
	sys.keepAlive()
}

func (m *Motif) loadFiles() {
	LoadFile(&m.Files.Spr, []string{m.Files.Spr}, func(filename string) error {
		if filename != "" {
			var err error
			m.Sff, err = loadSff(filename, false)
			if err != nil {
				sys.errLog.Printf("Failed to load %v: %v", filename, err)
			}
		}
		if m.Sff == nil {
			m.Sff = newSff()
		}
		return nil
	})
	sys.keepAlive()

	LoadFile(&m.Files.Glyphs, []string{m.Files.Glyphs}, func(filename string) error {
		if filename != "" {
			var err error
			m.GlyphsSff, err = loadSff(filename, false)
			if err != nil {
				sys.errLog.Printf("Failed to load %v: %v", filename, err)
			}
		}
		if m.GlyphsSff == nil {
			m.GlyphsSff = newSff()
		}
		return nil
	})
	sys.keepAlive()

	LoadFile(&m.Files.Model, []string{m.Files.Model}, func(filename string) error {
		if filename != "" {
			var err error
			m.Model, err = loadglTFModel(filename)
			if err != nil {
				sys.errLog.Printf("Failed to load %v: %v", filename, err)
			}
			sys.mainThreadTask <- func() {
				gfx.SetModelVertexData(1, m.Model.vertexBuffer)
				gfx.SetModelIndexData(1, m.Model.elementBuffer...)
			}
			sys.runMainThreadTask()
			sys.keepAlive()
		}
		return nil
	})

	m.loadBgDefProperties(&m.TitleBgDef, "titlebg", m.Files.Spr)
	m.loadBgDefProperties(&m.SelectBgDef, "selectbg", m.Files.Spr)
	m.loadBgDefProperties(&m.VersusBgDef, "versusbg", m.Files.Spr)
	m.loadBgDefProperties(&m.ContinueBgDef, "continuebg", m.Files.Spr)
	m.loadBgDefProperties(&m.VictoryBgDef, "victorybg", m.Files.Spr)
	m.loadBgDefProperties(&m.WinBgDef, "winbg", m.Files.Spr)
	m.loadBgDefProperties(&m.SurvivalResultsBgDef, "survivalresultsbg", m.Files.Spr)
	m.loadBgDefProperties(&m.TimeAttackResultsBgDef, "timeattackresultsbg", m.Files.Spr)
	m.loadBgDefProperties(&m.OptionBgDef, "optionbg", m.Files.Spr)
	if _, err := m.UserIniFile.GetSection("ReplayBgDef"); err == nil {
		m.loadBgDefProperties(&m.ReplayBgDef, "replaybg", m.Files.Spr)
	} else {
		m.ReplayBgDef = m.TitleBgDef
	}
	m.loadBgDefProperties(&m.MenuBgDef, "menubg", m.Files.Spr)
	if _, err := m.UserIniFile.GetSection("TrainingBgDef"); err == nil {
		m.loadBgDefProperties(&m.TrainingBgDef, "trainingbg", m.Files.Spr)
	} else {
		m.TrainingBgDef = m.MenuBgDef
	}
	if _, err := m.UserIniFile.GetSection("AttractBgDef"); err == nil {
		m.loadBgDefProperties(&m.AttractBgDef, "attractbg", m.Files.Spr)
	} else {
		m.AttractBgDef = m.TitleBgDef
	}
	m.loadBgDefProperties(&m.ChallengerBgDef, "challengerbg", m.Files.Spr)
	if _, err := m.UserIniFile.GetSection("HiscoreBgDef"); err == nil {
		m.loadBgDefProperties(&m.HiscoreBgDef, "hiscorebg", m.Files.Spr)
	} else {
		m.HiscoreBgDef = m.TitleBgDef
	}

	LoadFile(&m.Files.Snd, []string{m.Files.Snd}, func(filename string) error {
		if filename != "" {
			var err error
			m.Snd, err = LoadSnd(filename)
			if err != nil {
				sys.errLog.Printf("Failed to load %v: %v", filename, err)
			}
		}
		if m.Snd == nil {
			m.Snd = newSnd()
		}
		return nil
	})
	sys.keepAlive()

	for key, fnt := range m.Files.Font {
		LoadFile(&fnt.Font, []string{fnt.Font}, func(filename string) error {
			re := regexp.MustCompile(`\d+`)
			i := int(Atoi(re.FindString(key)))

			if filename != "" {
				var err error
				m.Fnt[i], err = loadFnt(filename, fnt.Height)
				registerFontIndex(m.fntIndexByKey, filename, fnt.Height, i)
				if err != nil {
					sys.errLog.Printf("Failed to load %v: %v", filename, err)
				}
			}
			if m.Fnt[i] == nil {
				m.Fnt[i] = newFnt()
			}
			// Populate extended properties from the loaded font
			if m.Fnt[i] != nil {
				fnt.Type = m.Fnt[i].Type
				fnt.Size = m.Fnt[i].Size
				fnt.Spacing = m.Fnt[i].Spacing
				fnt.Offset = m.Fnt[i].offset
			}
			return nil
		})
		sys.keepAlive()
	}
}

// Initialize struct
func (m *Motif) initStruct() {
	initMaps(reflect.ValueOf(m).Elem())
	applyDefaultsToValue(reflect.ValueOf(m).Elem())
	m.fadeIn = newFade()
	m.fadeOut = newFade()
	m.fadePolicy = FadeContinue
	m.fntIndexByKey = make(map[string]int)
}

// GetValue retrieves the value based on the query string.
func (m *Motif) GetValue(query string) (interface{}, error) {
	return GetValue(m, query)
}

// SetValue sets the value based on the query string and updates the IniFile.
func (m *Motif) SetValueUpdate(query string, value interface{}) error {
	return SetValueUpdate(m, m.IniFile, query, value)
}

// Save writes the current IniFile to disk, preserving comments and syntax.
func (m *Motif) Save(file string) error {
	return SaveINI(m.IniFile, file)
}

func (m *Motif) drawLoading() {
	// Build directly from the struct values so we don't need to populate everything.
	v := reflect.ValueOf(&m.TitleInfo.Loading).Elem()
	f := v.FieldByName("TextSpriteData")
	if f.IsValid() && f.CanSet() && f.IsNil() {
		SetTextSprite(m, f, v, reflect.Value{})
	}

	ts := m.TitleInfo.Loading.TextSpriteData
	if ts == nil {
		return
	}
	ts.SetLocalcoord(float32(m.Info.Localcoord[0]), float32(m.Info.Localcoord[1]))
	ts.SetWindow([4]float32{
		0, 0,
		float32(m.Info.Localcoord[0]),
		float32(m.Info.Localcoord[1]),
	})
	ts.Reset()

	sys.mainThreadTask <- func() {
		// Start a one-off frame
		gfx.BeginFrame(true)

		FillRect(sys.scrrect, 0x000000, [2]int32{255, 0})

		ts.Draw(ts.layerno)
		BlendReset()

		// Submit and present
		gfx.EndFrame()
		if strings.HasPrefix(gfx.GetName(), "OpenGL") {
			sys.window.SwapBuffers()
		} else {
			gfx.Await()
		}

		// Prepare a fresh frame for whatever comes next (no clear)
		gfx.BeginFrame(false)
	}
	sys.runMainThreadTask()
}

func (m *Motif) populateDataPointers() {
	PopulateDataPointers(m, m.Info.Localcoord)
}

func (m *Motif) button(btns []string, controllerNo int) bool {
	// Workaround for the lack of button release detection when reading KeyConfig inputs.
	if m.btnPressedFlag {
		return false
	}
	for _, btn := range btns {
		n := sys.button(btn)
		if n >= 0 && (n == controllerNo || controllerNo == -1) {
			return true
		}
	}
	return false
}

func (mo *Motif) processStateChange(c *Char, states []int32) bool {
	for _, stateNo := range states {
		if c.selfStatenoExist(BytecodeInt(stateNo)) == BytecodeBool(true) {
			c.changeState(int32(stateNo), -1, -1, "")
			return true
		}
	}
	return false
}

func (mo *Motif) processStateTransitions(winnerState, winnerTeammateState, loserState, loserTeammateState []int32) {
	isWinnerLeader, isLoserLeader := false, false
	for _, p := range sys.chars {
		if len(p) == 0 {
			continue
		}
		c := p[0]
		if c.win() {
			// Handle P1 state
			if !isWinnerLeader {
				mo.processStateChange(c, winnerState)
				isWinnerLeader = true
			} else {
				mo.processStateChange(c, winnerTeammateState)
			}
		} else {
			// Handle P2 state
			if !isLoserLeader {
				mo.processStateChange(c, loserState)
				isLoserLeader = true
			} else {
				mo.processStateChange(c, loserTeammateState)
			}
		}
	}
}

// replaceFormatSpecifiers converts Lua 5.1/C-style format specifiers
// to Go's fmt.Sprintf equivalents where they differ.
func (mo *Motif) replaceFormatSpecifiers(input string) string {
	// Verbs that exist in Lua's string.format/C but not in Go's fmt: %i, %u, %I
	// Everything else we care about (%d, %o, %x, %X, %e, %E, %f, %g, %G, %c, %s, %q, %p, %%) is already understood by fmt.
	var formatSpecifierMap = map[string]string{
		"i": "d",
		"I": "d",
		"u": "d",
	}
	re := regexp.MustCompile(`%%|%([-+ #0]*)?(\d+)?(\.\d+)?([hlLzjt]*)?([a-zA-Z])`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		// Keep literal %%
		if match == "%%" {
			return "%%"
		}
		sub := re.FindStringSubmatch(match)
		if len(sub) != 6 {
			return match // should not happen, but be safe
		}
		flags := sub[1]
		width := sub[2]
		precision := sub[3]
		// length := sub[4] // dropped for Go
		verb := sub[5]
		// Map Lua/C-only verbs to Go equivalents
		if mapped, ok := formatSpecifierMap[verb]; ok {
			verb = mapped
		}
		// Rebuild without the C length modifier.
		var b strings.Builder
		b.WriteByte('%')
		if flags != "" {
			b.WriteString(flags)
		}
		if width != "" {
			b.WriteString(width)
		}
		if precision != "" {
			b.WriteString(precision)
		}
		b.WriteString(verb)
		return b.String()
	})
}

func (m *Motif) reset() {
	m.fadeIn.reset()
	m.fadeOut.reset()
	m.ch.reset(m)
	m.de.reset(m)
	m.di.reset(m)
	m.di.clear(m)
	m.vi.reset(m)
	m.vi.clear(m)
	m.wi.reset(m)
	m.hi.reset(m)
	m.co.reset(m)
	m.textsprite = []*TextSprite{}
	//sys.storyboard.reset()
	m.me.reset(m)
}

func (m *Motif) step() {
	if sys.paused && !sys.frameStepFlag {
		return
	}
	if m.me.active {
		m.me.step(m)
	} else if sys.escExit() {
		sys.endMatch = true
		return
	}
	if m.ch.active {
		m.ch.step(m)
	}
	if m.de.active {
		m.de.step(m)
	}
	if m.di.active {
		m.di.step(m)
	}
	if m.vi.active {
		m.vi.step(m)
	}
	if m.wi.active {
		m.wi.step(m)
	}
	if m.hi.active {
		m.hi.step(m)
	}
	if m.co.active {
		m.co.step(m)
	}
	m.UpdateText()
	if sys.storyboard.active {
		sys.storyboard.step()
	}
	m.btnPressedFlag = sys.keyInput != KeyUnknown
}

func (m *Motif) UpdateText() {
	// Explod timers update at this time, so we'll do the same here
	if sys.tickNextFrame() {
		tempSlice := m.textsprite[:0]

		for _, ts := range m.textsprite {
			ts.Update()
			if ts.removetime != 0 {
				tempSlice = append(tempSlice, ts) // Keep this text
				if ts.removetime > 0 {
					ts.removetime--
				}
			}
		}

		m.textsprite = tempSlice
	}
}

func (m *Motif) removeText(id, index, ownerid int32) {
	n := int32(0)

	// Mark matching texts invalid
	for _, ts := range m.textsprite {
		if (id == -1 && ts.ownerid == ownerid) || (id != -1 && ts.id == id && ts.ownerid == ownerid) {
			if index < 0 || index == n {
				ts.id = IErr
				if index == n {
					break
				}
			}
			n++
		}
	}

	// Compact the slice to remove invalid texts
	tempSlice := m.textsprite[:0] // Reuse backing array
	for _, ts := range m.textsprite {
		if ts.id != IErr {
			tempSlice = append(tempSlice, ts)
		}
	}
	m.textsprite = tempSlice
}

func (m *Motif) draw(layerno int16) {
	if m.ch.active {
		m.ch.draw(m, layerno)
	}
	if m.de.active {
		m.de.draw(m, layerno)
	}
	if m.di.active {
		m.di.draw(m, layerno)
	}
	if m.vi.active {
		m.vi.draw(m, layerno)
	}
	if m.wi.active {
		m.wi.draw(m, layerno)
	}
	if m.hi.active {
		m.hi.draw(m, layerno)
	}
	if m.co.active {
		m.co.draw(m, layerno)
	}
	for _, v := range m.textsprite {
		v.Draw(layerno)
	}
	if sys.storyboard.active {
		sys.storyboard.draw(layerno)
	}
	if m.me.active {
		m.me.draw(m, layerno)
	}
	// Screen fading
	if layerno == 2 {
		if m.fadeOut.isActive() {
			m.fadeOut.draw()
		} else if m.fadeIn.isActive() {
			m.fadeIn.draw()
		}
	}
	BlendReset()
}

func (m *Motif) isDialogueSet() bool {
	if sys.dialogueForce != 0 {
		return false
	}
	for _, p := range sys.chars {
		if len(p) > 0 && len(p[0].dialogue) > 0 {
			return true
		}
	}
	return false
}

func (m *Motif) act() {
	if (sys.paused && !sys.frameStepFlag) || sys.gsf(GSF_roundfreeze) {
		return
	}
	// Storyboard
	//if sys.storyboard.IniFile != nil && !sys.storyboard.initialized  {
	//	sys.storyboard.init()
	//}
	// Menu / Exit
	if !m.me.initialized {
		m.me.init(m)
	}
	// Demo Mode
	if !m.de.initialized {
		m.de.init(m)
	}
	if sys.postMatchFlg {
		// Victory Screen
		if !m.vi.initialized {
			m.vi.init(m)
		}
		if m.vi.active {
			return
		}
		// Win / Results Screen
		if !m.wi.initialized {
			m.wi.init(m)
		}
		if m.wi.active {
			return
		}
		// Continue Screen
		if !m.co.initialized {
			m.co.init(m)
		}
		if m.co.active {
			return
		}
		sys.postMatchFlg = false
	} else {
		// Challenger
		if !m.ch.initialized {
			m.ch.init(m)
		}
		// TODO: Hiscore is initialized explicitly when we want to show it
		// Dialogue
		if !m.di.initialized && ((sys.round == 1 && sys.intro == sys.lifebar.ro.ctrl_time) ||
			(sys.roundStateTicks() == sys.lifebar.ro.fadeOut.time && sys.matchOver())) && m.isDialogueSet() {
			m.di.init(m)
		}
	}
}

func FormatTimeText(text string, totalSec float64) string {
	h := int(totalSec / 3600)
	m := int(totalSec/60) % 60
	s := int(totalSec) % 60
	x := int((totalSec - float64(int(totalSec))) * 100)
	// Ensure two-digit formatting for minutes, seconds, and fractions
	mStr := fmt.Sprintf("%02d", m)
	sStr := fmt.Sprintf("%02d", s)
	xStr := fmt.Sprintf("%02d", x)
	// Replace placeholders
	result := strings.ReplaceAll(text, "%h", fmt.Sprintf("%d", h))
	result = strings.ReplaceAll(result, "%m", mStr)
	result = strings.ReplaceAll(result, "%s", sStr)
	result = strings.ReplaceAll(result, "%x", xStr)
	return result
}

func (m *Motif) setMotifScale(localcoord [2]int32) {
	// not needed
}

func (m *Motif) resolvePath() {
	v := sys.cfg.Config.Motif
	if x, ok := sys.cmdFlags["-r"]; ok {
		v = x
	} else if x, ok := sys.cmdFlags["-rubric"]; ok {
		v = x
	}

	v = filepath.ToSlash(v)
	lower := strings.ToLower(v)

	if strings.HasPrefix(lower, "data/") {
		if FileExist(v) != "" {
			m.Def = v
			return
		}
	}

	if strings.HasSuffix(lower, ".def") {
		if cand := SearchFile(v, []string{"data/"}); FileExist(cand) != "" {
			m.Def = filepath.ToSlash(cand)
			return
		}
	}

	dir := filepath.ToSlash(filepath.Join("data", v)) + "/"
	if cand := SearchFile("system.def", []string{dir}); FileExist(cand) != "" {
		m.Def = filepath.ToSlash(cand)
		return
	}

	m.Def = v
}

func (m *Motif) applyPostParsePosAdjustments() {
	animSetPos := func(a *Anim, dx, dy float32) {
		a.SetPos(a.offsetInit[0]+dx, a.offsetInit[1]+dy)
	}
	textSetPos := func(ts *TextSprite, dx, dy float32) {
		ts.SetPos(ts.offsetInit[0]+dx, ts.offsetInit[1]+dy)
	}
	offsetAnims := func(dx, dy float32, anims ...*Anim) {
		for _, a := range anims {
			animSetPos(a, dx, dy)
		}
	}
	offsetTexts := func(dx, dy float32, texts ...*TextSprite) {
		for _, ts := range texts {
			textSetPos(ts, dx, dy)
		}
	}
	shiftMenu := func(me *MenuProperties) {
		dx, dy := me.Pos[0], me.Pos[1]
		// Arrows
		offsetAnims(dx, dy, me.Arrow.Up.AnimData, me.Arrow.Down.AnimData)
		// Common item texts
		offsetTexts(dx, dy,
			me.Item.TextSpriteData,
			me.Item.Selected.TextSpriteData,
			me.Item.Selected.Active.TextSpriteData,
			me.Item.Active.TextSpriteData,
			me.Item.Value.TextSpriteData,
			me.Item.Value.Active.TextSpriteData,
			// Extra fields some menus use:
			me.Item.Value.Conflict.TextSpriteData,
			me.Item.Info.TextSpriteData,
			me.Item.Info.Active.TextSpriteData,
		)
		// Backgrounds
		for _, ap := range me.Item.Bg {
			animSetPos(ap.AnimData, dx, dy)
		}
		for _, ap := range me.Item.Active.Bg {
			animSetPos(ap.AnimData, dx, dy)
		}
	}
	adjustSelect := func(ps *PlayerSelectProperties) {
		tm := &ps.TeamMenu
		// TeamMenu backgrounds
		for _, ap := range tm.Bg {
			animSetPos(ap.AnimData, tm.Pos[0], tm.Pos[1])
		}
		for _, ap := range tm.Active.Bg {
			animSetPos(ap.AnimData, tm.Pos[0], tm.Pos[1])
		}
		// Titles & base text
		offsetAnims(tm.Pos[0], tm.Pos[1], tm.SelfTitle.AnimData, tm.EnemyTitle.AnimData)
		offsetTexts(tm.Pos[0], tm.Pos[1], tm.SelfTitle.TextSpriteData, tm.EnemyTitle.TextSpriteData)
		offsetTexts(tm.Pos[0], tm.Pos[1], tm.Item.TextSpriteData)
		// Icons at (Pos + Item.Offset)
		offX := tm.Pos[0] + tm.Item.Offset[0]
		offY := tm.Pos[1] + tm.Item.Offset[1]
		offsetAnims(offX, offY,
			tm.Item.Cursor.AnimData,
			tm.Value.Icon.AnimData,
			tm.Value.Empty.Icon.AnimData,
			tm.Ratio1.Icon.AnimData,
			tm.Ratio2.Icon.AnimData,
			tm.Ratio3.Icon.AnimData,
			tm.Ratio4.Icon.AnimData,
			tm.Ratio5.Icon.AnimData,
			tm.Ratio6.Icon.AnimData,
			tm.Ratio7.Icon.AnimData,
		)
		// Palette menu
		pm := &ps.PalMenu
		animSetPos(pm.Bg.AnimData, pm.Pos[0], pm.Pos[1])
		offsetTexts(pm.Pos[0], pm.Pos[1], pm.Number.TextSpriteData, pm.Text.TextSpriteData)

		// Face.Random and Face2.Random
		offsetAnims(ps.Face.Pos[0], ps.Face.Pos[1], ps.Face.Random.AnimData)
		offsetAnims(ps.Face2.Pos[0], ps.Face2.Pos[1], ps.Face2.Random.AnimData)
	}

	// Select Screen: Players
	for _, ps := range []*PlayerSelectProperties{
		&m.SelectInfo.P1, &m.SelectInfo.P2, &m.SelectInfo.P3, &m.SelectInfo.P4,
		&m.SelectInfo.P5, &m.SelectInfo.P6, &m.SelectInfo.P7, &m.SelectInfo.P8,
	} {
		adjustSelect(ps)
	}

	// Select Screen: Stage portrait
	{
		st := &m.SelectInfo.Stage
		offsetAnims(st.Pos[0], st.Pos[1], st.Portrait.Bg.AnimData, st.Portrait.Random.AnimData)
		textSetPos(st.TextSpriteData, st.Pos[0], st.Pos[1])
	}

	// Menus
	for _, me := range []*MenuProperties{
		&m.TitleInfo.Menu,
		&m.OptionInfo.Menu,
		&m.ReplayInfo.Menu,
		&m.AttractMode.Menu,
		&m.MenuInfo.Menu,
		&m.TrainingInfo.Menu,
		&m.OptionInfo.KeyMenu.Menu,
	} {
		shiftMenu(me)
	}

	// KeyMenu
	{
		km := &m.OptionInfo.KeyMenu
		textSetPos(km.P1.Playerno.TextSpriteData, km.Menu.Pos[0]+km.P1.MenuOffset[0], km.Menu.Pos[1]+km.P1.MenuOffset[1])
		textSetPos(km.P2.Playerno.TextSpriteData, km.Menu.Pos[0]+km.P2.MenuOffset[0], km.Menu.Pos[1]+km.P2.MenuOffset[1])
	}

	// Movelists (MenuInfo, TrainingInfo)
	{
		mv := &m.MenuInfo.Movelist
		offsetAnims(mv.Pos[0], mv.Pos[1], mv.Arrow.Up.AnimData, mv.Arrow.Down.AnimData)
		offsetTexts(mv.Pos[0], mv.Pos[1], mv.Title.TextSpriteData, mv.Text.TextSpriteData)

		mv2 := &m.TrainingInfo.Movelist
		offsetAnims(mv2.Pos[0], mv2.Pos[1], mv2.Arrow.Up.AnimData, mv2.Arrow.Down.AnimData)
		offsetTexts(mv2.Pos[0], mv2.Pos[1], mv2.Title.TextSpriteData, mv2.Text.TextSpriteData)
	}

	// VS Screen: player name positions
	{
		type namePos struct {
			ts  *TextSprite
			pos [2]float32
		}
		names := []namePos{
			{m.VsScreen.P1.Name.TextSpriteData, m.VsScreen.P1.Name.Pos},
			{m.VsScreen.P2.Name.TextSpriteData, m.VsScreen.P2.Name.Pos},
			{m.VsScreen.P3.Name.TextSpriteData, m.VsScreen.P3.Name.Pos},
			{m.VsScreen.P4.Name.TextSpriteData, m.VsScreen.P4.Name.Pos},
			{m.VsScreen.P5.Name.TextSpriteData, m.VsScreen.P5.Name.Pos},
			{m.VsScreen.P6.Name.TextSpriteData, m.VsScreen.P6.Name.Pos},
			{m.VsScreen.P7.Name.TextSpriteData, m.VsScreen.P7.Name.Pos},
			{m.VsScreen.P8.Name.TextSpriteData, m.VsScreen.P8.Name.Pos},
		}
		for _, n := range names {
			textSetPos(n.ts, n.pos[0], n.pos[1])
		}
	}

	// VS Screen: stage
	{
		st := &m.VsScreen.Stage
		offsetAnims(st.Pos[0], st.Pos[1], st.Portrait.Bg.AnimData)
		offsetTexts(st.Pos[0], st.Pos[1], st.TextSpriteData)
	}
}
