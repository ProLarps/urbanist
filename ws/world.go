package ws

import (
	"math"
	"sync"
)

// Tile kinds. Must match the front-end tile atlas.
const (
	TileGrass = iota
	TileRoad
	TileHouse
	TileParkTree
	TileWater
	TileParkPath
)

// chunkSize is the side length of a square chunk in tiles.
const chunkSize = 16

// districtSeed is fixed so chunks are identical for everyone and regenerate
// identically across server restarts.
const districtSeed = 0x5EED1A95

// World is the room's old district. Base terrain is generated
// deterministically from the district seed; approved renovations are kept as
// an in-memory overlay that repaints affected chunks.
type World struct {
	mu sync.RWMutex
	// chunk coords -> winning proposal id, set when renovations are applied.
	overlays map[[2]int]string
}

func NewWorld() *World {
	return &World{overlays: map[[2]int]string{}}
}

// ChunkAt returns the fully generated chunk for coordinates x,y: the base
// terrain plus any approved renovation overlay for that chunk.
func (w *World) ChunkAt(x, y int) ChunkData {
	w.mu.RLock()
	id, modified := w.overlays[[2]int{x, y}]
	w.mu.RUnlock()

	tiles := make([]Tile, 0, chunkSize*chunkSize)
	for ty := 0; ty < chunkSize; ty++ {
		for tx := 0; tx < chunkSize; tx++ {
			worldX := x*chunkSize + tx
			worldY := y*chunkSize + ty
			kind := terrainKind(worldX, worldY)
			if modified {
				kind = renovationKind(id, kind, worldX, worldY)
			}
			tiles = append(tiles, Tile{Kind: kind})
		}
	}
	return ChunkData{X: x, Y: y, Tiles: tiles}
}

// ApplyProposal records a winning renovation over the affected chunks.
func (w *World) ApplyProposal(p Proposal, coords [][2]int) {
	w.mu.Lock()
	for _, xy := range coords {
		w.overlays[xy] = p.ID
	}
	w.mu.Unlock()
}

// renovationKind repaints base terrain per winning proposal. Roads and water
// always stay: you cannot fill the canal with blocks of flats.
func renovationKind(proposalID string, base, x, y int) int {
	keep := base == TileRoad || base == TileWater
	if keep {
		return base
	}

	switch proposalID {
	case "residential":
		// Dense housing with the occasional courtyard.
		if (x/2+y/2)%3 == 0 {
			return TileGrass
		}
		return TileHouse
	case "flats":
		// Big blocky slabs.
		return TileHouse
	case "park":
		if (x+y)%3 == 0 {
			return TileParkPath
		}
		return TileParkTree
	default:
		return base
	}
}

// terrainKind produces one tile of the old-district base map:
// water in low-noise areas, a road grid every 8 tiles, parks on noise
// patches, houses near roads, grass elsewhere.
func terrainKind(x, y int) int {
	// City-owned road grid.
	if x%8 == 4 || y%8 == 4 {
		return TileRoad
	}

	elev := valueNoise(float64(x)/10.0, float64(y)/10.0)               // large features: water bodies
	green := valueNoise(float64(x*3+elevSeed), float64(y*3-elevSeed))  // small patches: parks
	dens := valueNoise(float64(x*7+houseSeed), float64(y*7-houseSeed)) // house density

	switch {
	case elev < waterLevel:
		return TileWater
	case green > parkLevel && x%8 > 1 && y%8 > 1:
		if (x+y)%2 == 0 {
			return TileParkPath
		}
		return TileParkTree
	case dens > houseLevel && nearRoad(x, y):
		return TileHouse
	default:
		return TileGrass
	}
}

const (
	waterLevel = 0.34
	parkLevel  = 0.68
	houseLevel = 0.55
	elevSeed   = 101
	houseSeed  = 409
)

// nearRoad reports whether a tile sits next to a road (roads occupy the
// x%8 == 4 && y%8 == 4 lines of the district grid).
func nearRoad(x, y int) bool {
	return mod8(x) == 3 || mod8(x) == 5 || mod8(y) == 3 || mod8(y) == 5
}

func mod8(v int) int {
	if v < 0 {
		return ((v % 8) + 8) % 8
	}
	return v % 8
}

// valueNoise is a cheap deterministic value noise over a lattice of hashed
// points, bilinearly interpolated — good enough for tile art.
func valueNoise(x, y float64) float64 {
	xi, yi := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(xi), y-float64(yi)

	n00 := lattice(xi, yi)
	n10 := lattice(xi+1, yi)
	n01 := lattice(xi, yi+1)
	n11 := lattice(xi+1, yi+1)

	nx0 := n00*(1-fx) + n10*fx
	nx1 := n01*(1-fx) + n11*fx
	return nx0*(1-fy) + nx1*fy
}

func lattice(x, y int) float64 {
	h := uint64(uint32(x))<<32 | uint64(uint32(y))
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	h = mix(h, districtSeed)
	return float64(h&0x0000_ffff) / float64(0x0000_ffff)
}

func mix(a, b uint64) uint64 {
	a *= b
	return a ^ (a >> 31)
}
