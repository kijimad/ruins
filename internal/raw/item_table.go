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
	MinDepth int
	MaxDepth int
}

// SelectByWeight は重みで選択する
func (it ItemTable) SelectByWeight(rng *rand.Rand, depth int) (string, error) {
	// 深度範囲内のエントリのみをフィルタリング
	var filtered []ItemTableEntry
	for _, entry := range it.Entries {
		if depth < entry.MinDepth || depth > entry.MaxDepth {
			continue
		}
		filtered = append(filtered, entry)
	}

	return SelectByWeightFunc(
		filtered,
		func(e ItemTableEntry) float64 { return e.Weight },
		func(e ItemTableEntry) string { return e.ItemName },
		rng,
	)
}
