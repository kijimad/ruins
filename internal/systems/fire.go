package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// FireSystem は燃えている火を毎ターン進める。残量を一定量減らし、尽きたら収納の次の燃料へ移す。
// 収納が空で残量も尽きると Burning を外して火が消える。収納の上段から1つずつ燃やす。
// 光には関与しない。暖房が切れるのは Burning が外れて heatSourceWarmthAt が数えなくなる結果として起きる。
type FireSystem struct{}

// String はシステム名を返す
func (sys *FireSystem) String() string {
	return "FireSystem"
}

// fireBurnPerTurn は1ターンで燃える量。燃料の HeatContent はこの単位で釣り合わせる
const fireBurnPerTurn = 1

// Update は火の残量を減らし、尽きたら収納の次の燃料へ移し、燃料が無ければ鎮火する
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
		if burning.Remaining > 0 {
			continue
		}

		// 最上段の燃料が尽きた。収納の次の燃料へ移る。無ければ鎮火する。
		// Burning を外すと heatSourceWarmthAt が数えなくなり暖房が切れる
		if !lifecycle.LoadNextFuel(world, fire) {
			world.Components.Burning.Remove(fire)
		}
	}

	return nil
}
