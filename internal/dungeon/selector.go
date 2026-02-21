package dungeon

import (
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/raw"
)

// SelectPlanner はPlannerPoolから重み付き抽選でPlannerTypeを選択する
func SelectPlanner(def Definition, rng *rand.Rand) (mapplanner.PlannerType, error) {
	if len(def.PlannerPool) == 0 {
		return mapplanner.PlannerType{}, fmt.Errorf("PlannerPoolが空です: %s", def.Name)
	}

	result, err := raw.SelectByWeightFunc(
		def.PlannerPool,
		func(pw PlannerWeight) float64 { return float64(pw.Weight) },
		func(pw PlannerWeight) mapplanner.PlannerType { return pw.PlannerType },
		rng,
	)
	if err != nil {
		return mapplanner.PlannerType{}, err
	}

	if result.Name == "" {
		return mapplanner.PlannerType{}, fmt.Errorf("PlannerPoolの総重みが0です: %s", def.Name)
	}

	return result, nil
}
