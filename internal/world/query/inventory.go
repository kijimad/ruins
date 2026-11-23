package query

import (
	"fmt"
	"strings"

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
		if world.Components.RawID.Get(entity).ID == id && !partlyUsed(world, entity) {
			q.Close()
			return entity, true
		}
	}
	return gc.InvalidEntity, false
}

// partlyUsed は entity が残量を持ち、上限まで残っていないかを返す
func partlyUsed(world w.World, entity ecs.Entity) bool {
	if !world.Components.Charge.Has(entity) {
		return false
	}
	charge := world.Components.Charge.Get(entity)
	return charge.Current < charge.Max
}

// FindAmmoInInventory は口径タグでバックパック内の弾薬アイテムを検索する
func FindAmmoInInventory(world w.World, ammoTag rawschema.AmmoKindId) (ecs.Entity, bool) {
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

// FireStarterOf は owner が着火に使う火種を返す。残量を持たない物、オフの物、オンの物の順に選び、
// 同じ順位なら残量の少ない物を選ぶ。照らしている灯りを削らずに済むよう、オフの物から使う。
// 減るのを1本に集め、ほかを新品のまま束ねてクラフトの材料にも使えるようにする
func FireStarterOf(world w.World, owner ecs.Entity) (ecs.Entity, bool) {
	best := gc.InvalidEntity
	bestRank, bestCharge := 0, 0
	q := ecs.NewFilter1[gc.FireStarter](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if !CarriedBy(world, e, owner) {
			continue
		}
		charge := chargeKey(world, e)
		// 尽きて残った満たせる物は火を点けられない
		if charge == 0 {
			continue
		}
		rank := 1
		switch {
		case charge < 0:
			rank = 0
		case IsOn(world, owner, e):
			rank = 2
		}
		if best == gc.InvalidEntity || rank < bestRank || (rank == bestRank && charge < bestCharge) {
			best, bestRank, bestCharge = e, rank, charge
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

// ItemLabel は entity の名前に個数と、残量・鮮度・スイッチの印を添えた表示を返す。印は残量を先にし、数が名前のすぐ後ろに来るようにする。
// 腐敗食は新鮮以外のとき鮮度を添え、状態を見て食べるか捨てるか判断できるようにする
func ItemLabel(world w.World, entity ecs.Entity, count int) string {
	var label strings.Builder
	label.WriteString(FormatNameCount(GetEntityName(entity, world), count))
	for _, marker := range []string{ChargeMarker(world, entity), FreshnessMarker(world, entity), SwitchMarker(world, entity)} {
		if marker != "" {
			label.WriteString(" (" + marker + ")")
		}
	}
	return label.String()
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
