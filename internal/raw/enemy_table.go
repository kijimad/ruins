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
	MinDepth  int
	MaxDepth  int
}

// SelectByWeight は重みで選択する
func (et EnemyTable) SelectByWeight(rng *rand.Rand, depth int) (string, error) {
	// 深度範囲内のエントリのみをフィルタリング
	filtered := make([]EnemyTableEntry, 0, len(et.Entries))
	for _, entry := range et.Entries {
		if depth < entry.MinDepth || depth > entry.MaxDepth {
			continue
		}
		filtered = append(filtered, entry)
	}

	return SelectByWeightFunc(
		filtered,
		func(e EnemyTableEntry) float64 { return e.Weight },
		func(e EnemyTableEntry) string { return e.EnemyName },
		rng,
	)
}
