package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// 代謝の定数。値は実プレイで調整する
const (
	// metabolismVitBonus は VIT 1 あたり回復へ足す%
	metabolismVitBonus = 3
	// metabolismSleepingBonus は睡眠中に回復へ足す基準%。寝具 Quality を掛ける
	metabolismSleepingBonus = 50
)

// RecoverySources は回復レートへの寄与を内訳として返す。命中・速度と同じ 熟練 × 身体機能 の形で、
// 熟練側は VIT・睡眠の加算、身体機能側は代謝の値の乗算差分として載せる。空腹・疲労は代謝を、
// 意識は行動を下げるが回復には効かせない。痛くても傷は治るので痛みも回復には効かない。
// Metabolism と Effects タブが同じこの導出を読むので値と内訳がずれない
func RecoverySources(world w.World, entity ecs.Entity) []gc.ProficiencySource {
	var srcs []gc.ProficiencySource

	// 熟練側。基準 100 に VIT・睡眠を積む
	prof := int(consts.PercentBase)
	if world.Components.Abilities.Has(entity) {
		vit := world.Components.Abilities.Get(entity).Vitality.Total
		if v := vit * metabolismVitBonus; v != 0 {
			srcs = append(srcs, gc.ProficiencySource{Kind: gc.SourceAbility, Ability: gc.AblVIT, Amount: vit, Value: v})
			prof += v
		}
	}
	if world.Components.Sleeping.Has(entity) {
		quality := world.Components.Sleeping.Get(entity).Quality
		if v := quality.ApplyInt(metabolismSleepingBonus); v != 0 {
			srcs = append(srcs, gc.ProficiencySource{Kind: gc.SourceSleeping, Value: v})
			prof += v
		}
	}

	// 身体機能側。代謝の値を熟練へ乗算し、内訳には加法差分で載せる。命中の畳み込みと同じ形
	metab := EffectiveBodyFuncs(world, entity).Metabolism
	if withCap := metab.ApplyInt(prof); withCap != prof {
		srcs = append(srcs, gc.ProficiencySource{Kind: gc.SourceBodyFunc, BodyFunc: gc.BodyFuncMetabolism, Amount: int(metab), Value: withCap - prof})
	}
	return srcs
}

// Metabolism は HP の自然回復と病気の回復にかかる速度係数を返す。基準は 100、下限は 0。
// 熟練 VIT・睡眠 と 身体機能 代謝の値の合成で、命中・速度と同じ 熟練 × 身体機能 の形。
// Effects タブの内訳と同じ導出を読むので値と内訳がずれない
func Metabolism(world w.World, entity ecs.Entity) consts.Percent {
	total := int(consts.PercentBase)
	for _, s := range RecoverySources(world, entity) {
		total += s.Value
	}
	return consts.Percent(max(total, 0))
}
