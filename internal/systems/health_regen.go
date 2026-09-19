package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

const (
	// healthRegenIntervalTurns ターンに一度だけ自然回復する。毎ターン回復だと全快が速すぎるため間引き、
	// 実効を代謝100%で 0.2 HP/ターンにする。
	healthRegenIntervalTurns = 5
	// healthRegenPerInterval は回復ターンに足す基準 HP。Metabolism 倍率を掛け、代謝100%未満は切り捨て0で回復しない。
	healthRegenPerInterval = 1
)

// HealthRegenSystem は HP を代謝ぶん自然回復させるシステム
type HealthRegenSystem struct{}

// String はシステム名を返す
func (sys *HealthRegenSystem) String() string {
	return "HealthRegenSystem"
}

// Update は healthRegenIntervalTurns ターンに一度、生存エンティティの HP を代謝ぶん回復させる。
// 回復数値を出す ApplyHealing は使わず静かに直接足す。
func (sys *HealthRegenSystem) Update(world w.World) error {
	// TurnState は turn system 内で必ず存在する。TurnNumber は1始まりで最初の回復は healthRegenIntervalTurns ターン目
	if int(query.GetTurnState(world).TurnNumber)%healthRegenIntervalTurns != 0 {
		return nil
	}

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
		regen := query.Metabolism(world, entity).ApplyInt(healthRegenPerInterval)
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
