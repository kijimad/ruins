package raw

import "math/rand/v2"

// DropTable はドロップテーブル
type DropTable struct {
	Name    string
	Entries []DropTableEntry `toml:"entries"`
}

// DropTableEntry はドロップテーブルのエントリ
type DropTableEntry struct {
	Material string
	Weight   float64
}

// SelectByWeight は重みで選択する
func (dt DropTable) SelectByWeight(rng *rand.Rand) string {
	var totalWeight float64
	for _, entry := range dt.Entries {
		totalWeight += entry.Weight
	}

	if totalWeight == 0 {
		return ""
	}

	randomValue := rng.Float64() * totalWeight

	// 累積ウェイトで判定
	var cumulativeWeight float64
	for _, entry := range dt.Entries {
		cumulativeWeight += entry.Weight
		if randomValue < cumulativeWeight {
			return entry.Material
		}
	}

	return ""
}
