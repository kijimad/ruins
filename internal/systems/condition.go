package systems

import (
	"slices"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// conditionMinorNaturalRecoveryPerTurn は RecoverAfterTend の不調が発症前のとき代謝によらず固定で癒える1ターンの量
const conditionMinorNaturalRecoveryPerTurn = 1

// managedConditionDef は ConditionSystem が扱う不調の定義を返す。定義は components の conditionDefs に集約し、
// Recovery を持つ症状だけを管轄する。低体温は Recovery を持たず TemperatureSystem 管轄なので ok=false
func managedConditionDef(ct gc.ConditionType) (gc.ConditionDef, bool) {
	def, ok := gc.ConditionDefFor(ct)
	if !ok || def.Recovery == "" {
		return gc.ConditionDef{}, false
	}
	return def, true
}

// ConditionSystem は毎ターン怪我と病気の不調を進めるシステム。
// 低体温の進行と回復は TemperatureSystem が担うので、ここでは扱わない。
type ConditionSystem struct{}

// String はシステム名を返す
func (sys *ConditionSystem) String() string {
	return "ConditionSystem"
}

// conditionDamage は重症の不調がターン後に与える HP ダメージ
type conditionDamage struct {
	entity ecs.Entity
	amount int
	cause  gc.DeathCause
}

// Update は怪我と病気の回復軌道を進め、重症の不調で HP を削る
func (sys *ConditionSystem) Update(world w.World) error {
	var toDamage []conditionDamage

	healthQuery := query.ActiveFilter1[gc.HealthStatus](world).Query()
	for healthQuery.Next() {
		entity := healthQuery.Entity()
		hs := world.Components.HealthStatus.Get(entity)
		metab := query.Metabolism(world, entity)
		hasHP := world.Components.HP.Has(entity)

		for p := range hs.Parts {
			part := &hs.Parts[p]
			for i := range part.Conditions {
				cond := &part.Conditions[i]
				def, ok := managedConditionDef(cond.Type)
				if !ok {
					continue
				}

				if delta := conditionTimerDelta(def, cond, metab); delta != 0 {
					cond.UpdateTimer(delta)
				}

			}

			// Timer が 0 になった管理下の不調を除去する
			part.Conditions = slices.DeleteFunc(part.Conditions, func(c gc.HealthCondition) bool {
				_, managed := managedConditionDef(c.Type)
				return managed && c.Timer == 0
			})
		}

		// 血液量が危険域まで落ちたら、じわじわ HP を削る。失血・凍死・衰弱はすべて血液量経由で死なせる
		if hasHP {
			if drain, cause := hs.BloodLossHPDrain(); drain > 0 {
				toDamage = append(toDamage, conditionDamage{entity: entity, amount: drain, cause: cause})
			}
		}
	}

	for _, d := range toDamage {
		gameaction.ApplyConditionDamage(world, d.entity, d.amount, d.cause)
	}

	return nil
}

// conditionTimerDelta は不調の回復モードと治療の質から1ターンの Timer 増減を返す。
// 負なら回復、正なら悪化。int の目盛りで計算し、UpdateTimer の境界で float64 にする。
func conditionTimerDelta(def gc.ConditionDef, cond *gc.HealthCondition, metab consts.Percent) float64 {
	switch def.Recovery {
	case gc.RecoverAfterTend:
		if cond.TendQuality > 0 {
			return -float64(cond.TendQuality.ApplyInt(def.RecoverPer))
		}
		// 発症前の掠り傷相当は代謝によらず固定で僅かに癒える。発症後の未治療は保持する
		if !cond.IsActive() {
			return -float64(conditionMinorNaturalRecoveryPerTurn)
		}
		return 0
	case gc.ProgressUntilTend:
		if cond.TendQuality == 0 {
			return float64(def.WorsenPer)
		}
		// 治療済みは質と代謝の両方で回復が速まる
		rec := cond.TendQuality.ApplyInt(def.RecoverPer)
		rec = metab.ApplyInt(rec)
		return -float64(rec)
	case gc.RecoverOverTime:
		// 自己限定。未治療でも代謝で自然に治る。治療すればその質でさらに速まる
		rec := def.RecoverPer
		if cond.TendQuality > 0 {
			rec = cond.TendQuality.ApplyInt(rec)
		}
		return -float64(metab.ApplyInt(rec))
	}
	// default を置くと exhaustive linter が沈黙するので置かない。内部の信頼できる値なので未知は panic する
	panic("unknown RecoveryMode: " + string(def.Recovery))
}
