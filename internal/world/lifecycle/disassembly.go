package lifecycle

import (
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
)

// DestroySalvageChance は合う道具なしで壊したときに確定枠の各産出を回収できる百分率
const DestroySalvageChance = 30

// YieldStack は分解産出のアイテム名と個数の組
type YieldStack struct {
	Name  string
	Count int
}

// RollDisassemblyYields は分解定義から産出を抽選する。
// full が真なら確定枠をすべて出し、確率枠にスキルと工具グレードの補正を加算し、ボーナス枠の
// 条件判定も行う。偽なら確定枠のみを DestroySalvageChance で抽選して確率枠とボーナス枠は出さない
func RollDisassemblyYields(rng *rand.Rand, def *rawschema.Disassembly, skillValue int, toolGrade int, full bool) ([]YieldStack, error) {
	var stacks []YieldStack

	for _, y := range def.Yields {
		if y.Chance == nil {
			// chance 省略は確定枠。破壊回収では低確率でしか出ない
			if !full && rng.IntN(100) >= DestroySalvageChance {
				continue
			}
			count, err := rollCount(rng, y.Count)
			if err != nil {
				return nil, err
			}
			stacks = append(stacks, YieldStack{Name: y.Id, Count: count})
			continue
		}
		if !full {
			continue
		}
		// グレード補正は1を基準とし、想定外の0以下が来ても確率を下げない
		chance := min(*y.Chance+skillValue+max(0, (toolGrade-1)*10), 100)
		if rng.IntN(100) < chance {
			count, err := rollCount(rng, y.Count)
			if err != nil {
				return nil, err
			}
			stacks = append(stacks, YieldStack{Name: y.Id, Count: count})
		}
	}

	if full && def.Bonus != nil {
		for _, b := range *def.Bonus {
			// 指定された条件はすべて満たす必要がある。スキーマ上どちらか一方は
			// 必須だが、両方欠けた定義は無効として出さない
			if b.MinSkill == nil && b.MinGrade == nil {
				continue
			}
			if b.MinSkill != nil && skillValue < *b.MinSkill {
				continue
			}
			if b.MinGrade != nil && toolGrade < *b.MinGrade {
				continue
			}
			count, err := rollCount(rng, b.Count)
			if err != nil {
				return nil, err
			}
			stacks = append(stacks, YieldStack{Name: b.Id, Count: count})
		}
	}

	return stacks, nil
}

// ErrInvalidYieldCount は産出個数のダイス表記を解釈できないときに返す
var ErrInvalidYieldCount = errors.New("invalid disassembly yield count notation")

// rollCount は産出個数をダイス表記から抽選する。表記は raw ロード時の検証で担保済みだが、
// 万一パースに失敗したら 0 個を黙って返さず error を返し、呼び出し側で気づけるようにする。
func rollCount(rng *rand.Rand, count rawschema.Dice) (int, error) {
	d, err := consts.ParseDice(count)
	if err != nil {
		return 0, fmt.Errorf("%w: %q: %w", ErrInvalidYieldCount, count, err)
	}
	return d.Roll(rng), nil
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
