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
func (et EnemyTable) SelectByWeight(rng *rand.Rand, depth int) (string, error) {
	// 深度範囲内のエントリのみをフィルタリング
	var items []WeightedItem
	for _, entry := range et.Entries {
		// MinDepthチェック（0は制限なし）
		if entry.MinDepth > 0 && depth < entry.MinDepth {
			continue
		}
		// MaxDepthチェック（0は制限なし）
		if entry.MaxDepth > 0 && depth > entry.MaxDepth {
			continue
		}
		items = append(items, WeightedItem{Value: entry.EnemyName, Weight: entry.Weight})
	}

	return SelectByWeight(items, rng)
}
