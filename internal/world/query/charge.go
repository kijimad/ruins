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
// ほかは使うたびに減る
func ChargeTriggerOf(world w.World, item ecs.Entity) ChargeTrigger {
	switch {
	case item == gc.InvalidEntity || !world.Components.Charge.Has(item):
		return ChargeTriggerNone
	case world.Components.LightSource.Has(item):
		return ChargeTriggerWhileOn
	}
	return ChargeTriggerOnUse
}

// IsOn は、オンの間に減る item がいまオンかを返す。オンの間に減るのは今は灯りだけで、照らしているかで決まる。
// 灯りでない物は、スイッチを持っていてもオフとして扱う。holder は LightLit と同じく、持っている物を照らす物とみなす者
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

// FuelKindOf は燃料の fuel の種別を返す。燃料でない物は ok が偽
func FuelKindOf(world w.World, fuel ecs.Entity) (rawschema.FuelKindId, bool) {
	if !world.Components.RawID.Has(fuel) {
		return "", false
	}
	return raw.FindFuelKind(world.Resources.RawMaster, world.Components.RawID.Get(fuel).ID)
}

// FuelKindMatches は fuel が燃料で、その種別が want かを返す。灯りが受ける種別との一致を、一覧の絞り込みと補充の両方がこれで判定する
func FuelKindMatches(world w.World, want rawschema.FuelKindId, fuel ecs.Entity) bool {
	got, ok := FuelKindOf(world, fuel)
	return ok && got == want
}
