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

// EffectiveRot は now 時点の累積劣化量を返す。Perishable の RotAccrued に、RotUpdatedTurn からの
// 経過ぶんを現在の速度で加える。読み取り専用で副作用はない。段階算出と合流判定が通る。
func EffectiveRot(world w.World, entity ecs.Entity, now consts.Turn) consts.Turn {
	p := world.Components.Perishable.Get(entity)
	elapsed := now - p.RotUpdatedTurn
	return p.RotAccrued + consts.Turn(float64(elapsed)*perishRate(world, entity))
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

// FreshnessMarker はメニュー行に添える鮮度の訳語を返す。新鮮や非腐敗は空文字。
// 呼び出し側は空なら何も足さない
func FreshnessMarker(world w.World, entity ecs.Entity) string {
	stage, ok := FreshnessStageOf(world, entity)
	if !ok || stage == gc.FreshnessFresh {
		return ""
	}
	return T(world, stage.Label())
}
