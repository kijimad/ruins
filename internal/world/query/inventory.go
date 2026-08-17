package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// FindStackableInInventory は RawID でバックパック内の Stackable アイテムを1つ返す。
// 個数照会や消費の起点に使う。鮮度による束の分割は合流側の query.SameStack が担うため、
// ここは鮮度を見ず RawID だけで引く。腐敗食は鮮度違いの別束が複数ありうる点に注意する。
func FindStackableInInventory(world w.World, id string) (ecs.Entity, bool) {
	var foundEntity ecs.Entity
	var found bool

	stackableQuery := ecs.NewFilter3[gc.Stackable, gc.LocationInBackpack, gc.RawID](world.ECS).Query()
	for stackableQuery.Next() {
		entity := stackableQuery.Entity()
		if found {
			continue
		}
		if world.Components.RawID.Get(entity).ID == id {
			foundEntity = entity
			found = true
		}
	}

	return foundEntity, found
}

// FindAmmoInInventory は口径タグでバックパック内の弾薬アイテムを検索する
func FindAmmoInInventory(world w.World, ammoTag oapi.AmmoTag) (ecs.Entity, bool) {
	var foundEntity ecs.Entity
	var found bool

	ammoQuery := ecs.NewFilter3[gc.Stackable, gc.LocationInBackpack, gc.Ammo](world.ECS).Query()
	for ammoQuery.Next() {
		entity := ammoQuery.Entity()
		if found {
			continue
		}
		ammo := world.Components.Ammo.Get(entity)
		if ammo.AmmoTag == ammoTag {
			foundEntity = entity
			found = true
		}
	}

	return foundEntity, found
}

// GetEntityCount はエンティティの個数を返す。
// Stackableであれば Stackable.Count を返し、そうでなければ1を返す。
func GetEntityCount(world w.World, entity ecs.Entity) int {
	if world.Components.Stackable.Has(entity) {
		return world.Components.Stackable.Get(entity).Count
	}
	return 1
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
