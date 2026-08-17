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

// StackCountOf は candidates のうち entity と同一スタックのものを数える。スタック数は保存せず
// この数え上げで導出する。数える範囲は呼び出し側が候補集合として渡す。リロードやクラフトは
// バックパックの中を、総重量は装備や収納も含む所有全体を候補にする、というように範囲を選ぶ。
func StackCountOf(world w.World, entity ecs.Entity, candidates []ecs.Entity) int {
	key := StackKeyOf(world, entity)
	n := 0
	for _, c := range candidates {
		if StackKeyOf(world, c) == key {
			n++
		}
	}
	return n
}

// Stack は同一スタックに束ねたアイテム群。表示や一括操作の単位に使う。
type Stack struct {
	Rep     ecs.Entity   // 束の代表。入力での初出エンティティ
	Count   int          // 束に属する個数
	Members []ecs.Entity // 束に属する全エンティティ。初出順
}

// GroupStacks は entities を StackKey で束ね、各束の代表と個数を返す。
// 束の並びは入力での初出順にして決定的にする。表示の一覧はこの束1つを1行にする。
func GroupStacks(world w.World, entities []ecs.Entity) []Stack {
	index := make(map[StackKey]int, len(entities))
	stacks := make([]Stack, 0, len(entities))
	for _, e := range entities {
		key := StackKeyOf(world, e)
		if i, ok := index[key]; ok {
			stacks[i].Count++
			stacks[i].Members = append(stacks[i].Members, e)
			continue
		}
		index[key] = len(stacks)
		stacks = append(stacks, Stack{Rep: e, Count: 1, Members: []ecs.Entity{e}})
	}
	return stacks
}
