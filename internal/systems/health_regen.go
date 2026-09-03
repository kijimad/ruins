package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// healthRegenBasePerTurn は代謝が基準 100 のとき毎ターン回復する HP。
// 実際の回復量はこれに Metabolism 倍率を掛ける。値は実プレイで調整する
const healthRegenBasePerTurn = 2

// HealthRegenSystem は毎ターン HP を代謝ぶん自然回復させるシステム
type HealthRegenSystem struct{}

// String はシステム名を返す
func (sys *HealthRegenSystem) String() string {
	return "HealthRegenSystem"
}

// Update は HP を持つ生存エンティティの HP を代謝ぶん回復させる。
// 自然回復は静かに進めるので回復数値を出す ApplyHealing は使わず HP を直接足す。
// 数値を出す即時回復はアイテム使用に限る。
func (sys *HealthRegenSystem) Update(world w.World) error {
	var targets []ecs.Entity
	hpQuery := query.ActiveFilter1[gc.HP](world).Query()
	for hpQuery.Next() {
		targets = append(targets, hpQuery.Entity())
	}

	for _, entity := range targets {
		if world.Components.Dead.Has(entity) {
			continue
		}
		// HP を削る不調があるあいだは自然回復しない。回復が相殺して、じわじわ減っているのを隠さないようにする
		if world.Components.HealthStatus.Has(entity) && world.Components.HealthStatus.Get(entity).IsHPDraining() {
			continue
		}
		hp := world.Components.HP.Get(entity)
		if hp.Current >= hp.Max {
			continue
		}
		regen := query.Metabolism(world, entity).ApplyInt(healthRegenBasePerTurn)
		if regen <= 0 {
			continue
		}
		hp.Current += regen
		if hp.Current > hp.Max {
			hp.Current = hp.Max
		}
	}

	return nil
}
