package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

const (
	// healthRegenIntervalTurns ターンに一度だけ自然回復する。毎ターン整数を足すと実効の下限が 1/turn になり
	// 戦闘外の全快が速すぎるため、間引いて実効を代謝100%で 0.2 HP/ターンにする。
	healthRegenIntervalTurns = 5
	// healthRegenPerInterval は回復ターンに足す基準 HP。代謝が基準 100 のときの値で、実際は Metabolism
	// 倍率を掛ける。代謝が 100 未満だと切り捨てで 0 になり回復しない。値は実プレイで調整する。
	healthRegenPerInterval = 1
)

// HealthRegenSystem は毎ターン HP を代謝ぶん自然回復させるシステム
type HealthRegenSystem struct{}

// String はシステム名を返す
func (sys *HealthRegenSystem) String() string {
	return "HealthRegenSystem"
}

// Update は healthRegenIntervalTurns ターンに一度、HP を持つ生存エンティティの HP を代謝ぶん回復させる。
// 回復ターン以外は何もしない。自然回復は静かに進めるので回復数値を出す ApplyHealing は使わず HP を直接足す。
// 数値を出す即時回復はアイテム使用に限る。
func (sys *HealthRegenSystem) Update(world w.World) error {
	// 回復ターン以外は間引く。TurnState は turn system の中で必ず存在し、TurnNumber は1始まりなので
	// 0 での早期回復は起きず、最初の回復は healthRegenIntervalTurns ターン目になる
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
