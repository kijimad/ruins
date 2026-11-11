package balance

import (
	"fmt"
	"strings"
)

// Markdown はシミュレーションのレポートを Markdown の表にする。
func (r *Report) Markdown() string {
	var b strings.Builder
	b.WriteString("# バランスシミュレーション\n\n")
	fmt.Fprintf(&b, "プレイヤー %s。HP %d、筋力 %d、感覚 %d、器用 %d、敏捷 %d、防御 %d。\n\n",
		r.Player.Name, r.Player.Hp, r.Player.Strength, r.Player.Sensation, r.Player.Dexterity, r.Player.Agility, r.Player.Defense)
	fmt.Fprintf(&b, "武器 %s。ダメージ %d、命中 %d。\n\n", r.Weapon.Name, r.Weapon.Damage, r.Weapon.Accuracy)

	b.WriteString("## 敵テーブル\n\n")
	for i := range r.EnemyTables {
		run := &r.EnemyTables[i]
		fmt.Fprintf(&b, "### %s\n\n", run.Name)
		fmt.Fprintf(&b, "試行 %d、最大深度 %d、到達深度の中央値 %d、死亡率 %s。\n\n", run.Trials, run.MaxDepth, run.MedianDepth, percent(run.DeathRate))
		b.WriteString("| 深度 | HP 中央値 | HP p5 | HP p95 | 回復前 HP 中央値 | 撃破ターン中央値 | 武器ダメージ中央値 | 空腹中央値 | 被ダメージ中央値 | 回復中央値 | 急死率 |\n")
		b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|\n")
		for j := range run.Depths {
			d := &run.Depths[j]
			fmt.Fprintf(&b, "| %d | %d | %d | %d | %d | %d | %d | %d | %d | %d | %s |\n",
				d.Depth, d.MedianHP, d.P5HP, d.P95HP, d.MedianHPBeforeHeal, d.MedianKillTurns, d.MedianWeaponDamage,
				d.MedianHunger, d.MedianDamage, d.MedianHealing, percent(d.SuddenDeathRate))
		}
		b.WriteString("\n")
	}

	b.WriteString("## 武器×敵の戦闘\n\n")
	b.WriteString("| 武器 | 敵 | DPS | 遠距離 |\n|---|---|---|---|\n")
	for _, m := range r.BattleMetrics {
		fmt.Fprintf(&b, "| %s | %s | %.2f | %t |\n", m.Weapon, m.Enemy, m.Dps, m.IsRanged)
	}
	b.WriteString("\n")

	b.WriteString("## 施設の戦利品\n\n")
	for _, f := range r.RoomLoot {
		fmt.Fprintf(&b, "### %s\n\n試行 %d。\n\n", f.Facility, f.Trials)
		b.WriteString("| 部屋 | アイテム | 1棟で出る確率 | 1棟あたりの期待個数 | 売買価値 |\n|---|---|---|---|---|\n")
		for _, room := range f.Rooms {
			for _, it := range room.Items {
				fmt.Fprintf(&b, "| %s | %s | %s | %.2f | %d |\n", room.Role, it.Name, percent(it.Prob), it.ExpectedCount, it.Value)
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func percent(rate float64) string {
	return fmt.Sprintf("%.1f%%", rate*100)
}
