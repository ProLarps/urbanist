package gameplay

// Renovation scenarios of docs/API.md §8: cost multiplier, one-time
// profit rate and the citizens'-will shift each project applies.
type scenario struct {
	mult       float64
	profitRate float64
	willShift  float64
}

const (
	baseCost     = 30 // BASE_COST per project (§8 formula)
	costPerFloor = 12 // floors term of the same formula
)

var scenarios = map[string]scenario{
	"repair":     {mult: 0.4, profitRate: 0.40, willShift: +0.05},
	"rebuild":    {mult: 1.0, profitRate: 0.90, willShift: -0.10},
	"revitalize": {mult: 1.5, profitRate: 1.30, willShift: +0.15},
	"demolish":   {mult: 2.0, profitRate: 1.70, willShift: -0.25},
}

// projectCost mirrors the §8 formula with a fixed mid-tier building
// (floors=3): total = (BASE + floors*COST_PER_FLOOR) * scenarioMult.
func projectCost(name string) (float64, bool) {
	s, ok := scenarios[name]
	if !ok {
		return 0, false
	}
	return (baseCost + 3*costPerFloor) * s.mult, true
}

// cheapestScenarioCost backs the "bankrupt" defeat check: money below the
// price of the cheapest possible renovation means no progress is left.
func cheapestScenarioCost() float64 {
	best, ok := projectCost("repair")
	for name := range scenarios {
		if c, ok2 := projectCost(name); ok2 && c < best {
			best = c
		}
	}
	if !ok {
		return baseCost
	}
	return best
}
