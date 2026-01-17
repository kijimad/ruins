package raw

import "math/rand/v2"

// EnemyTable は敵出現テーブル
type EnemyTable struct {
	Name    string
	Entries []EnemyTableEntry `toml:"entries"`
}

// EnemyTableEntry は敵テーブルのエントリ
type EnemyTableEntry struct {
	EnemyName string
	Weight    float64
	MinDepth  int // 最小出現深度（0は制限なし）
	MaxDepth  int // 最大出現深度（0は制限なし）
}

// SelectByWeight は重みで選択する
func (et EnemyTable) SelectByWeight(rng *rand.Rand, depth int) string {
	// 深度範囲内のエントリのみをフィルタリング
	validEntries := make([]EnemyTableEntry, 0, len(et.Entries))
	for _, entry := range et.Entries {
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
			return entry.EnemyName
		}
	}

	return ""
}
