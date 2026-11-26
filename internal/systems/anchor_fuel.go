package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// anchorFuelTurns は展開中のキューブの燃料を1戻す間隔のターン数
const anchorFuelTurns consts.Turn = 10

// AnchorFuelSystem は展開中のキューブの燃料を、anchorFuelTurns ターンごとに1ずつ容量まで戻す。
// 錨の間に燃料が尽きて詰むのを防ぐためのもので、遠征で持ち帰る燃料よりずっと少ない
type AnchorFuelSystem struct{}

// String はシステム名を返す
func (sys *AnchorFuelSystem) String() string {
	return "AnchorFuelSystem"
}

// Update は経過ターンが間隔で割り切れるターンにだけ、現ステージの展開中のキューブの燃料を1戻す
func (sys *AnchorFuelSystem) Update(world w.World) error {
	if query.GetGameTime(world).TotalTurns%anchorFuelTurns != 0 {
		return nil
	}
	q := query.ActiveFilter3[gc.Cube, gc.Deployed, gc.Charge](world).Query()
	for q.Next() {
		_, _, tank := q.Get()
		if tank.Current < tank.Max {
			tank.Current++
		}
	}
	return nil
}
