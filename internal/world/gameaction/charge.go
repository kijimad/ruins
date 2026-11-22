package gameaction

import (
	"errors"

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
	owner, carried := query.Carrier(world, item)
	byPlayer := carried && world.Components.Player.Has(owner)
	if _, refillable := query.AcceptedFuelOf(world, item); refillable {
		if byPlayer {
			logItem(world, item, "%s ran out.")
		}
		// 物は残るので、持ち主の灯りを写し直させる
		if carried {
			lifecycle.MarkStatsChanged(world, owner)
		}
		query.GetVisionState(world).RequestUpdate()
		return
	}
	if byPlayer {
		logItem(world, item, "%s is used up.")
	}
	lifecycle.RemoveItem(world, item)
	query.GetVisionState(world).RequestUpdate()
}

// logItem は item の名前を埋めたログを1行出す
func logItem(world w.World, item ecs.Entity, msgid i18n.Msgid) {
	name := gamelog.Tag("item", query.GetEntityName(item, world))
	gamelog.New(query.GetGameLog(world)).Markup(query.T(world, msgid, name)).Log()
}

var (
	// ErrChargeFull は満タンの物を満たそうとしたときに返す
	ErrChargeFull = errors.New("charge is already full")
	// ErrRefillMismatch は種別の合わない燃料で満たそうとしたときに返す
	ErrRefillMismatch = errors.New("fuel kind does not match the accepted fuel")
)

// RefillCharge は fuel を1つ使い、item の残量を上限まで戻す。種別が合わなければ ErrRefillMismatch を、
// 満タンなら ErrChargeFull を返し、どちらも燃料を使わない。
// 尽きていた灯りが照らし直すので、持ち主の灯りと視界を計算し直させる
func RefillCharge(world w.World, item, fuel ecs.Entity) error {
	want, refillable := query.AcceptedFuelOf(world, item)
	if !refillable || !query.FuelKindMatches(world, want, fuel) {
		return ErrRefillMismatch
	}
	charge := world.Components.Charge.Get(item)
	if charge.Current >= charge.Max {
		return ErrChargeFull
	}
	if err := lifecycle.ChangeItemCount(world, fuel, -1); err != nil {
		return err
	}
	charge = world.Components.Charge.Get(item)
	charge.Current = charge.Max
	if owner, carried := query.Carrier(world, item); carried {
		lifecycle.MarkStatsChanged(world, owner)
	}
	query.GetVisionState(world).RequestUpdate()
	return nil
}
