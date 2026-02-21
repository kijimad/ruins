package raw

import (
	"fmt"
	"math/rand/v2"
)

// WeightedItem は重み付き選択の共通構造体
type WeightedItem struct {
	Value  string
	Weight float64
}

// SelectByWeight は重み付き抽選で値を選択する
func SelectByWeight(items []WeightedItem, rng *rand.Rand) (string, error) {
	if len(items) == 0 {
		return "", nil
	}

	var totalWeight float64
	for _, item := range items {
		totalWeight += item.Weight
	}

	if totalWeight == 0 {
		return "", nil
	}

	randomValue := rng.Float64() * totalWeight

	var cumulativeWeight float64
	for _, item := range items {
		cumulativeWeight += item.Weight
		if randomValue < cumulativeWeight {
			return item.Value, nil
		}
	}

	return "", fmt.Errorf("重み付き選択に失敗しました（到達不可能）")
}
