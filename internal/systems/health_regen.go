package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

const (
	// healthRegenIntervalTurns ターンに一度だけ回復する。毎ターンだと全快が速すぎるため間引く。
	healthRegenIntervalTurns = 5
	// healthRegenPerInterval は回復ターンに足す基準 HP。Metabolism を掛け、代謝100%未満は0で回復しない。
	healthRegenPerInterval = 1
)

// HealthRegenSystem は HP を代謝ぶん自然回復させるシステム
type HealthRegenSystem struct{}

// String はシステム名を返す
func (sys *HealthRegenSystem) String() string {
	return "HealthRegenSystem"
}

// Update は healthRegenIntervalTurns ターンに一度、生存エンティティの HP を代謝ぶん回復させる。数値は出さず直接足す。
func (sys *HealthRegenSystem) Update(world w.World) error {
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
		// HP を削る不調のあいだは回復しない。回復が相殺して減少を隠さないため
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
