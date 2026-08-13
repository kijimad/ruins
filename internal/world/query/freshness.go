package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// FreshnessStageOf は entity の鮮度段階を返す。Perishable を持たなければ ok=false。
// 鮮度の算出をここへ集約し、食べる処理と表示が同じ判定を通す
func FreshnessStageOf(world w.World, entity ecs.Entity) (gc.FreshnessStage, bool) {
	if !world.Components.Perishable.Has(entity) {
		return "", false
	}
	return world.Components.Perishable.Get(entity).Stage(GetGameTime(world).TotalTurns), true
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
