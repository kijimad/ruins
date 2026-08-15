package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// perishRate は entity の現在の劣化速度を返す。1.0 が基準。
// 将来、置き場所の温度でこの値を変える。冷所は遅く、暖所は速い。今は一律 1.0。
func perishRate(_ w.World, _ ecs.Entity) float64 {
	return 1.0
}

// EffectiveRot は now 時点の累積劣化量を返す。Perishable の RotAccrued に、RotAsOfTurn からの
// 経過ぶんを現在の速度で加える。読み取り専用で副作用はない。段階算出と合流判定が通る。
func EffectiveRot(world w.World, entity ecs.Entity, now consts.Turn) consts.Turn {
	p := world.Components.Perishable.Get(entity)
	elapsed := now - p.RotAsOfTurn
	return p.RotAccrued + consts.Turn(float64(elapsed)*perishRate(world, entity))
}

// AdvanceRot は entity の RotAccrued を now まで前進させ、RotAsOfTurn を揃える。
// 速度が変わる直前や合流の前に呼び、以後の実効値計算を正しく保つ。
func AdvanceRot(world w.World, entity ecs.Entity, now consts.Turn) {
	p := world.Components.Perishable.Get(entity)
	p.RotAccrued = EffectiveRot(world, entity, now)
	p.RotAsOfTurn = now
}

// FreshnessStageOf は entity の鮮度段階を返す。Perishable を持たなければ ok=false。
// 鮮度の算出をここへ集約し、食べる処理と表示が同じ判定を通す
func FreshnessStageOf(world w.World, entity ecs.Entity) (gc.FreshnessStage, bool) {
	if !world.Components.Perishable.Has(entity) {
		return "", false
	}
	now := GetGameTime(world).TotalTurns
	return world.Components.Perishable.Get(entity).Stage(EffectiveRot(world, entity, now)), true
}

// StacksWith は a と b が同じ束に合流できるかを返す。スタック合流の同一判定はここに集約する。
// 非腐敗は RawID 一致で合流する。腐敗食は RawID 一致かつ現在の鮮度段階が同じときだけ合流し、
// 合流時に劣化量を加重平均する。段階違いの合流を禁じ、新鮮で腐敗を薄める抜け穴を防ぐ。
func StacksWith(world w.World, a ecs.Entity, b ecs.Entity) bool {
	if world.Components.RawID.Get(a).ID != world.Components.RawID.Get(b).ID {
		return false
	}
	aPerish := world.Components.Perishable.Has(a)
	if aPerish != world.Components.Perishable.Has(b) {
		return false
	}
	if !aPerish {
		return true
	}
	sa, _ := FreshnessStageOf(world, a)
	sb, _ := FreshnessStageOf(world, b)
	return sa == sb
}

// FreshnessMarker はメニュー行に添える鮮度の訳語を返す。新鮮や非腐敗は空文字。
// 呼び出し側は空なら何も足さない
func FreshnessMarker(world w.World, entity ecs.Entity) string {
	stage, ok := FreshnessStageOf(world, entity)
	if !ok || stage == gc.FreshnessFresh {
		return ""
	}
	return T(world, FreshnessLabel(stage))
}

// FreshnessLabel は鮮度段階の表示に使う英語 msgid を返す
func FreshnessLabel(stage gc.FreshnessStage) string {
	switch stage {
	case gc.FreshnessFresh:
		return "Fresh"
	case gc.FreshnessStale:
		return "Stale"
	case gc.FreshnessRotten:
		return "Rotten"
	}
	panic("unknown FreshnessStage: " + string(stage))
}
