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

// findAdjacentEmptyTile はcenterの隣接タイルから空きタイルを探す。
// excludeは追加で除外する座標セット。空きがなければエラーを返す
func findAdjacentEmptyTile(world w.World, center consts.Coord[consts.Tile], exclude map[gc.GridElement]bool) (consts.Coord[consts.Tile], error) {
	si := query.GetSpatialIndex(world)
	// 上下左右を優先し、次に斜めを探す
	offsets := [][2]int{
		{0, -1}, {0, 1}, {-1, 0}, {1, 0},
		{-1, -1}, {1, -1}, {-1, 1}, {1, 1},
	}
	for _, off := range offsets {
		x, y := int(center.X)+off[0], int(center.Y)+off[1]
		if x < 0 || y < 0 {
			continue
		}
		tile := consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
		// SpatialIndexが構築済みの場合のみ範囲と衝突をチェックする
		if si != nil {
			if x >= si.MapWidth || y >= si.MapHeight {
				continue
			}
			if si.IsBlockPass(tile) {
				continue
			}
			if _, occupied := si.CharacterAt(tile); occupied {
				continue
			}
		}
		if exclude[gc.GridElement{Coord: tile}] {
			continue
		}
		return tile, nil
	}
	return consts.Coord[consts.Tile]{}, fmt.Errorf("(%d,%d)の隣接に空きタイルがありません", center.X, center.Y)
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

	// Active隊員をプレイヤーの隣接タイルに配置する
	exclude := map[gc.GridElement]bool{}
	for _, member := range query.SquadMembers(world) {
		memberGrid := world.Components.GridElement.Get(member)
		adj, err := findAdjacentEmptyTile(world, pos, exclude)
		if err != nil {
			return fmt.Errorf("隊員の配置に失敗: %w", err)
		}
		memberGrid.X = adj.X
		memberGrid.Y = adj.Y
		exclude[gc.GridElement{Coord: adj}] = true
	}

	return nil
}
