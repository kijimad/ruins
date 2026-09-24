package lifecycle

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// FacilityCellFree は展開空間のマスが設備を据えられる空きかを返す。壁・キャラ・prop・アイテムが
// 1つでもあれば偽。キューブ自身のマスは据付先にしない。メニューを開いたプレイヤーのマスは障害物として
// 見ない。deploySpaceFree の判定を1マスに絞ったもので、設置 UI のマス表示と PlaceFacility の
// 最終ガードが同じ判定を共有する。
func FacilityCellFree(world w.World, cube ecs.Entity, coord consts.Coord[consts.Tile]) bool {
	if world.Components.GridElement.Get(cube).Coord == coord {
		return false
	}
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

// PlaceFacility は据付アイテムをキューブの展開空間の coord へ据える。実体は削除せず、携行アイテムから
// フィールドの造作へマーカーを入れ替えてロケーションを移す。バックパックでは Item、据わっている間は Prop に
// なり、破壊・掃除・静物描画といった造作のパイプラインに乗る。効果 WeightCapacity 等は据付定義から実体に
// 内在するのでそのまま働く。装備がアイテムをスロットへ移すのと同じで、2 次元スロットがフィールドのマスにあたる。
// マスが空いていなければ据えずに error を返し、UI とロジックの食い違いを早期に検知する。
func PlaceFacility(world w.World, cube ecs.Entity, item ecs.Entity, coord consts.Coord[consts.Tile]) (ecs.Entity, error) {
	if !world.Components.Deployable.Has(item) {
		return gc.InvalidEntity, fmt.Errorf("place facility: item is not deployable")
	}
	if !FacilityCellFree(world, cube, coord) {
		return gc.InvalidEntity, fmt.Errorf("place facility: cell %s is occupied", coord)
	}
	// バックパックから移すので、前の所有者を渡して総重量の再計算を促す
	var previousOwner ecs.Entity
	if world.Components.LocationInBackpack.Has(item) {
		previousOwner = world.Components.LocationInBackpack.Get(item).Owner
	}
	// 携行アイテムからフィールドの造作へ。Prop になれば拾得対象から外れ、造作として振る舞う
	ensureRemoved(world.Components.Item, item)
	ensureMarker(world, world.Components.Prop, item, &gc.Prop{})
	MoveMembersToField(world, []ecs.Entity{item}, coord, previousOwner)
	return item, nil
}

// RemoveFacility は据えた設備をプレイヤーのバックパックへ戻す。実体は削除せず、造作から携行アイテムへ
// マーカーを入れ替えてロケーションを移す。収納設備に中身が残っているときは孤児化を避けるため戻さず
// error を返す。空にしてから撤去する。
func RemoveFacility(world w.World, facility ecs.Entity, player ecs.Entity) error {
	if !world.Components.Deployable.Has(facility) {
		return fmt.Errorf("remove facility: entity is not deployable")
	}
	if len(query.GetStorageItems(world, facility)) > 0 {
		return fmt.Errorf("remove facility: storage is not empty")
	}
	// フィールドの造作から携行アイテムへ戻す
	ensureRemoved(world.Components.Prop, facility)
	ensureMarker(world, world.Components.Item, facility, &gc.Item{})
	return MoveToBackpack(world, facility, player)
}
