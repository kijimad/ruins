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
func (dt DropTable) SelectByWeight(rng *rand.Rand) (string, error) {
	items := make([]WeightedItem, len(dt.Entries))
	for i, entry := range dt.Entries {
		items[i] = WeightedItem{Value: entry.Material, Weight: entry.Weight}
	}
	return SelectByWeight(items, rng)
}
