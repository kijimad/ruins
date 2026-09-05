package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// fatigueRecoverPerTurn は睡眠中に1ターンで抜ける疲労の基準量。寝具 Quality で乗算する
const fatigueRecoverPerTurn = 6

// progressTurnFatigue は Fatigue を持つ現ステージの全エンティティの疲労を1ターン進める。
// 起床中は蓄積し、睡眠中は寝具 Quality に比例して減る。空腹と同じくターン終了で呼び、
// 行動種別に依らず時間経過で溜まる。Current は 0..Max にクランプし、上限でも死なせない。
func progressTurnFatigue(world w.World) {
	q := query.ActiveFilter1[gc.Fatigue](world).Query()
	for q.Next() {
		entity := q.Entity()
		fatigue := world.Components.Fatigue.Get(entity)

		if world.Components.Sleeping.Has(entity) {
			quality := world.Components.Sleeping.Get(entity).Quality
			fatigue.Current -= quality.ApplyInt(fatigueRecoverPerTurn)
		} else {
			fatigue.Current += gc.FatigueGainPerTurn
		}

		// 0..Max に収める。上限でも死なせず Exhausted のペナルティが続く
		fatigue.Current = max(0, min(fatigue.Current, fatigue.Max))
	}
}
