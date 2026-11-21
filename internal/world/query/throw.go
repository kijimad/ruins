package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

const (
	// throwMaxWeight を超える物は投げられない
	throwMaxWeight = 5 * consts.MilligramPerKg
	// throwBaseAccuracy は重さから導く命中の基準
	throwBaseAccuracy = 70
	// throwMaxWeightDamage は重さから導く威力の上限。投げ向きの物は欄でこれを上回る
	throwMaxWeightDamage = 5
)

// CanThrow は物が重すぎず投げられるかを返す。5 kg を超える物は投げられない
func CanThrow(world w.World, item ecs.Entity) bool {
	return itemWeight(world, item) <= throwMaxWeight
}

// ThrowAttackOf は物を投げたときの攻撃の値を返す。raw の行に投げの欄があればその値、無ければ重さから導く。
// 重い物ほど当たれば痛いが当てにくい。投げられない物には ok=false を返す
func ThrowAttackOf(world w.World, item ecs.Entity) (gc.ThrowAttack, bool) {
	if !CanThrow(world, item) {
		return gc.ThrowAttack{}, false
	}
	mg := itemWeight(world, item)
	if attack, ok := raw.FindThrow(world.Resources.RawMaster, GetEntityID(item, world)); ok {
		return attack, true
	}
	return throwAttackFromWeight(mg), true
}

// throwAttackFromWeight は重さから投げの値を導く。威力は 1 kg あたり 1、命中は 1 kg あたり 5 下がる。どちらも四捨五入する
func throwAttackFromWeight(mg consts.Milligram) gc.ThrowAttack {
	kg := int(consts.MilligramPerKg)
	damage := (int(mg) + kg/2) / kg
	damage = max(1, min(throwMaxWeightDamage, damage))
	accuracy := throwBaseAccuracy - (int(mg)*5+kg/2)/kg
	return gc.ThrowAttack{Damage: damage, Accuracy: accuracy}
}

// itemWeight は物の重さを返す。重さを持たない物は 0 とする
func itemWeight(world w.World, item ecs.Entity) consts.Milligram {
	if !world.Components.Weight.Has(item) {
		return 0
	}
	return world.Components.Weight.Get(item).Milligram
}
