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
	setLocation(world, entity, &gc.LocationInBackpack{Owner: owner})
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
	setLocation(world, entity, &gc.LocationEquipped{
		Owner:         owner,
		EquipmentSlot: slot,
	})
	ensureMarker(world, world.Components.StatsChanged, owner, &gc.StatsChanged{})
	ensureMarker(world, world.Components.WeightDirty, owner, &gc.WeightDirty{})
}

// MoveToField はエンティティをフィールドに移動する。
// previousOwnerは移動元の所有者で、nilでなければWeightDirtyマーカーを付与する。
// 新規生成時など前のOwnerが存在しない場合はnilを渡す
func MoveToField(world w.World, entity ecs.Entity, previousOwner *ecs.Entity) {
	setLocation(world, entity, &gc.LocationOnField{})
	if previousOwner != nil {
		ensureMarker(world, world.Components.WeightDirty, *previousOwner, &gc.WeightDirty{})
	}
}

// MoveToStorage はエンティティを収納に移動する。
// Stackableアイテムの場合、収納内の同名アイテムと自動的に統合する
func MoveToStorage(world w.World, entity ecs.Entity, storage ecs.Entity) error {
	setLocation(world, entity, &gc.LocationInStorage{Owner: storage})
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
	equippedQuery := ecs.NewFilter1[gc.LocationEquipped](world.World).Query()
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
	if !world.World.Alive(entity) {
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

// setLocation はエンティティの位置を設定する。排他制御を保証する。
// 既存の位置コンポーネントをすべて削除してから、新しい位置を設定する。
// 内部用関数なので直接呼び出さず、MoveToBackpack, MoveToField等を使用すること
func setLocation(world w.World, entity ecs.Entity, data gc.Location) {
	// 移動元のOwnerにWeightDirtyマーカーを付与する
	if world.Components.LocationInBackpack.Has(entity) {
		loc := world.Components.LocationInBackpack.Get(entity)
		ensureMarker(world, world.Components.WeightDirty, loc.Owner, &gc.WeightDirty{})
	}
	if world.Components.LocationEquipped.Has(entity) {
		loc := world.Components.LocationEquipped.Get(entity)
		ensureMarker(world, world.Components.WeightDirty, loc.Owner, &gc.WeightDirty{})
	}
	if world.Components.LocationInStorage.Has(entity) {
		loc := world.Components.LocationInStorage.Get(entity)
		ensureMarker(world, world.Components.WeightDirty, loc.Owner, &gc.WeightDirty{})
	}

	// すべての位置コンポーネントを削除（排他制御）
	ensureRemoved(world.Components.LocationInBackpack, entity)
	ensureRemoved(world.Components.LocationEquipped, entity)
	ensureRemoved(world.Components.LocationOnField, entity)
	ensureRemoved(world.Components.LocationInStorage, entity)

	// dataの型に応じて位置コンポーネントを追加
	switch v := data.(type) {
	case *gc.LocationInBackpack:
		world.Components.LocationInBackpack.Add(entity, v)
	case *gc.LocationEquipped:
		world.Components.LocationEquipped.Add(entity, v)
	case *gc.LocationOnField:
		world.Components.LocationOnField.Add(entity, v)
	case *gc.LocationInStorage:
		world.Components.LocationInStorage.Add(entity, v)
	}

	// フィールド以外に移動する場合はグリッド座標を除去する
	if _, ok := data.(*gc.LocationOnField); !ok {
		ensureRemoved(world.Components.GridElement, entity)
	}
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
		q := ecs.NewFilter3[gc.Stackable, gc.LocationInBackpack, gc.Name](world.World).Query()
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
		q := ecs.NewFilter3[gc.Stackable, gc.LocationInStorage, gc.Name](world.World).Query()
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

		world.World.RemoveEntity(itemToMerge)
	}

	return nil
}

// findAdjacentEmptyTile はcenterの隣接タイルから空きタイルを探す。
// excludeは追加で除外する座標セット。空きがなければエラーを返す
func findAdjacentEmptyTile(world w.World, centerX, centerY int, exclude map[gc.GridElement]bool) (int, int, error) {
	si := query.GetSpatialIndex(world)
	// 上下左右を優先し、次に斜めを探す
	offsets := [][2]int{
		{0, -1}, {0, 1}, {-1, 0}, {1, 0},
		{-1, -1}, {1, -1}, {-1, 1}, {1, 1},
	}
	for _, off := range offsets {
		x, y := centerX+off[0], centerY+off[1]
		if x < 0 || y < 0 {
			continue
		}
		// SpatialIndexが構築済みの場合のみ範囲と衝突をチェックする
		if si != nil {
			if x >= si.MapWidth || y >= si.MapHeight {
				continue
			}
			if si.IsBlockPass(x, y) {
				continue
			}
			if _, occupied := si.CharacterAt(x, y); occupied {
				continue
			}
		}
		pos := gc.GridElement{X: consts.Tile(x), Y: consts.Tile(y)}
		if exclude[pos] {
			continue
		}
		return x, y, nil
	}
	return 0, 0, fmt.Errorf("(%d,%d)の隣接に空きタイルがありません", centerX, centerY)
}

// MovePlayerToPosition は既存のプレイヤーエンティティを指定位置に移動させる
func MovePlayerToPosition(world w.World, tileX int, tileY int) error {
	var playerEntity ecs.Entity
	var found bool

	playerQuery := ecs.NewFilter4[gc.Player, gc.GridElement, gc.SpriteRender, gc.Camera](world.World).Query()
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
	gridElement.X = consts.Tile(tileX)
	gridElement.Y = consts.Tile(tileY)

	// カメラ位置も同期する
	camera := world.Components.Camera.Get(playerEntity)
	tileSize := float64(consts.TileSize)
	camera.X = float64(tileX)*tileSize + tileSize/2
	camera.Y = float64(tileY)*tileSize + tileSize/2
	camera.TargetX = camera.X
	camera.TargetY = camera.Y

	// Active隊員をプレイヤーの隣接タイルに配置する
	exclude := map[gc.GridElement]bool{}
	for _, member := range query.SquadMembers(world) {
		memberGrid := world.Components.GridElement.Get(member)
		x, y, err := findAdjacentEmptyTile(world, tileX, tileY, exclude)
		if err != nil {
			return fmt.Errorf("隊員の配置に失敗: %w", err)
		}
		memberGrid.X = consts.Tile(x)
		memberGrid.Y = consts.Tile(y)
		exclude[gc.GridElement{X: consts.Tile(x), Y: consts.Tile(y)}] = true
	}

	return nil
}
