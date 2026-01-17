package raw

import "math/rand/v2"

// ItemTable はアイテム出現テーブル
type ItemTable struct {
	Name    string
	Entries []ItemTableEntry `toml:"entries"`
}

// ItemTableEntry はアイテムテーブルのエントリ
type ItemTableEntry struct {
	ItemName string
	Weight   float64
	MinDepth int // 最小出現深度（0は制限なし）
	MaxDepth int // 最大出現深度（0は制限なし）
}

// SelectByWeight は重みで選択する
func (it ItemTable) SelectByWeight(rng *rand.Rand, depth int) string {
	// 深度範囲内のエントリのみをフィルタリング
	validEntries := make([]ItemTableEntry, 0, len(it.Entries))
	for _, entry := range it.Entries {
		// MinDepthチェック（0は制限なし）
		if entry.MinDepth > 0 && depth < entry.MinDepth {
			continue
		}
		// MaxDepthチェック（0は制限なし）
		if entry.MaxDepth > 0 && depth > entry.MaxDepth {
			continue
		}
		validEntries = append(validEntries, entry)
	}

	// 有効なエントリがない場合
	if len(validEntries) == 0 {
		return ""
	}

	// 総重みを計算
	var totalWeight float64
	for _, entry := range validEntries {
		totalWeight += entry.Weight
	}

	if totalWeight == 0 {
		return ""
	}

	randomValue := rng.Float64() * totalWeight

	// 累積ウェイトで判定
	var cumulativeWeight float64
	for _, entry := range validEntries {
		cumulativeWeight += entry.Weight
		if randomValue < cumulativeWeight {
			return entry.ItemName
		}
	}

	return ""
}
