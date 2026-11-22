package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// ChargeTrigger は残量が減るきっかけ
type ChargeTrigger int

const (
	// ChargeTriggerNone は残量を持たない
	ChargeTriggerNone ChargeTrigger = iota
	// ChargeTriggerWhileOn はオンになっている1ターンごとに減る
	ChargeTriggerWhileOn
	// ChargeTriggerOnUse は使う行動1回ごとに減る
	ChargeTriggerOnUse
)

// ChargeTriggerOf は item の残量が何で減るかを、持つコンポーネントから導く。光源を持つ物はオンの間に減り、
// ほかは使うたびに減る。燃料もほかに入るが、燃料は武器にも灯りにもならず、灯りへ移すときだけ減る
func ChargeTriggerOf(world w.World, item ecs.Entity) ChargeTrigger {
	switch {
	case item == gc.InvalidEntity || !world.Components.Charge.Has(item):
		return ChargeTriggerNone
	case world.Components.LightSource.Has(item):
		return ChargeTriggerWhileOn
	}
	return ChargeTriggerOnUse
}

// IsOn は、オンの間に減る item がいまオンかを返す。今は灯りだけで、照らしているかで決まる
func IsOn(world w.World, holder, item ecs.Entity) bool {
	if world.Components.LightSource.Has(item) {
		return LightLit(world, holder, item)
	}
	return false
}

// ChargeRatio は残量の上限に対する割合を 0 から 1 で返す。残量を持たなければ ok は偽。上限は raw で1以上に決まる
func ChargeRatio(world w.World, entity ecs.Entity) (float64, bool) {
	if !world.Components.Charge.Has(entity) {
		return 0, false
	}
	charge := world.Components.Charge.Get(entity)
	return max(0, min(1, float64(charge.Current)/float64(charge.Max))), true
}

// AcceptedFuelOf は item が受ける燃料の種別を返す。使い切りの物は ok が偽
func AcceptedFuelOf(world w.World, item ecs.Entity) (rawschema.FuelKindId, bool) {
	if !world.Components.RawID.Has(item) {
		return "", false
	}
	return raw.FindAcceptedFuel(world.Resources.RawMaster, world.Components.RawID.Get(item).ID)
}

// RefillOf は燃料の fuel の燃料の欄を返す。燃料でない物は ok が偽
func RefillOf(world w.World, fuel ecs.Entity) (rawschema.Refill, bool) {
	if !world.Components.RawID.Has(fuel) {
		return rawschema.Refill{}, false
	}
	return raw.FindRefill(world.Resources.RawMaster, world.Components.RawID.Get(fuel).ID)
}

// FuelKindMatches は fuel が燃料で、その種別が want かを返す
func FuelKindMatches(world w.World, want rawschema.FuelKindId, fuel ecs.Entity) bool {
	refill, ok := RefillOf(world, fuel)
	return ok && refill.Kind == want
}

// ChargeRandomOnLoot は item が見つかったときにランダムな残量で生まれるかを返す。残量を移し合う燃料と、燃料を受ける灯りが当たる
func ChargeRandomOnLoot(world w.World, item ecs.Entity) bool {
	if _, ok := RefillOf(world, item); ok {
		return true
	}
	_, ok := AcceptedFuelOf(world, item)
	return ok
}
