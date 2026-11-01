package gameplay

import (
	"math/rand/v2"

	"github.com/prolarps/urbanist/backend/config"
)

// point addresses one chunk of the room world grid.
type point struct{ cx, cz int }

// World is one room's city: ChunksPerAxis x ChunksPerAxis districts,
// generated deterministically from the room seed (docs/API.md §13).
type World struct {
	chunks map[point]*District
	cfg    config.Config
}

// NewWorld generates all districts deterministically from seed.
func NewWorld(seed int64, cfg config.Config) *World {
	rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed>>32)))
	w := &World{chunks: make(map[point]*District), cfg: cfg}
	for cz := range config.ChunksPerAxis {
		for cx := range config.ChunksPerAxis {
			w.chunks[point{cx, cz}] = GenerateDistrict(&cfg, rng)
		}
	}
	return w
}

// Chunk exposes a full district; letter case is intentional: rooms wrap it.
func (w *World) Chunk(cx, cz int) (ChunkDTO, bool) {
	d, ok := w.chunks[point{cx, cz}]
	if !ok {
		return ChunkDTO{}, false
	}
	size := w.cfg.ChunkSize
	cells := make([]uint8, len(d.Data))
	copy(cells, d.Data)
	return ChunkDTO{CX: cx, CZ: cz, Rennovation: d.Rennovation, Size: size, Cells: cells}, true
}
