package raw

import (
	"math/rand/v2"
)

// SelectByWeightFunc は重み付き抽選で値を選択する汎用関数
// getWeight: 各要素から重みを取得する関数
// getValue: 各要素から戻り値を取得する関数
func SelectByWeightFunc[T any, V any](items []T, getWeight func(T) float64, getValue func(T) V, rng *rand.Rand) (V, error) {
	var zero V
	if len(items) == 0 {
		return zero, nil
	}

	var totalWeight float64
	for _, item := range items {
		totalWeight += getWeight(item)
	}

	if totalWeight == 0 {
		return zero, nil
	}

	randomValue := rng.Float64() * totalWeight

	var cumulativeWeight float64
	for _, item := range items {
		cumulativeWeight += getWeight(item)
		if randomValue < cumulativeWeight {
			return getValue(item), nil
		}
	}

	return zero, nil
}
