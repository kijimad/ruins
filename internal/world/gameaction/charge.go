package gameaction

import (
	"errors"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/i18n"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ConsumeCharge は Charge を持つ item の残量を amount 減らす。0になったら、満たせる物は残し、使い切りの物は消す。
// どちらも視界を計算し直させる。プレイヤーが持つ物なら尽きたことをログに出す。減っていく途中はゲージで見せるのでログを出さない
func ConsumeCharge(world w.World, item ecs.Entity, amount int) {
	charge := world.Components.Charge.Get(item)
	charge.Current = max(charge.Current-amount, 0)
	if charge.Current > 0 {
		return
	}
	owner, carried := carrier(world, item)
	byPlayer := carried && world.Components.Player.Has(owner)
	if _, refillable := query.RefillTagOf(world, item); refillable {
		if byPlayer {
			logItem(world, item, "%s ran out.")
		}
		// 物は残るので、持ち主の灯りを写し直させる
		if carried && !world.Components.StatsChanged.Has(owner) {
			world.Components.StatsChanged.Add(owner, &gc.StatsChanged{})
		}
		query.GetVisionState(world).RequestUpdate()
		return
	}
	if byPlayer {
		if world.Components.LightSource.Has(item) {
			logItem(world, item, "%s burned out.")
		} else {
			logItem(world, item, "%s broke.")
		}
	}
	lifecycle.RemoveItem(world, item)
	query.GetVisionState(world).RequestUpdate()
}

// logItem は item の名前を埋めたログを1行出す
func logItem(world w.World, item ecs.Entity, msgid i18n.Msgid) {
	name := gamelog.Tag("item", query.GetEntityName(item, world))
	gamelog.New(query.GetGameLog(world)).Markup(query.T(world, msgid, name)).Log()
}

// carrier は item を持つ者を返す。バックパックか装備になければ ok は偽
func carrier(world w.World, item ecs.Entity) (ecs.Entity, bool) {
	switch {
	case world.Components.LocationEquipped.Has(item):
		return world.Components.LocationEquipped.Get(item).Owner, true
	case world.Components.LocationInBackpack.Has(item):
		return world.Components.LocationInBackpack.Get(item).Owner, true
	}
	return gc.InvalidEntity, false
}

// ErrChargeFull は満タンの物を満たそうとしたときに返す
var ErrChargeFull = errors.New("charge is already full")

// RefillCharge は fuel を1つ使い、item の残量を上限まで戻す。満タンなら燃料を使わず ErrChargeFull を返す。
// 尽きていた灯りが照らし直すので、持ち主の灯りと視界を計算し直させる
func RefillCharge(world w.World, item, fuel ecs.Entity) error {
	charge := world.Components.Charge.Get(item)
	if charge.Current >= charge.Max {
		return ErrChargeFull
	}
	if err := lifecycle.ChangeItemCount(world, fuel, -1); err != nil {
		return err
	}
	charge = world.Components.Charge.Get(item)
	charge.Current = charge.Max
	if owner, carried := carrier(world, item); carried && !world.Components.StatsChanged.Has(owner) {
		world.Components.StatsChanged.Add(owner, &gc.StatsChanged{})
	}
	query.GetVisionState(world).RequestUpdate()
	return nil
}
