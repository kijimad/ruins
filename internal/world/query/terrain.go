package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// terrainAndPropKinds は地形と置物を引く印の並び。Ark のフィルタは「または」を書けないので2本に分けて引く
var terrainAndPropKinds = []ecs.Comp{ecs.C[gc.Tile](), ecs.C[gc.Prop]()}

// ForEachTerrainOrPropWithout は markers のどれも持たない地形と置物ごとに fn を呼ぶ。
// 退避中のステージの実体と、壊れて掃除を待つ置物は除く。通り方と見通しの印は地形と置物だけが持ち、
// キャラクターやアイテムは印を持たなくても塞がず遮らないので、印を持たないことで判定する側はこれで対象を絞る。
// fn はクエリを回している間に呼ばれ、ワールドがロックされているので、fn の中で実体やコンポーネントを足し引きしない。
func ForEachTerrainOrPropWithout(world w.World, fn func(ecs.Entity), markers ...ecs.Comp) {
	exclude := append([]ecs.Comp{ecs.C[gc.Dead]()}, markers...)
	for _, kind := range terrainAndPropKinds {
		q := ActiveFilter1[gc.GridElement](world).With(kind).Without(exclude...).Query()
		for q.Next() {
			fn(q.Entity())
		}
	}
}

// IsStandingTerrainOrProp は e が地形か置物で、壊れていないかを返す。1件ずつ判定する側で
// ForEachTerrainOrPropWithout と同じ対象に絞る
func IsStandingTerrainOrProp(world w.World, e ecs.Entity) bool {
	if world.Components.Dead.Has(e) {
		return false
	}
	return world.Components.Tile.Has(e) || world.Components.Prop.Has(e)
}
