package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// 通り方と見通しの印は地形と置物だけが持つ。マスにはキャラクター・アイテム・跡も乗り、それらは印を持たなくても
// 塞いだり遮ったりしない。壊れて掃除を待つ置物も塞がず遮らない。印を持たないことで判定する側は、この2つの関数で
// 対象を絞る。

// terrainAndPropKinds は地形と置物を引く印の並び。Ark のフィルタは「または」を書けないので2本に分けて引く
var terrainAndPropKinds = []ecs.Comp{ecs.C[gc.Tile](), ecs.C[gc.Prop]()}

// ForEachTerrainOrPropWithout は markers のどれも持たない地形と置物ごとに fn を呼ぶ。
// 退避中のステージの実体と、壊れて掃除を待つ置物は除く。
func ForEachTerrainOrPropWithout(world w.World, fn func(ecs.Entity), markers ...ecs.Comp) {
	exclude := append([]ecs.Comp{ecs.C[gc.Dead]()}, markers...)
	for _, kind := range terrainAndPropKinds {
		q := ActiveFilter1[gc.GridElement](world).With(kind).Without(exclude...).Query()
		for q.Next() {
			fn(q.Entity())
		}
	}
}

// IsStandingTerrainOrProp は e が地形か置物で、壊れていないかを返す
func IsStandingTerrainOrProp(world w.World, e ecs.Entity) bool {
	if world.Components.Dead.Has(e) {
		return false
	}
	return world.Components.Tile.Has(e) || world.Components.Prop.Has(e)
}
