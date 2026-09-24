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

// PlaceFacility は据付アイテムをキューブの展開空間の coord へ設備 prop として据える。据付アイテムの
// PropID の prop を生成し、撤去で戻せるよう据付アイテムの RawID を prop の DeployedFacility へ写す。
// 据付アイテムはフィールドへ出さず消費する。マスが空いていなければ据えずに error を返し、UI とロジックの
// 食い違いを早期に検知する。
func PlaceFacility(world w.World, cube ecs.Entity, item ecs.Entity, coord consts.Coord[consts.Tile]) (ecs.Entity, error) {
	if !world.Components.Deployable.Has(item) {
		return gc.InvalidEntity, fmt.Errorf("place facility: item is not deployable")
	}
	if !world.Components.RawID.Has(item) {
		return gc.InvalidEntity, fmt.Errorf("place facility: item has no raw id")
	}
	if !FacilityCellFree(world, cube, coord) {
		return gc.InvalidEntity, fmt.Errorf("place facility: cell %s is occupied", coord)
	}
	propID := world.Components.Deployable.Get(item).PropID
	itemID := world.Components.RawID.Get(item).ID
	prop, err := SpawnProp(world, propID, coord.X, coord.Y)
	if err != nil {
		return gc.InvalidEntity, err
	}
	world.Components.DeployedFacility.Add(prop, &gc.DeployedFacility{ItemID: itemID})
	// 据付アイテムを消費する。所有者の総重量が変わるので WeightDirty を立てる
	if world.Components.LocationInBackpack.Has(item) {
		owner := world.Components.LocationInBackpack.Get(item).Owner
		ensureMarker(world, world.Components.WeightDirty, owner, &gc.WeightDirty{})
	}
	world.ECS.RemoveEntity(item)
	return prop, nil
}

// RemoveFacility は据えた設備 prop を撤去し、覚えている据付アイテムをプレイヤーのバックパックへ戻す。
// 収納設備に中身が残っているときは孤児化を避けるため撤去せず error を返す。空にしてから撤去する。
// prop は削除する。DeployedFacility を持たない prop はここへ来ない。
func RemoveFacility(world w.World, prop ecs.Entity, player ecs.Entity) error {
	if !world.Components.DeployedFacility.Has(prop) {
		return fmt.Errorf("remove facility: prop is not a deployed facility")
	}
	if len(query.GetStorageItems(world, prop)) > 0 {
		return fmt.Errorf("remove facility: storage is not empty")
	}
	itemID := world.Components.DeployedFacility.Get(prop).ItemID
	item, err := spawnItemBase(world, itemID)
	if err != nil {
		return fmt.Errorf("remove facility: %w", err)
	}
	if err := MoveToBackpack(world, item, player); err != nil {
		return err
	}
	world.ECS.RemoveEntity(prop)
	return nil
}
