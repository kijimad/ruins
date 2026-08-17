package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// StackKey はアイテムのスタック同一性キー。キーが等しいアイテムだけが同一スタックに束ねられる。
// スタック数はこのキーで束ねた同値クラスの大きさとして導出し、保存しない。
// 個体差を足すときはキーに項を1つ加える。鮮度のような連続量は段階へ量子化すれば束ね、
// 生値のまま持てば個体ごとに分かれる。
type StackKey struct {
	RawID          string            // 生成元の同定キー。まず品種が一致すること
	FreshnessStage gc.FreshnessStage // 腐敗品の鮮度段階。非腐敗品は空文字
}

// StackKeyOf は entity のスタック同一性キーを返す。スタック同一判定の唯一の権威。
// RawID に加え、腐敗品なら現在の鮮度段階を含める。非腐敗品の段階は空文字になる。
func StackKeyOf(world w.World, entity ecs.Entity) StackKey {
	key := StackKey{RawID: world.Components.RawID.Get(entity).ID}
	if stage, ok := FreshnessStageOf(world, entity); ok {
		key.FreshnessStage = stage
	}
	return key
}

// SameStack は a と b が同一スタックに束ねられるかを返す。StackKey の等価そのもの。
func SameStack(world w.World, a ecs.Entity, b ecs.Entity) bool {
	return StackKeyOf(world, a) == StackKeyOf(world, b)
}
