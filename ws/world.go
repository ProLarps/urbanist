package ws

import "math"

// BuildingDTO per docs/API.md §6. willBase is the citizens' default opinion
// (server-side only; not sent to clients).
type BuildingDTO struct {
	ID        int     `json:"id"`
	Type      int     `json:"type"`
	Floors    int     `json:"floors"`
	Condition float64 `json:"condition"`
	Heritage  float64 `json:"heritage"`
	Residents int     `json:"residents"`
	Status    string  `json:"status"`
	Progress  float64 `json:"progress"`
	Needs     bool    `json:"needs"`
	WillBase  float64 `json:"-"`
}

// ChunkDTO per docs/API.md §6.
type ChunkDTO struct {
	CX        int           `json:"cx"`
	CZ        int           `json:"cz"`
	Buildings []BuildingDTO `json:"buildings"`
}

// Building statuses.
const (
	StatusIdle       = "idle"
	StatusRenovating = "renovating"
	StatusDone       = "done"
)

// Building types per docs/API.md §4.
const (
	TypeVacantLot    = 0
	TypePanelka      = 1
	TypeBrickHouse   = 2
	TypeKhrushchevka = 3
	TypeMansion      = 4
	TypeFactory      = 5
	TypePark         = 6
	TypeShop         = 7
)

// World is one room's city: ChunksPerAxis x ChunksPerAxis chunks,
// LotsPerChunk x LotsPerChunk lots per chunk, deterministically generated
// from the room seed. Authoritative and persisted (docs/API.md §13).
type World struct {
	lots []BuildingDTO // index == lotId
	cfg  Config
}

func NewWorld(seed int64, cfg Config) *World {
	total := cfg.ChunksPerAxis * cfg.ChunksPerAxis * cfg.LotsPerChunk * cfg.LotsPerChunk
	w := &World{lots: make([]BuildingDTO, total), cfg: cfg}

	for lotID := 0; lotID < total; lotID++ {
		cx, cz, ix, iz := decomposeLot(lotID, cfg)
		// Distinct texture per (chunk, lot position) from hashed salts.
		sx := float64(cx*cfg.LotsPerChunk + ix)
		sz := float64(cz*cfg.LotsPerChunk + iz)

		b := BuildingDTO{ID: lotID, Type: townKind(seed, sx, sz), Status: StatusIdle}
		if b.Type == TypeVacantLot || b.Type == TypePark {
			w.lots[lotID] = b
			continue
		}

		lot := hash01(seed*7919, sx*13.37+sz*1.37)
		b.Floors = 1 + int(lot*float64(maxFloorsFor(b.Type)))
		b.Condition = clamp(0.25+0.7*hash01(seed*104729, sx*3.1+sz*7.7), 0, 0.999)
		b.Heritage = hash01(seed*1299709, sx*1.7+sz*5.3)
		b.Progress = 0
		b.Needs = b.Condition >= cfg.NeedsThreshold
		b.Residents = residentsFor(b)
		b.WillBase = willBase(cfg, b)
		w.lots[lotID] = b
	}
	return w
}

func (w *World) Chunk(cx, cz int) ChunkDTO {
	n := w.cfg.LotsPerChunk
	all := make([]BuildingDTO, 0, n*n)
	for iz := 0; iz < n; iz++ {
		for ix := 0; ix < n; ix++ {
			all = append(all, w.lots[composeLot(cx, cz, ix, iz, w.cfg)])
		}
	}
	return ChunkDTO{CX: cx, CZ: cz, Buildings: all}
}

func (w *World) Building(lotID int) (BuildingDTO, bool) {
	if lotID < 0 || lotID >= len(w.lots) {
		return BuildingDTO{}, false
	}
	return w.lots[lotID], true
}

func (w *World) SetBuilding(lotID int, b BuildingDTO) {
	if lotID >= 0 && lotID < len(w.lots) {
		w.lots[lotID] = b
	}
}

// maxFloorsFor keeps per-type height ranges simple and explicit.
func maxFloorsFor(t int) int {
	switch t {
	case TypePanelka, TypeKhrushchevka:
		return 9
	case TypeBrickHouse:
		return 5
	case TypeFactory:
		return 4
	case TypeMansion, TypeShop:
		return 3
	default:
		return 1
	}
}

// residentsFor computes the inhabitants of a building; parks and vacant
// lots have none.
func residentsFor(b BuildingDTO) int {
	base := residentsBaseFor(b.Type)
	n := int(float64(base) * (0.6 + b.Condition*0.4))
	return max(n, 0)
}

func residentsBaseFor(t int) int {
	switch t {
	case TypePanelka:
		return 60
	case TypeBrickHouse:
		return 40
	case TypeKhrushchevka:
		return 30
	case TypeMansion:
		return 12
	case TypeFactory:
		return 8
	case TypeShop:
		return 3
	default:
		return 0
	}
}

// willBase is the default citizens' opinion (0..1) about renovating this
// building, before any dialogue influence. Ruined buildings welcome change,
// heritage buildings resist, more residents tilt toward care.
func willBase(cfg Config, b BuildingDTO) float64 {
	base := residentsBaseFor(b.Type)
	fam := 0.0
	if base > 0 {
		fam = float64(b.Residents) / float64(base) // 0..~1
	}
	proto := 0.5 + 0.2*b.Condition - 0.2*b.Heritage + 0.1*fam
	return clamp(proto, cfg.WillLow, cfg.WillHigh)
}

// decomposeLot: linear lotId -> (cx, cz, ix, iz), row-major:
//
//	lotId = ((cz * ChunksPerAxis + cx) * LotsPerChunk + ix) * LotsPerChunk + iz
//
// NOTE: this fixes the buggy formulas in docs/API.md §4 (cz there only
// depended on the low bits of lotId); the doc will be corrected.
func decomposeLot(lotID int, cfg Config) (cx, cz, ix, iz int) {
	cell := lotID / (cfg.LotsPerChunk * cfg.LotsPerChunk) // linear chunk index
	cz = cell / cfg.ChunksPerAxis
	cx = cell % cfg.ChunksPerAxis
	rest := lotID % (cfg.LotsPerChunk * cfg.LotsPerChunk)
	ix = rest / cfg.LotsPerChunk
	iz = rest % cfg.LotsPerChunk
	return cx, cz, ix, iz
}

func composeLot(cx, cz, ix, iz int, cfg Config) int {
	cell := cz*cfg.ChunksPerAxis + cx
	return (cell*cfg.LotsPerChunk+ix)*cfg.LotsPerChunk + iz
}

// townKind distributes building types: parks clumped by large-scale noise,
// factories/shops sparse by hash, houses by frequency bands.
func townKind(seed int64, x, z float64) int {
	r01 := hash01(seed, x*3.3+z*7.1)
	park := valueNoise(seed, x/4.0, z/4.0)
	switch {
	case park < 0.22:
		return TypePark
	case r01 < 0.03:
		return TypeFactory
	case r01 < 0.12:
		return TypeShop
	case r01 < 0.35:
		return TypePanelka
	case r01 < 0.55:
		return TypeBrickHouse
	case r01 < 0.75:
		return TypeKhrushchevka
	case r01 < 0.82:
		return TypeMansion
	default:
		return TypeVacantLot
	}
}

// clamp fits v into [lo, hi].
func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(v, hi))
}

// hash01 is a deterministic 0..1 sample from (seed, coordinate).
func hash01(seed int64, x float64) float64 {
	return valueNoise(seed, x, 0.5)
}

// valueNoise is deterministic value noise over hashed lattice points,
// bilinearly interpolated — smooth terrain for distributions.
func valueNoise(seed int64, x, y float64) float64 {
	xi, yi := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(xi), y-float64(yi)

	n00 := lattice01(seed, xi, yi)
	n10 := lattice01(seed, xi+1, yi)
	n01 := lattice01(seed, xi, yi+1)
	n11 := lattice01(seed, xi+1, yi+1)

	nx0 := n00*(1-fx) + n10*fx
	nx1 := n01*(1-fx) + n11*fx
	return nx0*(1-fy) + nx1*fy
}

// lattice01 maps an integer lattice point + seed to deterministic 0..1.
func lattice01(seed int64, x, y int) float64 {
	h := mix64(uint64(x)<<32|uint64(uint32(y)), uint64(seed))
	return float64(h&0x0000_ffff) / float64(0x0000_ffff)
}

// mix64 is a splitmix-style avalanche hash.
func mix64(a, b uint64) uint64 {
	a += b * 0x9e3779b97f4a7c15
	a ^= a >> 30
	a *= 0xbf58476d1ce4e5b9
	a ^= a >> 27
	a *= 0x94d049bb133111eb
	a ^= a >> 31
	return a
}
