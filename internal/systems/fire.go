package systems

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
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

// fireBurnPerTurn は1ターンで減る燃焼ターン数。残量は毎ターンこれだけ減る
const fireBurnPerTurn consts.Turn = 1

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
			// 燃料が尽きた。燃え尽きた跡へ灰を残してから、暖房を切り火のエンティティごと消す。
			// 灰は火のあった座標へ落とすフィールドアイテムで、拾って素材に使える。
			// 暖房を切るのは HeatSource を外すこと。暖房は HeatSource だけで決まり Burning と独立なので、
			// 火が燃え尽きた側で自分の熱源を落とす。Dead にすると dead_cleanup がスプライトの
			// フェードアウトを出して除去し、燃え尽きた火は残らない
			if world.Components.GridElement.Has(fire) {
				coord := world.Components.GridElement.Get(fire).Coord
				if _, err := lifecycle.SpawnFieldItem(world, "ashes", coord.X, coord.Y, 1); err != nil {
					return fmt.Errorf("spawn ashes: %w", err)
				}
			}
			world.Components.Burning.Remove(fire)
			if world.Components.HeatSource.Has(fire) {
				world.Components.HeatSource.Remove(fire)
			}
			if !world.Components.Dead.Has(fire) {
				world.Components.Dead.Add(fire, &gc.Dead{})
			}
		}
	}

	return nil
}
