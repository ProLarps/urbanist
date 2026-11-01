package ws

// Config holds every tunable gameplay constant. Nothing gameplay-relevant is
// hardcoded: tweak here (or via config file later) without touching logic.
type Config struct {
	// World shape (see docs/API.md §4).
	ChunksPerAxis int // 10 => 100 chunks per room
	LotsPerChunk  int // 8x8 => 64 lots => 6400 buildings
	LotSize       int // world units per lot side
	ViewChunks    int // 3 => 3x3=9 chunks loaded around the camera

	// Economy.
	StartMoney      float64 // money on join
	RatingStart     float64 // rating multiplier on join
	RatingMin       float64 // below this -> automatic loss
	RatingStep      float64 // max rating delta per renovation (scaled by will)
	BaseCost        float64 // renovation cost formula base
	CostPerFloor    float64 // ... per floor
	MaxDiscount     float64 // player-supplied discount cap
	NeedsThreshold  float64 // condition >= this => building needs renovation
	WillLow         float64 // will floor
	WillHigh        float64 // will ceiling
	HandednessCoeff float64 // how strongly residents outnumber influence will

	// Rounds.
	RoundSeconds      int     // deadline per assigned chunk
	ChunksPerGrant    int     // N options offered each round (N!=hardcode: config)
	MaxCyclesHeld     int     // pending chunk survives this many cycles, then skip
	TimeoutRatingLoss float64 // fixed rating penalty on timeout
	TimeoutMoneyRate  float64 // fraction of current money lost on timeout
	SkipRatingLoss    float64 // penalty when a chunk is finally skipped
	SkipMoneyRate     float64 // fraction of money lost on skip
	MinGraceSeconds   int     // minimum time between grants

	// Movement anti-cheat.
	MoveUnitsPerSec float64 // max player displacement per second

	// Auth.
	TokenTTL  int64 // seconds (7 days)
	SecretDir string

	// Frequencies (Hz) for tick loops.
	TickHz       float64 // player updates & timers
	NpcHz        float64 // NPC broadcasts
	SaveHz       float64 // periodic player save
	DeltaSeconds float64 // modified-buildings -> chunkDelta flush
}

// DefaultConfig mirrors docs/API.md §8 values.
func DefaultConfig() Config {
	return Config{
		ChunksPerAxis: 10,
		LotsPerChunk:  8,
		LotSize:       4,
		ViewChunks:    3,

		StartMoney:      500,
		RatingStart:     1.0,
		RatingMin:       0.25,
		RatingStep:      0.05,
		BaseCost:        30,
		CostPerFloor:    12,
		MaxDiscount:     0.3,
		NeedsThreshold:  0.5,
		WillLow:         0.05,
		WillHigh:        0.95,
		HandednessCoeff: 0.5,

		RoundSeconds:      300,
		ChunksPerGrant:    2, // NOT hardcoded: change via config
		MaxCyclesHeld:     4,
		TimeoutRatingLoss: 0.10,
		TimeoutMoneyRate:  0.20,
		SkipRatingLoss:    0.15,
		SkipMoneyRate:     0.30,
		MinGraceSeconds:   0,

		MoveUnitsPerSec: 6,

		TokenTTL:  7 * 24 * 3600,
		SecretDir: ".",

		TickHz:       10,
		NpcHz:        4,
		SaveHz:       1.0 / 30.0,
		DeltaSeconds: 2,
	}
}
