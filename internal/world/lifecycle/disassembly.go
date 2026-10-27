package lifecycle

import (
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	w "github.com/kijimaD/ruins/internal/world"
)

// DestroySalvageChance は攻撃破壊時に確定枠の各産出を回収できる百分率の既定値。
// 工具なしでも素材が少しは手に入る立ち上がり用の値で、工具分解より明確に不利にする。
// 分解定義が destroySalvageChance を持てばそちらを使う
const DestroySalvageChance = 30

// destroySalvageChance は def の攻撃破壊時の回収確率を返す
func destroySalvageChance(def *oapi.Disassembly) int {
	if def.DestroySalvageChance != nil {
		return *def.DestroySalvageChance
	}
	return DestroySalvageChance
}

// YieldStack は分解産出のアイテム名と個数の組
type YieldStack struct {
	Name  string
	Count int
}

// RollDisassemblyYields は分解定義から産出を抽選する。
// full が真なら工具による通常分解として扱い、確率枠にスキルと工具グレードの補正を
// 加算し、ボーナス枠の条件判定も行う。偽なら攻撃破壊の回収として扱い、確定枠のみを
// destroySalvageChance で抽選して確率枠とボーナス枠は出さない
func RollDisassemblyYields(rng *rand.Rand, def *oapi.Disassembly, skillValue int, toolGrade int, full bool) ([]YieldStack, error) {
	var stacks []YieldStack

	for _, y := range def.Yields {
		chance := yieldChance(def, y, skillValue, toolGrade, full)
		if chance == 0 {
			continue
		}
		// 工具分解の確定枠は抽選しない。乱数を引くと以降の抽選列がずれる
		guaranteed := full && y.Chance == nil
		if !guaranteed && rng.IntN(100) >= chance {
			continue
		}
		count, err := rollCount(rng, y.Count)
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, YieldStack{Name: y.Id, Count: count})
	}

	for _, b := range activeBonuses(def, skillValue, toolGrade, full) {
		count, err := rollCount(rng, b.Count)
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, YieldStack{Name: b.Id, Count: count})
	}

	return stacks, nil
}

// ExpectedDisassemblyYields は RollDisassemblyYields の産出個数の期待値をアイテム名ごとに返す
func ExpectedDisassemblyYields(def *oapi.Disassembly, skillValue int, toolGrade int, full bool) (map[string]float64, error) {
	expected := map[string]float64{}
	for _, y := range def.Yields {
		mean, err := meanCount(y.Count)
		if err != nil {
			return nil, err
		}
		expected[y.Id] += mean * float64(yieldChance(def, y, skillValue, toolGrade, full)) / 100
	}
	for _, b := range activeBonuses(def, skillValue, toolGrade, full) {
		mean, err := meanCount(b.Count)
		if err != nil {
			return nil, err
		}
		expected[b.Id] += mean
	}
	return expected, nil
}

// yieldChance は産出 y が出る百分率を返す。chance 省略は確定枠で、攻撃破壊では
// destroySalvageChance に下がる。確率枠は工具分解でだけ出て、スキルと工具グレードを加算する
func yieldChance(def *oapi.Disassembly, y oapi.DisassemblyYield, skillValue int, toolGrade int, full bool) int {
	if y.Chance == nil {
		if full {
			return 100
		}
		return destroySalvageChance(def)
	}
	if !full {
		return 0
	}
	// グレード補正は1を基準とし、想定外の0以下が来ても確率を下げない
	return min(*y.Chance+skillValue+max(0, (toolGrade-1)*10), 100)
}

// activeBonuses は条件を満たすボーナス枠を返す。ボーナス枠は工具分解でだけ出る
func activeBonuses(def *oapi.Disassembly, skillValue int, toolGrade int, full bool) []oapi.DisassemblyBonus {
	if !full || def.Bonus == nil {
		return nil
	}
	var bonuses []oapi.DisassemblyBonus
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
		bonuses = append(bonuses, b)
	}
	return bonuses
}

// rollCount は産出個数をダイス表記から抽選する。表記は raw ロード時の検証で担保済みだが、
// 万一パースに失敗したら 0 個を黙って返さず error を返し、呼び出し側で気づけるようにする。
func rollCount(rng *rand.Rand, count oapi.Dice) (int, error) {
	d, err := consts.ParseDice(count)
	if err != nil {
		return 0, fmt.Errorf("invalid disassembly yield count notation: %q: %w", count, err)
	}
	return d.Roll(rng), nil
}

// meanCount は産出個数のダイス表記の期待値を返す
func meanCount(count oapi.Dice) (float64, error) {
	d, err := consts.ParseDice(count)
	if err != nil {
		return 0, fmt.Errorf("invalid disassembly yield count notation: %q: %w", count, err)
	}
	return float64(d.Min()+d.Max()) / 2, nil
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
