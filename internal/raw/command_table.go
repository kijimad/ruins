package raw

import (
	"math/rand/v2"
)

// CommandTable はコマンドテーブル
type CommandTable struct {
	Name    string
	Entries []CommandTableEntry `toml:"entries"`
}

// CommandTableEntry はコマンドテーブルのエントリ
type CommandTableEntry struct {
	Weapon string
	Weight float64
}

// SelectByWeight は重みで選択する
func (ct CommandTable) SelectByWeight(rng *rand.Rand) (string, error) {
	items := make([]WeightedItem, len(ct.Entries))
	for i, entry := range ct.Entries {
		items[i] = WeightedItem{Value: entry.Weapon, Weight: entry.Weight}
	}
	return SelectByWeight(items, rng)
}
