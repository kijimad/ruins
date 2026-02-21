package dungeon

import (
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/mapplanner"
)

// SelectPlanner はPlannerPoolから重み付き抽選でPlannerTypeを選択する
func SelectPlanner(def Definition, rng *rand.Rand) (mapplanner.PlannerType, error) {
	pool := def.PlannerPool
	if len(pool) == 0 {
		return mapplanner.PlannerType{}, fmt.Errorf("PlannerPoolが空です: %s", def.Name)
	}

	totalWeight := 0
	for _, pw := range pool {
		totalWeight += pw.Weight
	}
	if totalWeight == 0 {
		return mapplanner.PlannerType{}, fmt.Errorf("PlannerPoolの総重みが0です: %s", def.Name)
	}

	r := rng.IntN(totalWeight)
	cumulative := 0
	for _, pw := range pool {
		cumulative += pw.Weight
		if r < cumulative {
			return pw.PlannerType, nil
		}
	}

	return mapplanner.PlannerType{}, fmt.Errorf("PlannerTypeの選択に失敗しました（到達不可能）: %s", def.Name)
}
