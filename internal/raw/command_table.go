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
	return SelectByWeightFunc(
		ct.Entries,
		func(e CommandTableEntry) float64 { return e.Weight },
		func(e CommandTableEntry) string { return e.Weapon },
		rng,
	)
}
