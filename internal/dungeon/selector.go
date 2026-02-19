package dungeon

import (
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/mapplanner"
)

// SelectPlanner はPlannerPoolから重み付き抽選でPlannerTypeを選択する
// 選択されたPlannerTypeにDefinitionのEnemyTableNameとItemTableNameを設定して返す
func SelectPlanner(def Definition, seed uint64) (mapplanner.PlannerType, error) {
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

	rng := rand.New(rand.NewPCG(seed, seed))
	r := rng.IntN(totalWeight)
	cumulative := 0
	var selected mapplanner.PlannerType
	for _, pw := range pool {
		cumulative += pw.Weight
		if r < cumulative {
			selected = pw.PlannerType
			break
		}
	}

	selected.EnemyTableName = def.EnemyTableName
	selected.ItemTableName = def.ItemTableName
	return selected, nil
}
