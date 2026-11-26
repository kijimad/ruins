package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// ForEachImpassable は現ステージの壊れていない通れない実体ごとに fn を呼ぶ。
// クエリの最中に呼ぶので、fn の中で実体やコンポーネントを足し引きしない
func ForEachImpassable(world w.World, fn func(ecs.Entity)) {
	q := ActiveFilter2[gc.GridElement, gc.Passage](world).Without(ecs.C[gc.Dead]()).Query()
	for q.Next() {
		if !world.Components.Passage.Get(q.Entity()).Passable {
			fn(q.Entity())
		}
	}
}
