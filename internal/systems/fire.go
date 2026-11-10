package systems

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// FireSystem は燃えている火を毎ターン進める。残量を一定量減らし、尽きたら Burning を外して火が消える。
// 火は燃料を貯めず残量だけを持つ。燃料は着火や給油のときに残ターン数へ畳み込まれる。
// 燃え尽きたら Dead にするだけ。燃焼も暖房も灯りも、同じターンの turn-end で dead_cleanup が
// 火のエンティティごと除去することで止まる。個々の component は外さない。
type FireSystem struct{}

// String はシステム名を返す
func (sys *FireSystem) String() string {
	return "FireSystem"
}

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
		burning.Remaining--
		if burning.Remaining <= 0 {
			// 燃料が尽きた。prop の行が燃え残りを持てば火のあった座標へ落とし、Dead にする。
			// Dead は同じターンの turn-end で dead_cleanup が火のエンティティごと除去する。Burning・HeatSource・
			// LightSource も一緒に消えるので、次ターンには燃焼も暖房も灯りも残らない
			if err := spawnBurnoutItem(world, fire); err != nil {
				return err
			}
			if !world.Components.Dead.Has(fire) {
				world.Components.Dead.Add(fire, &gc.Dead{})
			}
		}
	}

	return nil
}

// spawnBurnoutItem は燃え尽きた prop の行が持つ燃え残りのアイテムを、prop の座標へ落とす
func spawnBurnoutItem(world w.World, fire ecs.Entity) error {
	if !world.Components.GridElement.Has(fire) || !world.Components.RawID.Has(fire) {
		return nil
	}
	prop, err := raw.GetProp(world.Resources.RawMaster, world.Components.RawID.Get(fire).ID)
	if err != nil || prop.BurnoutItem == nil {
		return err
	}
	coord := world.Components.GridElement.Get(fire).Coord
	if _, err := lifecycle.SpawnFieldItem(world, *prop.BurnoutItem, coord.X, coord.Y, 1); err != nil {
		return fmt.Errorf("spawn burnout item: %w", err)
	}
	return nil
}
