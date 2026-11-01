package gameplay

import (
	"slices"
	"strings"
)

// sortScores orders scoreboard rows by score = money x rating, then by name
// as a stable tie-breaker. Sorted copies keep the room lock time tiny.
func sortScores(rows []ScoreRow) []ScoreRow {
	slices.SortStableFunc(rows, func(a, b ScoreRow) int {
		switch {
		case a.Score < b.Score:
			return 1
		case a.Score > b.Score:
			return -1
		default:
			return strings.Compare(a.Name, b.Name)
		}
	})
	return rows
}
