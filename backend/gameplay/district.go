package gameplay

import (
	"math/rand/v2"

	"github.com/prolarps/urbanist/backend/config"
)

type District struct {
	Rennovation int
	Data        []uint8 // Flat array: index = y * ChunkSize + x
}

func GenerateDistrict(cfg *config.Config, rng *rand.Rand) *District {
	res := &District{
		Rennovation: chooseDistrictRennovation(cfg, rng),
	}
	res.generateMap(cfg, rng)
	return res
}

func chooseDistrictRennovation(cfg *config.Config, rng *rand.Rand) int {
	var total int
	for rennovation := range config.RENNOVATIONS_TOTAL {
		total += cfg.RennovationAppearChances[rennovation]
	}

	// IntN returns [0, total), which perfectly maps to our cumulative integer weights
	roll := rng.IntN(total)

	var cumulative int
	for rennovation := range config.RENNOVATIONS_TOTAL {
		cumulative += cfg.RennovationAppearChances[rennovation]
		if roll < cumulative {
			return rennovation
		}
	}

	return config.RENNOVATE_PANEL_FLAT // fallback
}

func (d *District) generateMap(cfg *config.Config, rng *rand.Rand) {
	size := cfg.ChunkSize
	d.Data = make([]uint8, size*size)

	// 1. Fill with the surrounding building type
	surroundingBuilding := getSurroundingBuildingType(d.Rennovation)
	for i := range d.Data {
		d.Data[i] = uint8(surroundingBuilding)
	}

	// 2. Calculate and clamp renovation zone size
	minSize := cfg.RennovationMinSizes[d.Rennovation]
	maxSize := cfg.RennovationMaxSizes[d.Rennovation]

	if minSize > size {
		minSize = size
	}
	if maxSize > size {
		maxSize = size
	}
	if minSize > maxSize {
		minSize, maxSize = maxSize, minSize
	}

	w := rng.IntN(maxSize-minSize+1) + minSize
	h := rng.IntN(maxSize-minSize+1) + minSize

	// 3. Find a random top-left position that guarantees it fits in bounds
	maxX := size - w
	maxY := size - h

	x := rng.IntN(maxX + 1)
	y := rng.IntN(maxY + 1)

	// 4. Carve out the main renovation area
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			idx := row*size + col
			d.Data[idx] = config.BUILDING_RENNOVATION
		}
	}
}

func getSurroundingBuildingType(renovation int) int {
	switch renovation {
	case config.RENNOVATE_PANEL_FLAT:
		return config.BUILDING_FLAT
	case config.RENNOVATE_OLD_HOUSE:
		return config.BUILDING_SMALL_HOUSE
	case config.RENNOVATE_PARK:
		return config.BUILDING_SMALL_HOUSE
	case config.RENNOVATE_ALLEY:
		return config.BUILDING_FLAT
	case config.RENNOVATE_LAKE:
		return config.BUILDING_SMALL_HOUSE
	default:
		return config.BUILDING_FLAT
	}
}
