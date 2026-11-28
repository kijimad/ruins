package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// deployedFuelTurns は展開中のキューブの燃料を1戻す間隔のターン数
const deployedFuelTurns consts.Turn = 10

// DeployedFuelSystem は展開中のキューブの燃料を、deployedFuelTurns ターンごとに1ずつ容量まで戻す。
// 拠点を置いている間に燃料が尽きて詰むのを防ぐ
type DeployedFuelSystem struct{}

// String はシステム名を返す
func (sys *DeployedFuelSystem) String() string {
	return "DeployedFuelSystem"
}

// Update は間隔のターンにだけ燃料を戻す
func (sys *DeployedFuelSystem) Update(world w.World) error {
	if query.GetGameTime(world).TotalTurns%deployedFuelTurns != 0 {
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
