package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// progressTurnHunger は Hunger を持つ現ステージの全エンティティの空腹を、1ターンにつき1回進める。ターン終了で
// 呼ぶので、移動や射撃をしない待機中心の隊員も等しく空腹になる。行動種別に紐づけると動く隊員だけ空腹になり
// 不公平だったため、活動側でなくここへ集約する。
//
// ゲートは共有 RNG を使わず、entity と turn から決定的な擬似乱数を作る。共有 RNG を消費すると隊規模ぶん敵AIや
// ドロップの乱数列がずれて再現性が壊れるため。同じ seed・同じ盤面なら空腹の進みも一意に決まる。
func progressTurnHunger(world w.World) {
	turn := query.GetTurnState(world).TurnNumber
	q := query.ActiveFilter1[gc.Hunger](world).Query()
	for q.Next() {
		entity := q.Entity()
		hungerPct := int(consts.PercentBase)
		if world.Components.CharModifiers.Has(entity) {
			hungerPct = int(world.Components.CharModifiers.Get(entity).HungerProgress)
		}
		// 分母を HungerDrainTurns 倍に伸ばして基準速度を緩和する。耐性が高く hungerPct が 0 以下なら比較が常に
		// 偽になり空腹が進まない。下限を置くかは進行系倍率の共通課題として将来まとめて検討する。
		if int(hungerNoise(entity, turn)%uint64(int(consts.PercentBase)*gc.HungerDrainTurns)) < hungerPct {
			world.Components.Hunger.Get(entity).Decrease(1)
		}
	}
}

// hungerNoise は entity と turn から決定的な擬似乱数を撹拌する。splitmix64 の finalizer を使い、共有 RNG を
// 汚さずに確率ゲートを再現する。
func hungerNoise(entity ecs.Entity, turn consts.Turn) uint64 {
	x := uint64(entity.ID())*0x9E3779B97F4A7C15 ^ uint64(turn)*0xC2B2AE3D27D4EB4F
	x ^= x >> 30
	x *= 0xBF58476D1CE4E5B9
	x ^= x >> 27
	x *= 0x94D049BB133111EB
	x ^= x >> 31
	return x
}
