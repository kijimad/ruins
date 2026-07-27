package lifecycle

import (
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	w "github.com/kijimaD/ruins/internal/world"
)

// DestroySalvageChance は攻撃破壊時に確定枠の各産出を回収できる百分率。
// 工具なしでも素材が少しは手に入る立ち上がり用の値で、工具分解より明確に不利にする
const DestroySalvageChance = 30

// YieldStack は分解産出のアイテム名と個数の組
type YieldStack struct {
	Name  string
	Count int
}

// RollDisassemblyYields は分解定義から産出を抽選する。
// full が真なら工具による通常分解として扱い、確率枠にスキルと工具グレードの補正を
// 加算し、ボーナス枠の条件判定も行う。偽なら攻撃破壊の回収として扱い、確定枠のみを
// DestroySalvageChance で抽選して確率枠とボーナス枠は出さない
func RollDisassemblyYields(rng *rand.Rand, def *oapi.Disassembly, skillValue int, toolGrade int, full bool) []YieldStack {
	var stacks []YieldStack

	for _, y := range def.Yields {
		if y.Chance == nil {
			// chance 省略は確定枠。破壊回収では低確率でしか出ない
			if !full && rng.IntN(100) >= DestroySalvageChance {
				continue
			}
			stacks = append(stacks, YieldStack{Name: y.Name, Count: rollAmount(rng, y.Amount, y.AmountMax)})
			continue
		}
		if !full {
			continue
		}
		// グレード補正は1を基準とし、想定外の0以下が来ても確率を下げない
		chance := min(int(*y.Chance)+skillValue+max(0, (toolGrade-1)*10), 100)
		if rng.IntN(100) < chance {
			stacks = append(stacks, YieldStack{Name: y.Name, Count: rollAmount(rng, y.Amount, y.AmountMax)})
		}
	}

	if full && def.Bonus != nil {
		for _, b := range *def.Bonus {
			// 指定された条件はすべて満たす必要がある。スキーマ上どちらか一方は
			// 必須だが、両方欠けた定義は無効として出さない
			if b.MinSkill == nil && b.MinGrade == nil {
				continue
			}
			if b.MinSkill != nil && skillValue < int(*b.MinSkill) {
				continue
			}
			if b.MinGrade != nil && toolGrade < int(*b.MinGrade) {
				continue
			}
			stacks = append(stacks, YieldStack{Name: b.Name, Count: rollAmount(rng, b.Amount, b.AmountMax)})
		}
	}

	return stacks
}

// rollAmount は個数を決める。amountMax があれば amount..amountMax の一様抽選にする
func rollAmount(rng *rand.Rand, amount oapi.ItemCount, amountMax *oapi.ItemCount) int {
	minCount := int(amount)
	if amountMax == nil || int(*amountMax) <= minCount {
		return minCount
	}
	return minCount + rng.IntN(int(*amountMax)-minCount+1)
}

// SpawnDisassemblyYields は産出一覧を指定タイルへフィールドアイテムとして生成する
func SpawnDisassemblyYields(world w.World, stacks []YieldStack, x consts.Tile, y consts.Tile) error {
	for _, s := range stacks {
		if _, err := SpawnFieldItem(world, s.Name, x, y, s.Count); err != nil {
			return err
		}
	}
	return nil
}
