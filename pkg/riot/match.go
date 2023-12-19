// Match data structure and some helper methods.

package riot

import "time"

type Match struct {
	Kills   int32
	Deaths  int32
	Assists int32
	Won     bool
	Champ   int32
	Time    time.Time
}

func (m *Match) KillDeathRatio() float64 {
	return float64(m.Kills) / float64(m.Deaths)
}

// Annoying this has to be a free function but whatever
// API abides by slices.SortFunc function
// -1 if one < two
// 0 if one == two
// 1 if one > two
func CompareMatches(one, two *Match) int {
	// First compare kills
	if one.Kills < two.Kills {
		return -1
	} else if one.Kills > two.Kills {
		return 1
	}

	// Kills are equivalent, compare KDs
	kdOne := one.KillDeathRatio()
	kdTwo := two.KillDeathRatio()
	if kdOne < kdTwo {
		return -1
	} else if kdOne > kdTwo {
		return 1
	}

	// KDs are equivalent, compare win or loss
	if one.Won == two.Won {
		// "Basically same game" at face level
		return 0
	} else if two.Won {
		// This implies basically !one.Won && two.Won
		return -1
	} else {
		// This implies basically one.Two && !two.Won
		return 1
	}
}
