package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// FindStackInInventory は RawID でバックパック内のスタックアイテムを1つ返す。
// クラフトの個数照会や消費の起点に使う。残量を持つ物は新品だけを選び、使いかけの物は材料にしない。
// 鮮度による束の分割は合流側の query.SameStack が担うため、ここは鮮度を見ない。腐敗食は鮮度違いの別束が複数ありうる点に注意する。
func FindStackInInventory(world w.World, id string) (ecs.Entity, bool) {
	q := ecs.NewFilter2[gc.LocationInBackpack, gc.RawID](world.ECS).Query()
	for q.Next() {
		entity := q.Entity()
		if world.Components.RawID.Get(entity).ID == id && !worn(world, entity) {
			q.Close()
			return entity, true
		}
	}
	return gc.InvalidEntity, false
}

// worn は entity が残量を持ち、上限まで残っていないかを返す
func worn(world w.World, entity ecs.Entity) bool {
	if !world.Components.Charge.Has(entity) {
		return false
	}
	charge := world.Components.Charge.Get(entity)
	return charge.Current < charge.Max
}

// FindAmmoInInventory は口径タグでバックパック内の弾薬アイテムを検索する
func FindAmmoInInventory(world w.World, ammoTag rawschema.AmmoTag) (ecs.Entity, bool) {
	q := ecs.NewFilter2[gc.LocationInBackpack, gc.Ammo](world.ECS).Query()
	for q.Next() {
		entity := q.Entity()
		if world.Components.Ammo.Get(entity).AmmoTag == ammoTag {
			q.Close()
			return entity, true
		}
	}
	return gc.InvalidEntity, false
}

// HoldsFireStarter は owner が火種の道具を所持しているかを返す。バックパックと装備の両方を見る
func HoldsFireStarter(world w.World, owner ecs.Entity) bool {
	_, ok := FireStarterOf(world, owner)
	return ok
}

// FireStarterOf は owner が着火に使う火種を返す。残量を持たない物、バックパックの物、装備の物の順に選び、
// 同じ順位なら残量の少ない物を選ぶ。照らしている装備の灯りを削らずに済むよう、照らしていない物から使う。
// 減るのを1本に集め、ほかを新品のまま束ねてクラフトの材料にも使えるようにする
func FireStarterOf(world w.World, owner ecs.Entity) (ecs.Entity, bool) {
	best := gc.InvalidEntity
	bestRank, bestCharge := 0, 0
	consider := func(e ecs.Entity, rank int) {
		charge := chargeKey(world, e)
		if charge < 0 {
			rank = 0
		}
		if best == gc.InvalidEntity || rank < bestRank || (rank == bestRank && charge < bestCharge) {
			best, bestRank, bestCharge = e, rank, charge
		}
	}
	backpack := ecs.NewFilter2[gc.LocationInBackpack, gc.FireStarter](world.ECS).Query()
	for backpack.Next() {
		if world.Components.LocationInBackpack.Get(backpack.Entity()).Owner == owner {
			consider(backpack.Entity(), 1)
		}
	}
	equipped := ecs.NewFilter2[gc.LocationEquipped, gc.FireStarter](world.ECS).Query()
	for equipped.Next() {
		if world.Components.LocationEquipped.Get(equipped.Entity()).Owner == owner {
			consider(equipped.Entity(), 2)
		}
	}
	return best, best != gc.InvalidEntity
}

// GetEntityCount は entity が属するスタックの個数を返す。個数は保存せず、同じ所有者かつ
// 同じ位置にある同一スタックのエンティティを数えて導出する。位置が無ければ1になる。
func GetEntityCount(world w.World, entity ecs.Entity) int {
	return len(StackMembers(world, entity))
}

// FormatNameCount は個数が2以上のとき名前の前に個数を置く。1個や非スタックは名前だけを返す。
// メニュー行とログの両方がこの1関数を通し、"個数 名前" の表記を揃える
func FormatNameCount(name string, count int) string {
	if count > 1 {
		return fmt.Sprintf("%d %s", count, name)
	}
	return name
}

// FormatItemName はアイテムエンティティから名前と個数を取得してフォーマットする。
// 名前はNameコンポーネントから取得し、見つからない場合は "Unknown Item" を返す。
// 表記は FormatNameCount に委ね、個数が2以上のとき "個数 名前" になる
func FormatItemName(world w.World, itemEntity ecs.Entity) string {
	name := T(world, "Unknown Item")
	if nameComp := world.Components.Name.Get(itemEntity); nameComp != nil {
		name = T(world, nameComp.Name)
	}
	return FormatNameCount(name, GetEntityCount(world, itemEntity))
}
