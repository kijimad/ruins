package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// 代謝の定数。値は実プレイで調整する
const (
	// metabolismVitBonus は VIT 1 あたり代謝倍率へ足す%
	metabolismVitBonus = 3
	// metabolismSatiatedBonus は満腹のとき代謝倍率へ足す%
	metabolismSatiatedBonus = 20
	// metabolismHungryPenalty は空腹のとき代謝倍率から引く%
	metabolismHungryPenalty = 30
	// metabolismStarvingPenalty は飢餓のとき代謝倍率から引く%
	metabolismStarvingPenalty = 60
	// metabolismSleepingBonus は睡眠中に代謝倍率へ足す基準%。寝具 Quality を掛ける
	metabolismSleepingBonus = 50
)

// Metabolism は HP の自然回復と病気の回復にかかる速度係数を返す。基準は 100。
// VIT が高いほど速く、空腹や飢餓で遅くなる。よく食べ休めば速く、飢えれば遅い。
// 下限は 0 で、負にはならない。
func Metabolism(world w.World, entity ecs.Entity) consts.Percent {
	pct := consts.PercentBase

	if world.Components.Abilities.Has(entity) {
		vit := world.Components.Abilities.Get(entity).Vitality.Total
		pct += consts.Percent(vit * metabolismVitBonus)
	}

	if world.Components.Hunger.Has(entity) {
		switch world.Components.Hunger.Get(entity).GetLevel() {
		case gc.HungerSatiated:
			pct += metabolismSatiatedBonus
		case gc.HungerNormal:
			// 標準。増減なし
		case gc.HungerHungry:
			pct -= metabolismHungryPenalty
		case gc.HungerStarving:
			pct -= metabolismStarvingPenalty
		}
	}

	// 疲労のペナルティ。係数は Fatigue.Penalty の1表から読む
	if world.Components.Fatigue.Has(entity) {
		pct += world.Components.Fatigue.Get(entity).Penalty().RecoveryAdd
	}

	// 睡眠中は回復が上がる。基準ボーナスに寝具 Quality を掛ける
	if world.Components.Sleeping.Has(entity) {
		quality := world.Components.Sleeping.Get(entity).Quality
		pct += consts.Percent(quality.ApplyInt(metabolismSleepingBonus))
	}

	if pct < 0 {
		pct = 0
	}

	return pct
}
