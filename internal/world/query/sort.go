package query

import (
	"cmp"
	"slices"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// SortEntities はエンティティリストをソートする汎用関数
// Nameコンポーネントを持つエンティティを名前順でソートする
// Nameコンポーネントを持っていないエンティティはスキップされる
func SortEntities(world w.World, entities []ecs.Entity) []ecs.Entity {
	if len(entities) == 0 {
		return entities
	}

	named := make([]ecs.Entity, 0, len(entities))
	for _, entity := range entities {
		if world.Components.Name.Has(entity) {
			named = append(named, entity)
		}
	}

	// 表示名で並べ、同名は品種・鮮度・装備の個体差で決定的に割る。ECS の走査順に依存させない。
	// 走査順に頼ると、ドロップ等の構造変更で Ark の swap-remove が格納順を変え、
	// 同名の別スタックの並びが入れ替わる。同定に使える値だけで全順序を定めて安定させる。
	slices.SortStableFunc(named, func(a, b ecs.Entity) int {
		if c := cmp.Compare(world.Components.Name.Get(a).Name, world.Components.Name.Get(b).Name); c != 0 {
			return c
		}
		if c := cmp.Compare(sortRawID(world, a), sortRawID(world, b)); c != 0 {
			return c
		}
		if c := cmp.Compare(freshnessRank(world, a), freshnessRank(world, b)); c != 0 {
			return c
		}
		return cmp.Compare(equipFingerprint(world, a), equipFingerprint(world, b))
	})

	return named
}

// sortRawID は同定キー RawID を返す。持たなければ空文字。同名の別品種を決定的に分ける副キー。
func sortRawID(world w.World, entity ecs.Entity) string {
	if world.Components.RawID.Has(entity) {
		return world.Components.RawID.Get(entity).ID
	}
	return ""
}

// freshnessRank は鮮度段階を並び順の数値にする。非腐敗品は先頭の0、腐敗品は段階の Rank に従う。
// 同一品種でも鮮度が違えば別スタックになるため、その並びを決定的にする副キー。
func freshnessRank(world w.World, entity ecs.Entity) int {
	stage, ok := FreshnessStageOf(world, entity)
	if !ok {
		return 0
	}
	return stage.Rank()
}
