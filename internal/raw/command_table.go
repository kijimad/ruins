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
func (ct CommandTable) SelectByWeight() string {
	var totalWeight float64
	for _, entry := range ct.Entries {
		totalWeight += entry.Weight
	}
	randomValue := rand.Float64() * totalWeight

	// 累積ウェイトで判定
	var cumulativeWeight float64
	for _, entry := range ct.Entries {
		cumulativeWeight += entry.Weight
		if randomValue < cumulativeWeight {
			return entry.Weapon
		}
	}

	return ""
}
