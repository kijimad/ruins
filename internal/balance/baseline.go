package balance

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// TargetPowerRatio は序盤戦闘の目標帯の中心値を返す。day1=2.5 から day20=1.3 へ線形に下げ、
// 序盤は余裕あり、終盤で拮抗へ寄せる。値は人間が決める設計仮説で、プレイで見直す。
func TargetPowerRatio(day int) float64 {
	const d0, m0, d1, m1 = 1, 2.5, 20, 1.3
	switch {
	case day <= d0:
		return m0
	case day >= d1:
		return m1
	default:
		return m0 + (m1-m0)*float64(day-d0)/float64(d1-d0)
	}
}

// targetBand は目標帯の許容幅。中心±この値を目標帯内とみなす。
const targetBand = 0.3

// BaselineRow はベースライン表の1行。ある日の実測 powerRatio と目標帯の関係を持つ。
type BaselineRow struct {
	Day        int
	Danger     int
	PowerRatio float64
	Target     float64
	InRange    bool
}

// Baseline は指定した敵テーブルの序盤戦闘ベースラインを返す。プレイヤーは強化なしの素手を
// 最悪ケースとして固定し、難易度カーブを目標帯と突き合わせる。
func Baseline(master oapi.Raws, playerName, weaponName, enemyTableName string, days int) ([]BaselineRow, error) {
	player, err := LoadCombatantFromMember(master, playerName)
	if err != nil {
		return nil, err
	}
	weapon, err := LoadWeaponFromItem(master, weaponName)
	if err != nil {
		return nil, err
	}
	curve, err := DifficultyCurve(master, player, weapon, enemyTableName, days)
	if err != nil {
		return nil, err
	}
	rows := make([]BaselineRow, len(curve))
	for i, m := range curve {
		tg := TargetPowerRatio(m.Day)
		rows[i] = BaselineRow{
			Day:        m.Day,
			Danger:     m.Danger,
			PowerRatio: m.PowerRatio,
			Target:     tg,
			InRange:    m.PowerRatio >= tg-targetBand && m.PowerRatio <= tg+targetBand,
		}
	}
	return rows, nil
}

// RenderBaselineMarkdown は全敵テーブルのベースラインを markdown 表にする。パラメータ変更の
// 影響が diff で読めるよう、テーブルごとに日次の powerRatio と目標・目標帯内かを並べる。
// 文字列リテラルは源泉英語化の方針に合わせ英語にする。テーブル名は raw 由来の実行時データ。
func RenderBaselineMarkdown(master oapi.Raws, playerName, weaponName string, days int) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# Balance baseline\n\n")
	fmt.Fprintf(&b, "Early-combat difficulty curve. Player is fixed to the worst case of un-upgraded `%s` + `%s`.\n", playerName, weaponName)
	fmt.Fprintf(&b, "power ratio is the ratio of how fast the player kills to how fast the player dies; 1.0 is even, higher is easier.\n")
	fmt.Fprintf(&b, "Target power ratio is day1=2.5 to day20=1.3, band +/-%.1f. The target is a design hypothesis to revisit by playing.\n\n", targetBand)

	tables := raw.PtrSlice(master.EnemyTables)
	sort.Slice(tables, func(i, j int) bool { return tables[i].Id < tables[j].Id })
	for _, table := range tables {
		rows, err := Baseline(master, playerName, weaponName, table.Id, days)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "## %s (%s)\n\n", table.Name, table.Id)
		fmt.Fprintf(&b, "| day | danger | power ratio | target | in band |\n")
		fmt.Fprintf(&b, "|---:|---:|---:|---:|:--:|\n")
		for _, r := range rows {
			mark := "out"
			if r.InRange {
				mark = "in"
			}
			fmt.Fprintf(&b, "| %d | %d | %.2f | %.2f | %s |\n", r.Day, r.Danger, r.PowerRatio, r.Target, mark)
		}
		fmt.Fprintf(&b, "\n")
	}

	// 生存圧。食料は日単位、寒さはターン単位で時間スケールが異なる
	fmt.Fprintf(&b, "## survival pressure\n\n")
	fmt.Fprintf(&b, "Time to starve without food, and time to hypothermia by effective temperature (ambient + insulation).\n\n")
	fmt.Fprintf(&b, "| metric | value |\n|---|---:|\n")
	fmt.Fprintf(&b, "| days until starving (hunger < 33%%) | %.2f |\n", DaysUntilStarving())
	fmt.Fprintf(&b, "| days until hunger empty | %.2f |\n", DaysUntilHungerEmpty())
	fmt.Fprintf(&b, "\n| effective temp (C) | turns to hypothermia |\n|---:|---:|\n")
	for _, temp := range []int{-20, -10, 0, 5, 10, 15} {
		fmt.Fprintf(&b, "| %d | %.0f |\n", temp, TurnsToHypothermia(temp))
	}
	fmt.Fprintf(&b, "\n")
	return b.String(), nil
}
