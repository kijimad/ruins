package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// FireSystem は燃えている火を毎ターン進める。残量を一定量減らし、尽きたら Burning を外して火が消える。
// 火は燃料を貯めず残量だけを持つ。燃料は着火や給油のときに残ターン数へ畳み込まれる。
// 光には関与しない。暖房が切れるのは Burning が外れて heatSourceWarmthAt が数えなくなる結果として起きる。
type FireSystem struct{}

// String はシステム名を返す
func (sys *FireSystem) String() string {
	return "FireSystem"
}

// fireBurnPerTurn は1ターンで燃える量。燃料の HeatContent はこの単位で釣り合わせる
const fireBurnPerTurn = 1

// Update は火の残量を減らし、尽きたら鎮火する
func (sys *FireSystem) Update(world w.World) error {
	// 反復中の構造変更を避けるため、火を集めてからループ後に処理する
	var fires []ecs.Entity
	fireQuery := query.ActiveFilter1[gc.Burning](world).Query()
	for fireQuery.Next() {
		fires = append(fires, fireQuery.Entity())
	}

	for _, fire := range fires {
		burning := world.Components.Burning.Get(fire)
		burning.Remaining -= fireBurnPerTurn
		if burning.Remaining <= 0 {
			// 燃料が尽きた。暖房を切り、火のエンティティごと消す。Burning を外すと
			// heatSourceWarmthAt が数えなくなり暖房が切れる。Dead にすると dead_cleanup が
			// スプライトのフェードアウトを出して除去し、燃え尽きた火が残らない
			world.Components.Burning.Remove(fire)
			if !world.Components.Dead.Has(fire) {
				world.Components.Dead.Add(fire, &gc.Dead{})
			}
		}
	}

	return nil
}
