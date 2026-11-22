package gameaction

import (
	"errors"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ConsumeCharge は item の残量を amount 減らす。0になれば、燃料は空の物に変え、満たせる物は残し、使い切りの物は消す。
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
	if _, fuel := query.RefillOf(world, item); fuel {
		if byPlayer {
			logItem(world, item, "%s is used up.")
		}
		lifecycle.EmptyFuel(world, item)
		return
	}
	if _, refillable := query.AcceptedFuelOf(world, item); refillable {
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
	// ErrChargeFull は満タンの物を満たそうとしたときに返す
	ErrChargeFull = errors.New("charge is already full")
	// ErrRefillMismatch は種別の合わない燃料で満たそうとしたときに返す
	ErrRefillMismatch = errors.New("fuel kind does not match the accepted fuel")
	// ErrChargeEmpty は残量0の灯りから抜こうとしたときに返す
	ErrChargeEmpty = errors.New("charge is empty")
	// ErrNoContainer は抜いた燃料を入れる空の物を持っていないときに返す
	ErrNoContainer = errors.New("no empty container to drain into")
)

// TransferCharge は燃料の fuel から灯りの light へ、灯りの不足分と燃料の残りの少ない方を移し、移した量を返す。
// 燃料が0になれば空の物に変える
func TransferCharge(world w.World, fuel, light ecs.Entity) (int, error) {
	want, refillable := query.AcceptedFuelOf(world, light)
	if !refillable || !query.FuelKindMatches(world, want, fuel) {
		return 0, ErrRefillMismatch
	}
	lightCharge := world.Components.Charge.Get(light)
	if lightCharge.Current >= lightCharge.Max {
		return 0, ErrChargeFull
	}
	fuelCharge := world.Components.Charge.Get(fuel)
	before := lightCharge.Current
	moved := min(lightCharge.Max-lightCharge.Current, fuelCharge.Current)
	lightCharge.Current += moved
	fuelCharge.Current -= moved
	empty := fuelCharge.Current == 0
	syncCarrierLight(world, light, before)
	if empty {
		lifecycle.EmptyFuel(world, fuel)
	}
	return moved, nil
}

// DrainContainer は灯りの light から抜くときに使う、持ち主が持つ空の物と、戻す燃料の行の id を返す
func DrainContainer(world w.World, light ecs.Entity) (ecs.Entity, string, error) {
	kind, refillable := query.AcceptedFuelOf(world, light)
	if !refillable {
		return gc.InvalidEntity, "", ErrRefillMismatch
	}
	raws := world.Resources.RawMaster
	fuelID, ok := raw.FindFuelItem(raws, kind)
	if !ok {
		return gc.InvalidEntity, "", ErrRefillMismatch
	}
	refill, ok := raw.FindRefill(raws, fuelID)
	if !ok {
		return gc.InvalidEntity, "", ErrRefillMismatch
	}
	owner, carried := query.Carrier(world, light)
	if !carried {
		return gc.InvalidEntity, "", ErrNoContainer
	}
	container, ok := query.FindInBackpack(world, owner, refill.EmptiesTo)
	if !ok {
		return gc.InvalidEntity, "", ErrNoContainer
	}
	return container, fuelID, nil
}

// DrainCharge は灯りの light の持ち主が持つ空の物を1つ燃料に戻し、灯りの残量を燃料の上限まで移す。戻した燃料と移した量を返す
func DrainCharge(world w.World, light ecs.Entity) (ecs.Entity, int, error) {
	if world.Components.Charge.Get(light).Current <= 0 {
		return gc.InvalidEntity, 0, ErrChargeEmpty
	}
	container, fuelID, err := DrainContainer(world, light)
	if err != nil {
		return gc.InvalidEntity, 0, err
	}
	fuel, err := lifecycle.ReplaceItem(world, container, fuelID)
	if err != nil {
		return gc.InvalidEntity, 0, err
	}
	lightCharge := world.Components.Charge.Get(light)
	fuelCharge := world.Components.Charge.Get(fuel)
	before := lightCharge.Current
	moved := min(lightCharge.Current, fuelCharge.Max)
	fuelCharge.Current = moved
	lightCharge.Current -= moved
	syncCarrierLight(world, light, before)
	return fuel, moved, nil
}

// syncCarrierLight は灯りの light の残量が0の境目をまたいだとき、持ち主に灯りを写し直させ、視界を計算し直させる
func syncCarrierLight(world w.World, light ecs.Entity, before int) {
	if (before > 0) == (world.Components.Charge.Get(light).Current > 0) {
		return
	}
	if owner, carried := query.Carrier(world, light); carried {
		lifecycle.EnsureMarker(world, world.Components.StatsChanged, owner, &gc.StatsChanged{})
	}
	query.GetVisionState(world).RequestUpdate()
}
