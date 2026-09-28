package lifecycle

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// FacilityCellFree は展開空間のマスが設備を装着できる空きかを返す。壁・キャラ・prop・アイテムが
// 1つでもあれば偽。キューブ自身のマスは装着先にしない。メニューを開いたプレイヤーのマスは障害物として
// 見ない。deploySpaceFree の判定を1マスに絞ったもので、装着 UI のマス表示と PlaceFacility の
// 最終ガードが同じ判定を共有する。
func FacilityCellFree(world w.World, cube ecs.Entity, coord consts.Coord[consts.Tile]) bool {
	if world.Components.GridElement.Get(cube).Coord == coord {
		return false
	}
	// 空間索引が未構築なら nil。その間は空きなし扱いで装着を拒否し、安全側に倒す
	si := query.GetSpatialIndex(world)
	if si == nil {
		return false
	}
	if si.IsBlockPass(coord) {
		return false
	}
	// 取得失敗時は InvalidEntity になるが実キャラと一致しないので、誰も除外しないだけで安全側に倒れる
	player, _ := query.GetPlayerEntity(world)
	if e, ok := si.CharacterAt(coord); ok && e != player {
		return false
	}
	// 空間索引は prop・アイテムを持たないので LocationOnField を走査する
	q := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	defer q.Close()
	for q.Next() {
		e := q.Entity()
		if e == cube || !world.Components.GridElement.Has(e) {
			continue
		}
		if world.Components.GridElement.Get(e).Coord != coord {
			continue
		}
		if world.Components.Prop.Has(e) || world.Components.Item.Has(e) {
			return false
		}
	}
	return true
}

// PlaceFacility は装着アイテムをキューブの展開空間の coord へ装着する。実体は削除せず、Item を外し Prop を
// 付けてロケーションをフィールドへ移す。マスが空いていなければ装着せず error を返す。
func PlaceFacility(world w.World, cube ecs.Entity, item ecs.Entity, coord consts.Coord[consts.Tile]) (ecs.Entity, error) {
	if !world.Components.Deployable.Has(item) {
		return gc.InvalidEntity, fmt.Errorf("place facility: item is not deployable")
	}
	// 装着は設備画面がバックパックの装着アイテムから呼ぶ。他ロケーションからの経路は現状無いが、
	// 前提を握りつぶさず error にして、将来の経路追加で総重量の再計算が漏れるのを早期に検知する
	if !world.Components.LocationInBackpack.Has(item) {
		return gc.InvalidEntity, fmt.Errorf("place facility: item is not in backpack")
	}
	if !FacilityCellFree(world, cube, coord) {
		return gc.InvalidEntity, fmt.Errorf("place facility: cell %s is occupied", coord)
	}
	// 前の所有者を渡して総重量の再計算を促す
	owner := world.Components.LocationInBackpack.Get(item).Owner
	ensureRemoved(world.Components.Item, item)
	ensureMarker(world, world.Components.Prop, item, &gc.Prop{})
	MoveMembersToField(world, []ecs.Entity{item}, coord, owner)
	return item, nil
}

// RemoveFacility は装着した設備をプレイヤーのバックパックへ戻す。実体は削除せず、Prop を外し Item を付けて
// 移す。収納に中身が残るときは孤児化を避けるため戻さず error を返す。
func RemoveFacility(world w.World, facility ecs.Entity, player ecs.Entity) error {
	if !world.Components.Deployable.Has(facility) {
		return fmt.Errorf("remove facility: entity is not deployable")
	}
	if len(query.GetStorageItems(world, facility)) > 0 {
		return fmt.Errorf("remove facility: storage is not empty")
	}
	ensureRemoved(world.Components.Prop, facility)
	ensureMarker(world, world.Components.Item, facility, &gc.Item{})
	return MoveToBackpack(world, facility, player)
}
