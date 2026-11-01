// Package config holds every gameplay tunable. The admin panel rewrites the
// GlobalConfig copy in package gameplay; running rooms keep the snapshot they
// took when entering run mode, so live games never see admin changes.
package config

const (
	RENNOVATE_PANEL_FLAT = iota
	RENNOVATE_PARK
	RENNOVATE_ALLEY
	RENNOVATE_OLD_HOUSE
	RENNOVATE_LAKE
	RENNOVATIONS_TOTAL // must be last
)

const (
	BUILDING_RENNOVATION = iota // empty cell waiting for a renovation
	BUILDING_FLAT
	BUILDING_SCYSCRAPER
	BUILDING_SMALL_HOUSE
	BUILDINGS_TOTAL // must be last
)

// World shape (docs/API.md §4, adjusted for district generation).
const (
	ChunksPerAxis = 10 // 10x10 chunks per room
	ViewChunks    = 3  // 3x3 chunks loaded around the camera

	ChunkMin = 8 // district side limits for the admin panel
	ChunkMax = 64
)

type Config struct {
	// District generation (§ chunks are districts).
	RennovationAppearChances map[int]int `json:"rennovationAppearChances"` // weight per renovation type
	RennovationMinSizes      map[int]int `json:"rennovationMinSizes"`
	RennovationMaxSizes      map[int]int `json:"rennovationMaxSizes"`
	ChunkSize                int         `json:"chunkSize"` // district side: ChunkSize x ChunkSize lots

	// Economy (docs/API.md §8): money and rating per player.
	StartMoney     float64 `json:"startMoney"`     // money when entering a room
	RatingStart    float64 `json:"ratingStart"`    // rating multiplier on join
	RatingMin      float64 `json:"ratingMin"`      // below this the player loses automatically
	RatingStep     float64 `json:"ratingStep"`     // max rating delta per finished renovation
	NeedsThreshold float64 `json:"needsThreshold"` // condition >= this means the lot needs renovation

	// Rounds.
	RoundSeconds      int     `json:"roundSeconds"`      // deadline per assigned district
	ChunksPerGrant    int     `json:"chunksPerGrant"`    // district options offered each cycle
	MaxCyclesHeld     int     `json:"maxCyclesHeld"`     // pending district survives this many cycles
	TimeoutRatingLoss float64 `json:"timeoutRatingLoss"` // fixed rating penalty on timeout
	TimeoutMoneyRate  float64 `json:"timeoutMoneyRate"`  // fraction of money lost on timeout
	SkipRatingLoss    float64 `json:"skipRatingLoss"`    // penalty when a district is finally skipped
	SkipMoneyRate     float64 `json:"skipMoneyRate"`     // fraction of money lost on skip

	// Movement anti-cheat (§5): max player displacement per second.
	MoveUnitsPerSec float64 `json:"moveUnitsPerSec"`
}

// DefaultConfig mirrors docs/API.md §8; the admin panel may override it.
var DefaultConfig = Config{
	ChunkSize: 16,
	RennovationAppearChances: map[int]int{
		RENNOVATE_PANEL_FLAT: 35,
		RENNOVATE_OLD_HOUSE:  30,
		RENNOVATE_PARK:       20,
		RENNOVATE_ALLEY:      10,
		RENNOVATE_LAKE:       5,
	},
	RennovationMinSizes: map[int]int{
		RENNOVATE_PANEL_FLAT: 4,
		RENNOVATE_OLD_HOUSE:  3,
		RENNOVATE_PARK:       4,
		RENNOVATE_ALLEY:      3,
		RENNOVATE_LAKE:       3,
	},
	RennovationMaxSizes: map[int]int{
		RENNOVATE_PANEL_FLAT: 8,
		RENNOVATE_OLD_HOUSE:  10,
		RENNOVATE_PARK:       12,
		RENNOVATE_ALLEY:      14,
		RENNOVATE_LAKE:       8,
	},

	StartMoney:     500,
	RatingStart:    1.0,
	RatingMin:      0.25,
	RatingStep:     0.05,
	NeedsThreshold: 0.5,

	RoundSeconds:      300,
	ChunksPerGrant:    2,
	MaxCyclesHeld:     4,
	TimeoutRatingLoss: 0.10,
	TimeoutMoneyRate:  0.20,
	SkipRatingLoss:    0.15,
	SkipMoneyRate:     0.30,

	MoveUnitsPerSec: 6,
}
