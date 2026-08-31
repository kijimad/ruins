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

// RecoveryMode は不調が未治療のときどう振る舞い、治療でどう治るかを表す。
// 低体温は TemperatureSystem が体温から進めるのでこの区分には入らない。
type RecoveryMode int

const (
	// RecoverAfterTend は治療して初めて回復する。未治療は保持し悪化しない。骨折・切り傷。
	// ただし発症前の掠り傷相当は代謝で僅かに癒える。
	RecoverAfterTend RecoveryMode = iota
	// ProgressUntilTend は未治療なら悪化し続け、治療して初めて回復軌道へ乗る。病気。
	ProgressUntilTend
)

// conditionMinorNaturalRecoveryPerTurn は RecoverAfterTend の不調が発症前のとき代謝で癒える1ターンの量
const conditionMinorNaturalRecoveryPerTurn = 1

// conditionSpec は不調の種類ごとの固定パラメータ。ConditionSystem が扱う怪我と病気を定める。
// 低体温は TemperatureSystem が担うのでこのカタログには載せない。
type conditionSpec struct {
	Recovery   RecoveryMode // 未治療の振る舞いと治し方
	WorsenPer  int          // ProgressUntilTend で未治療のとき1ターン Timer を増やす量
	RecoverPer int          // 治療済みで1ターン Timer を減らす基準量。質と代謝で増減する
	HPDamage   int          // 重症で毎ターン与える HP ダメージ。0 なら無害
	Cause      string       // HPDamage で倒したときの死因。HPDamage が0なら未使用
}

// conditionCatalog は ConditionSystem が扱う不調の種類を網羅する実行時定数。
// 登録漏れがあると引けずに動作不全になるので、扱う種類はここに必ず載せる。
var conditionCatalog = map[gc.ConditionType]conditionSpec{
	gc.ConditionFracture: {
		Recovery:   RecoverAfterTend,
		RecoverPer: 3,
	},
	gc.ConditionLaceration: {
		Recovery:   RecoverAfterTend,
		RecoverPer: 4,
	},
	gc.ConditionLiverIllness: {
		Recovery:   ProgressUntilTend,
		WorsenPer:  2,
		RecoverPer: 3,
		HPDamage:   2,
		Cause:      gc.CauseIllness,
	},
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
	cause  string
}

// Update は怪我と病気の回復軌道を進め、重症の不調で HP を削る
func (sys *ConditionSystem) Update(world w.World) error {
	var toMark []ecs.Entity
	var toDamage []conditionDamage

	healthQuery := query.ActiveFilter1[gc.HealthStatus](world).Query()
	for healthQuery.Next() {
		entity := healthQuery.Entity()
		hs := world.Components.HealthStatus.Get(entity)
		metab := query.Metabolism(world, entity)
		hasHP := world.Components.HP.Has(entity)
		changed := false

		for p := range hs.Parts {
			part := &hs.Parts[p]
			for i := range part.Conditions {
				cond := &part.Conditions[i]
				spec, ok := conditionCatalog[cond.Type]
				if !ok {
					continue
				}

				if delta := conditionTimerDelta(spec, cond, metab); delta != 0 {
					if prev, cur := cond.UpdateTimer(delta); prev != cur {
						changed = true
					}
				}

				if cond.Severity == gc.SeveritySevere && spec.HPDamage > 0 && hasHP {
					toDamage = append(toDamage, conditionDamage{entity: entity, amount: spec.HPDamage, cause: spec.Cause})
				}
			}

			// Timer が 0 になった管理下の不調を除去する
			part.Conditions = slices.DeleteFunc(part.Conditions, func(c gc.HealthCondition) bool {
				_, managed := conditionCatalog[c.Type]
				return managed && c.Timer == 0
			})
		}

		if changed && world.Components.Player.Has(entity) {
			toMark = append(toMark, entity)
		}
	}

	for _, entity := range toMark {
		if !world.Components.StatsChanged.Has(entity) {
			world.Components.StatsChanged.Add(entity, &gc.StatsChanged{})
		}
	}

	for _, d := range toDamage {
		gameaction.ApplyConditionDamage(world, d.entity, d.amount, d.cause)
	}

	return nil
}

// conditionTimerDelta は不調の回復モードと治療の質から1ターンの Timer 増減を返す。
// 負なら回復、正なら悪化。int の目盛りで計算し、UpdateTimer の境界で float64 にする。
func conditionTimerDelta(spec conditionSpec, cond *gc.HealthCondition, metab consts.Percent) float64 {
	switch spec.Recovery {
	case RecoverAfterTend:
		if cond.TendQuality > 0 {
			return -float64(cond.TendQuality.ApplyInt(spec.RecoverPer))
		}
		// 発症前の掠り傷相当は代謝で僅かに癒える。発症後の未治療は保持する
		if !cond.IsActive() {
			return -float64(conditionMinorNaturalRecoveryPerTurn)
		}
		return 0
	case ProgressUntilTend:
		if cond.TendQuality == 0 {
			return float64(spec.WorsenPer)
		}
		// 治療済みは質と代謝の両方で回復が速まる
		rec := cond.TendQuality.ApplyInt(spec.RecoverPer)
		rec = metab.ApplyInt(rec)
		return -float64(rec)
	default:
		return 0
	}
}
