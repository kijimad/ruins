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
	return SelectByWeightFunc(
		dt.Entries,
		func(e DropTableEntry) float64 { return e.Weight },
		func(e DropTableEntry) string { return e.Material },
		rng,
	)
}
