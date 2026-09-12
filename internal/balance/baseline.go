package balance

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// CorridorMargin は序盤戦闘の目標回廊の中心値を返す。day1=2.5 から day20=1.3 へ線形に下げ、
// 序盤は余裕あり、終盤で拮抗へ寄せる。値は人間が決める設計仮説で、プレイで見直す。
func CorridorMargin(day int) float64 {
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

// corridorBand は回廊の許容幅。中心±この値を回廊内とみなす。
const corridorBand = 0.3

// BaselineRow はベースライン表の1行。ある日の実測 margin と目標回廊の関係を持つ。
type BaselineRow struct {
	Day        int
	Danger     int
	Margin     float64
	Target     float64
	InCorridor bool
}

// Baseline は指定した敵テーブルの序盤戦闘ベースラインを返す。プレイヤーは強化なしの素手を
// 最悪ケースとして固定し、難易度カーブを目標回廊と突き合わせる。
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
		tg := CorridorMargin(m.Day)
		rows[i] = BaselineRow{
			Day:        m.Day,
			Danger:     m.Danger,
			Margin:     m.Margin,
			Target:     tg,
			InCorridor: m.Margin >= tg-corridorBand && m.Margin <= tg+corridorBand,
		}
	}
	return rows, nil
}

// RenderBaselineMarkdown は全敵テーブルのベースラインを markdown 表にする。パラメータ変更の
// 影響が diff で読めるよう、テーブルごとに日次の margin と目標・回廊内かを並べる。
// 文字列リテラルは源泉英語化の方針に合わせ英語にする。テーブル名は raw 由来の実行時データ。
func RenderBaselineMarkdown(master oapi.Raws, playerName, weaponName string, days int) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# Balance baseline\n\n")
	fmt.Fprintf(&b, "Early-combat difficulty curve. Player is fixed to the worst case of un-upgraded `%s` + `%s`.\n", playerName, weaponName)
	fmt.Fprintf(&b, "margin is the ratio of how fast the player kills to how fast the player dies; 1.0 is even, higher is easier.\n")
	fmt.Fprintf(&b, "Target corridor is day1=2.5 to day20=1.3, band +/-%.1f. The corridor is a design hypothesis to revisit by playing.\n\n", corridorBand)

	tables := raw.PtrSlice(master.EnemyTables)
	sort.Slice(tables, func(i, j int) bool { return tables[i].Id < tables[j].Id })
	for _, table := range tables {
		rows, err := Baseline(master, playerName, weaponName, table.Id, days)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "## %s (%s)\n\n", table.Name, table.Id)
		fmt.Fprintf(&b, "| day | danger | margin | target | corridor |\n")
		fmt.Fprintf(&b, "|---:|---:|---:|---:|:--:|\n")
		for _, r := range rows {
			mark := "out"
			if r.InCorridor {
				mark = "in"
			}
			fmt.Fprintf(&b, "| %d | %d | %.2f | %.2f | %s |\n", r.Day, r.Danger, r.Margin, r.Target, mark)
		}
		fmt.Fprintf(&b, "\n")
	}
	return b.String(), nil
}
