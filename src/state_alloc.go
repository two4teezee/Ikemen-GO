package main

type Allocator[T any] struct {
	c       uint64
	storage []T
	size    uint64
	factor  int
}

func NewAllocator[T any](size int, factor int) Allocator[T] {
	return Allocator[T]{
		storage: make([]T, size),
		size:    uint64(size),
		factor:  factor,
	}
}

func (a *Allocator[T]) GetSlice(size int) []T {
	a.c++
	c := int(a.c)
	if c+size >= int(a.size) {
		a.Reallocate(size * a.factor)
	}
	return a.storage[c : c+size : c+size]
}

func (a *Allocator[T]) GetPointer() *T {
	a.c++
	if a.c >= a.size {
		a.Reallocate(1 * a.factor)
	}
	return &a.storage[a.c]
}

func (a *Allocator[T]) Get() T {
	a.c++
	if a.c >= a.size {
		a.Reallocate(1 * a.factor)
	}
	return a.storage[a.c]
}

func (a *Allocator[T]) Reallocate(size int) {
	storage := a.storage
	a.storage = make([]T, int(a.size)+size)
	copy(a.storage, storage)
	a.size += uint64(size)
}

func (a *Allocator[T]) Reset() {
	a.c = 0
}

type StateAllocator struct {
	gameState                Allocator[GameState]
	commandListSliceStorage  Allocator[CommandList]
	coomandKeySliceStorage   Allocator[CommandKey]
	commandKey2dSliceStorage Allocator[[]CommandKey]
	commandSliceStorage      Allocator[Command]
	command2dSliceStorage    Allocator[[]Command]
	cmdElemSliceStorage      Allocator[cmdElem]
	boolSliceStorage         Allocator[bool]
	stringIntMapStorage      Allocator[map[string]int]

	palFXSliceStorage Allocator[PalFX]

	hitscaleMapStorage Allocator[map[int32][3]*HitScale]
	hitscaleStorage    Allocator[[3]*HitScale]

	charPointerSliceStorage Allocator[*Char]
	int32SliceStorage       Allocator[int32]
	float32SliceStorage     Allocator[float32]

	stringSliceStorage      Allocator[string]
	stringFloat32MapStorage Allocator[map[string]float32]

	animFrameSliceStorage       Allocator[AnimFrame]
	intSliceStorage             Allocator[int]
	bytecodeExpStorage          Allocator[BytecodeExp]
	stateControllerSliceStorage Allocator[StateController]
	stateDefStorage             Allocator[stateDef]
	charSliceStorage            Allocator[Char]

	float322dSliceStorage Allocator[[]float32]

	hitByStorage                 Allocator[[2]int32]
	overrideCharDataSliceStorage Allocator[OverrideCharData]
	int322dSliceStorage          Allocator[[]int32]

	string2dSliceStorage          Allocator[[]string]
	backGroundPointerSliceStorage Allocator[*backGround]
	bgCtrlPointerSliceStorage     Allocator[*bgCtrl]
	bgCtrlSliceStorage            Allocator[bgCtrl]
	bgctNodeSliceStorage          Allocator[bgctNode]
	animationTableStorage         Allocator[AnimationTable]
	mapArraySliceStorage          Allocator[([]map[string]float32)]
	int32CharPointerMapStorage    Allocator[map[int32]*Char]

	healthBarPointerSliceStorage   Allocator[*HealthBar]
	powerBarPointerSliceStorage    Allocator[*PowerBar]
	guardBarPointerSliceStorage    Allocator[*GuardBar]
	stunBarPointerSliceStorage     Allocator[*StunBar]
	lifeBarFacePointerSliceStorage Allocator[*LifeBarFace]
	LifeBarNamePointerSliceStorage Allocator[*LifeBarName]

	healthBarPointerStorage   Allocator[HealthBar]
	powerBarPointerStorage    Allocator[PowerBar]
	guardBarPointerStorage    Allocator[GuardBar]
	stunBarPointerStorage     Allocator[StunBar]
	lifeBarFacePointerStorage Allocator[LifeBarFace]
	LifeBarNamePointerStorage Allocator[LifeBarName]
}

func (sa *StateAllocator) AllocGameState() *GameState {
	return sa.gameState.GetPointer()
}

func (sa *StateAllocator) ResetGameState() {
	sa.gameState.Reset()
}
