package lifecycle

import (
	"errors"
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// squadPlacementMaxRadius はステージ遷移で隊員を再配置するときの空きタイル探索の最大半径。
// 街や遺跡入口の密集地でも近くの空きを拾えるだけの広さを確保する。
const squadPlacementMaxRadius = 6

// MoveToBackpack はエンティティをバックパックに移動する。
// Stackableアイテムの場合、バックパック内の同名アイテムと自動的に統合する
func MoveToBackpack(world w.World, entity ecs.Entity, owner ecs.Entity) error {
	clearLocation(world, entity)
	world.Components.LocationInBackpack.Add(entity, &gc.LocationInBackpack{Owner: owner})
	ensureRemoved(world.Components.GridElement, entity)
	ensureMarker(world, world.Components.StatsChanged, owner, &gc.StatsChanged{})
	ensureMarker(world, world.Components.WeightDirty, owner, &gc.WeightDirty{})

	if world.Components.Stackable.Has(entity) {
		name := world.Components.Name.Get(entity)
		if err := mergeStackableItems(world, name.Name, mergeInBackpack, owner); err != nil {
			return fmt.Errorf("バックパック内のアイテム統合に失敗: %w", err)
		}
	}
	return nil
}

// MoveToEquip はエンティティを指定スロットに装備する
func MoveToEquip(world w.World, entity ecs.Entity, owner ecs.Entity, slot gc.EquipmentSlotNumber) {
	clearLocation(world, entity)
	world.Components.LocationEquipped.Add(entity, &gc.LocationEquipped{
		Owner:         owner,
		EquipmentSlot: slot,
	})
	ensureRemoved(world.Components.GridElement, entity)
	ensureMarker(world, world.Components.StatsChanged, owner, &gc.StatsChanged{})
	ensureMarker(world, world.Components.WeightDirty, owner, &gc.WeightDirty{})
}

// MoveToField はエンティティをフィールドに移動する。
// previousOwnerは移動元の所有者で、nilでなければWeightDirtyマーカーを付与する。
// 新規生成時など前のOwnerが存在しない場合はnilを渡す
func MoveToField(world w.World, entity ecs.Entity, previousOwner *ecs.Entity) {
	clearLocation(world, entity)
	world.Components.LocationOnField.Add(entity, &gc.LocationOnField{})
	// フィールド配置ではGridElement（座標）を残す
	if previousOwner != nil {
		ensureMarker(world, world.Components.WeightDirty, *previousOwner, &gc.WeightDirty{})
	}
}

// MoveToStorage はエンティティを収納に移動する。
// Stackableアイテムの場合、収納内の同名アイテムと自動的に統合する
func MoveToStorage(world w.World, entity ecs.Entity, storage ecs.Entity) error {
	clearLocation(world, entity)
	world.Components.LocationInStorage.Add(entity, &gc.LocationInStorage{Owner: storage})
	ensureRemoved(world.Components.GridElement, entity)
	ensureMarker(world, world.Components.WeightDirty, storage, &gc.WeightDirty{})

	if world.Components.Stackable.Has(entity) {
		name := world.Components.Name.Get(entity)
		if err := mergeStackableItems(world, name.Name, mergeInStorage, storage); err != nil {
			return fmt.Errorf("収納内のアイテム統合に失敗: %w", err)
		}
	}
	return nil
}

// UnequipAll はプレイヤーの装備中アイテムを全てバックパックに移動する
func UnequipAll(world w.World, playerEntity ecs.Entity) error {
	var equipped []ecs.Entity
	equippedQuery := ecs.NewFilter1[gc.LocationEquipped](world.ECS).Query()
	for equippedQuery.Next() {
		entity := equippedQuery.Entity()
		loc := world.Components.LocationEquipped.Get(entity)
		if loc.Owner == playerEntity {
			equipped = append(equipped, entity)
		}
	}

	for _, item := range equipped {
		if err := MoveToBackpack(world, item, playerEntity); err != nil {
			return err
		}
	}
	return nil
}

// ensureMarker はマーカーコンポーネントを冪等に付与する。
// エンティティが死亡している場合や既に付与済みの場合は何もしない。
// Arkは死亡エンティティへの付与と二重付与でパニックするため、ここで吸収する
func ensureMarker[T any](world w.World, comp *ecs.Map[T], entity ecs.Entity, data *T) {
	if !world.ECS.Alive(entity) {
		return
	}
	if !comp.Has(entity) {
		comp.Add(entity, data)
	}
}

// ensureRemoved はコンポーネントを保持している場合のみ取り除く。
// Arkは不在コンポーネントのRemoveでパニックするため、ここで吸収する
func ensureRemoved[T any](comp *ecs.Map[T], entity ecs.Entity) {
	if comp.Has(entity) {
		comp.Remove(entity)
	}
}

// clearLocation はエンティティの既存の位置コンポーネントをすべて取り除く。
// 排他制御のため、新しい位置を設定する前に呼ぶ（内部用）。
// 移動元にOwnerがある場合はそのOwnerに WeightDirty マーカーを付与する
func clearLocation(world w.World, entity ecs.Entity) {
	if world.Components.LocationInBackpack.Has(entity) {
		owner := world.Components.LocationInBackpack.Get(entity).Owner
		ensureMarker(world, world.Components.WeightDirty, owner, &gc.WeightDirty{})
	}
	if world.Components.LocationEquipped.Has(entity) {
		owner := world.Components.LocationEquipped.Get(entity).Owner
		ensureMarker(world, world.Components.WeightDirty, owner, &gc.WeightDirty{})
	}
	if world.Components.LocationInStorage.Has(entity) {
		owner := world.Components.LocationInStorage.Get(entity).Owner
		ensureMarker(world, world.Components.WeightDirty, owner, &gc.WeightDirty{})
	}

	ensureRemoved(world.Components.LocationInBackpack, entity)
	ensureRemoved(world.Components.LocationEquipped, entity)
	ensureRemoved(world.Components.LocationOnField, entity)
	ensureRemoved(world.Components.LocationInStorage, entity)
}

type mergeLocation int

const (
	mergeInBackpack mergeLocation = iota
	mergeInStorage
)

// mergeStackableItems は指定ロケーション内の同一Owner配下にある同名Stackableアイテムを1つに統合する
func mergeStackableItems(world w.World, itemName string, loc mergeLocation, owner ecs.Entity) error {
	// Ark のフィルタは静的な型引数を要求するため、ロケーション種別ごとに分岐する
	var stackableItems []ecs.Entity
	switch loc {
	case mergeInBackpack:
		q := ecs.NewFilter3[gc.Stackable, gc.LocationInBackpack, gc.Name](world.ECS).Query()
		for q.Next() {
			entity := q.Entity()
			if world.Components.Name.Get(entity).Name != itemName {
				continue
			}
			if world.Components.LocationInBackpack.Get(entity).Owner == owner {
				stackableItems = append(stackableItems, entity)
			}
		}
	case mergeInStorage:
		q := ecs.NewFilter3[gc.Stackable, gc.LocationInStorage, gc.Name](world.ECS).Query()
		for q.Next() {
			entity := q.Entity()
			if world.Components.Name.Get(entity).Name != itemName {
				continue
			}
			if world.Components.LocationInStorage.Get(entity).Owner == owner {
				stackableItems = append(stackableItems, entity)
			}
		}
	default:
		return fmt.Errorf("未対応のmergeLocation: %d", loc)
	}

	if len(stackableItems) <= 1 {
		return nil
	}

	targetEntity := stackableItems[0]
	for i := 1; i < len(stackableItems); i++ {
		itemToMerge := stackableItems[i]
		mergeCount := query.GetEntityCount(world, itemToMerge)

		if err := ChangeItemCount(world, targetEntity, mergeCount); err != nil {
			return fmt.Errorf("数量統合エラー: %w", err)
		}

		world.ECS.RemoveEntity(itemToMerge)
	}

	return nil
}

// tileAvailable はtileが配置可能かを返す。範囲外・進入不可・キャラ占有・除外指定なら不可。
// SpatialIndex が未構築(nil)のときは範囲と衝突を判定できないため、除外指定だけで可否を決める。
func tileAvailable(si *gc.SpatialIndex, tile consts.Coord[consts.Tile], exclude map[gc.GridElement]bool) bool {
	if tile.X < 0 || tile.Y < 0 {
		return false
	}
	if si != nil {
		if tile.X >= si.MapWidth || tile.Y >= si.MapHeight {
			return false
		}
		if si.IsBlockPass(tile) {
			return false
		}
		if _, occupied := si.CharacterAt(tile); occupied {
			return false
		}
	}
	return !exclude[gc.GridElement{Coord: tile}]
}

// findNearbyEmptyTile はcenterから近い順に空きタイルを探す。隣接(半径1)から外側へリングを
// 広げ、maxRadius まで探す。密集地でも遠くの空きを拾えるようにするための拡張探索。
// 見つからなければ ok=false を返す。呼び出し側が最終手段の退避先を決める。
func findNearbyEmptyTile(world w.World, center consts.Coord[consts.Tile], exclude map[gc.GridElement]bool, maxRadius int) (consts.Coord[consts.Tile], bool) {
	si := query.GetSpatialIndex(world)
	for r := 1; r <= maxRadius; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				// リングの外周だけを見る。内側の半径は前の反復で探索済み
				if dx > -r && dx < r && dy > -r && dy < r {
					continue
				}
				tile := center.Add(consts.Coord[consts.Tile]{X: consts.Tile(dx), Y: consts.Tile(dy)})
				if tileAvailable(si, tile, exclude) {
					return tile, true
				}
			}
		}
	}
	return consts.Coord[consts.Tile]{}, false
}

// MovePlayerToPosition は既存のプレイヤーエンティティを指定位置に移動させる
func MovePlayerToPosition(world w.World, pos consts.Coord[consts.Tile]) error {
	var playerEntity ecs.Entity
	var found bool

	playerQuery := ecs.NewFilter4[gc.Player, gc.GridElement, gc.SpriteRender, gc.Camera](world.ECS).Query()
	for playerQuery.Next() {
		entity := playerQuery.Entity()
		if !found {
			playerEntity = entity
			found = true
		}
	}
	if !found {
		return errors.New("必須コンポーネントを持つプレイヤーエンティティが見つかりません")
	}

	// プレイヤーの位置を更新する
	gridElement := world.Components.GridElement.Get(playerEntity)
	gridElement.X = pos.X
	gridElement.Y = pos.Y

	// カメラ位置も同期する
	camera := world.Components.Camera.Get(playerEntity)
	camera.Pos = consts.TileCenterToWorld(pos)
	camera.Target = camera.Pos

	// Active隊員をプレイヤーの近くに配置する。街や遺跡入口など密集地へ戻ると隣接が
	// 埋まっていることがあるため、近い順に外側へ広げて空きを探す。それでも見つからなければ
	// プレイヤーと同じタイルへ退避させる。隊員配置の失敗で遷移全体を止めるとダンジョンから
	// 戻れず詰むため、ここは絶対に失敗させない。重なっても次の移動で追従処理が空きへ散らす。
	exclude := map[gc.GridElement]bool{}
	for _, member := range query.SquadMembers(world) {
		memberGrid := world.Components.GridElement.Get(member)
		dest, ok := findNearbyEmptyTile(world, pos, exclude, squadPlacementMaxRadius)
		if !ok {
			dest = pos
		}
		memberGrid.X = dest.X
		memberGrid.Y = dest.Y
		exclude[gc.GridElement{Coord: dest}] = true
	}

	return nil
}
