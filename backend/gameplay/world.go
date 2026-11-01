package gameplay

import (
	"math/rand/v2"

	"github.com/prolarps/urbanist/backend/config"
)

// chunkPos addresses one chunk of the room world grid.
type chunkPos struct{ cx, cz int }

// World is one room's city: ChunksPerAxis x ChunksPerAxis districts,
// generated deterministically from the room seed (docs/API.md §13).
type World struct {
	chunks map[chunkPos]*District
	cfg    config.Config
}

// NewWorld generates all districts deterministically from seed.
func NewWorld(seed int64, cfg config.Config) *World {
	rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed>>32)))
	w := &World{chunks: make(map[chunkPos]*District), cfg: cfg}
	for cz := range config.ChunksPerAxis {
		for cx := range config.ChunksPerAxis {
			w.chunks[chunkPos{cx, cz}] = GenerateDistrict(&cfg, rng)
		}
	}
	return w
}

// Chunk exposes a full district; letter case is intentional: rooms wrap it.
func (w *World) Chunk(cx, cz int) (ChunkDTO, bool) {
	d, ok := w.chunks[chunkPos{cx, cz}]
	if !ok {
		return ChunkDTO{}, false
	}
	size := w.cfg.ChunkSize
	cells := make([]uint8, len(d.Data))
	copy(cells, d.Data)
	return ChunkDTO{CX: cx, CZ: cz, Rennovation: d.Rennovation,
		Will: d.Will, Size: size, Cells: cells}, true
}

// grantable reports whether (cx,cz) is inside the world grid.
func grantable(cx, cz int) bool {
	const max = config.ChunksPerAxis - 1
	return cx >= 0 && cx <= max && cz >= 0 && cz <= max
}

// freeNeighbours lists unowned chunks adjacent to any of the player's held
// districts (§8 step 3). The caller holds r.mu; ownership lives in Room.
func (w *World) freeNeighbours(held map[chunkPos]bool, taken func(chunkPos) bool) []chunkPos {
	seen := map[chunkPos]bool{}
	var out []chunkPos
	for p := range held {
		for _, n := range []chunkPos{{p.cx + 1, p.cz}, {p.cx - 1, p.cz},
			{p.cx, p.cz + 1}, {p.cx, p.cz - 1}} {
			if seen[n] || !grantable(n.cx, n.cz) || taken(n) {
				continue
			}
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}
