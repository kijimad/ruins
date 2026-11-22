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

// ConsumeCharge は item の残量を amount 減らす。0になれば、燃料と満たせる灯りは残し、使い切りの物は消す。
// ログは尽きたときだけ出す。減っていく途中はゲージで見せる
func ConsumeCharge(world w.World, item ecs.Entity, amount int) {
	charge := world.Components.Charge.Get(item)
	before := charge.Current
	charge.Current = max(charge.Current-amount, 0)
	if charge.Current > 0 {
		return
	}
	owner, carried := query.Carrier(world, item)
	byPlayer := carried && world.Components.Player.Has(owner)
	if _, kept := query.ChargeKindOf(world, item); kept {
		if byPlayer {
			logItem(world, item, "%s ran out.")
		}
		syncCarrierLight(world, item, before)
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
	// ErrChargeFull は満タンの物へ移そうとしたときに返す
	ErrChargeFull = errors.New("charge is already full")
	// ErrChargeEmpty は残量0の物から移そうとしたときに返す
	ErrChargeEmpty = errors.New("charge is empty")
	// ErrRefillMismatch は種別の合わない物の間で移そうとしたときに返す
	ErrRefillMismatch = errors.New("fuel kind does not match the accepted fuel")
)

// TransferCharge は from から to へ、to の空きと from の残りの少ない方を移し、移した量を返す。
// 燃料から灯りへ満たすのも、灯りから燃料へ抜くのもこれを通す
func TransferCharge(world w.World, from, to ecs.Entity) (int, error) {
	fromKind, fromOK := query.ChargeKindOf(world, from)
	toKind, toOK := query.ChargeKindOf(world, to)
	if from == to || !fromOK || !toOK || fromKind != toKind {
		return 0, ErrRefillMismatch
	}
	fromCharge := world.Components.Charge.Get(from)
	toCharge := world.Components.Charge.Get(to)
	if toCharge.Current >= toCharge.Max {
		return 0, ErrChargeFull
	}
	if fromCharge.Current <= 0 {
		return 0, ErrChargeEmpty
	}
	fromBefore, toBefore := fromCharge.Current, toCharge.Current
	moved := min(toCharge.Max-toCharge.Current, fromCharge.Current)
	fromCharge.Current -= moved
	toCharge.Current += moved
	syncCarrierLight(world, from, fromBefore)
	syncCarrierLight(world, to, toBefore)
	return moved, nil
}

// syncCarrierLight は灯りの light の残量が0の境目をまたいだとき、持ち主に灯りを写し直させ、視界を計算し直させる。灯りでない物には何もしない
func syncCarrierLight(world w.World, light ecs.Entity, before int) {
	if !world.Components.LightSource.Has(light) || (before > 0) == (world.Components.Charge.Get(light).Current > 0) {
		return
	}
	if owner, carried := query.Carrier(world, light); carried {
		lifecycle.EnsureMarker(world, world.Components.StatsChanged, owner, &gc.StatsChanged{})
	}
	query.GetVisionState(world).RequestUpdate()
}
