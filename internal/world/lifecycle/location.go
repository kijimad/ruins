package lifecycle

import (
	"errors"
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// MoveToBackpack はエンティティをバックパックに移動する。
// Stackableアイテムの場合、バックパック内の同名アイテムと自動的に統合する
func MoveToBackpack(world w.World, entity ecs.Entity, owner ecs.Entity) error {
	setLocation(world, entity, &gc.LocationInBackpack{Owner: owner})
	owner.AddComponent(world.Components.StatsChanged, &gc.StatsChanged{})
	owner.AddComponent(world.Components.WeightDirty, &gc.WeightDirty{})

	if entity.HasComponent(world.Components.Stackable) {
		name := world.Components.Name.Get(entity).(*gc.Name)
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
	owner.AddComponent(world.Components.StatsChanged, &gc.StatsChanged{})
	owner.AddComponent(world.Components.WeightDirty, &gc.WeightDirty{})
}

// MoveToField はエンティティをフィールドに移動する。
// previousOwnerは移動元の所有者で、nilでなければWeightDirtyマーカーを付与する。
// 新規生成時など前のOwnerが存在しない場合はnilを渡す
func MoveToField(world w.World, entity ecs.Entity, previousOwner *ecs.Entity) {
	setLocation(world, entity, &gc.LocationOnField{})
	if previousOwner != nil {
		previousOwner.AddComponent(world.Components.WeightDirty, &gc.WeightDirty{})
	}
}

// MoveToStorage はエンティティを収納に移動する。
// Stackableアイテムの場合、収納内の同名アイテムと自動的に統合する
func MoveToStorage(world w.World, entity ecs.Entity, storage ecs.Entity) error {
	setLocation(world, entity, &gc.LocationInStorage{Owner: storage})
	storage.AddComponent(world.Components.WeightDirty, &gc.WeightDirty{})

	if entity.HasComponent(world.Components.Stackable) {
		name := world.Components.Name.Get(entity).(*gc.Name)
		if err := mergeStackableItems(world, name.Name, mergeInStorage, storage); err != nil {
			return fmt.Errorf("収納内のアイテム統合に失敗: %w", err)
		}
	}
	return nil
}

// UnequipAll はプレイヤーの装備中アイテムを全てバックパックに移動する
func UnequipAll(world w.World, playerEntity ecs.Entity) error {
	var equipped []ecs.Entity
	world.Manager.Join(
		world.Components.LocationEquipped,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		loc := world.Components.LocationEquipped.Get(entity).(*gc.LocationEquipped)
		if loc.Owner == playerEntity {
			equipped = append(equipped, entity)
		}
	}))

	for _, item := range equipped {
		if err := MoveToBackpack(world, item, playerEntity); err != nil {
			return err
		}
	}
	return nil
}

// setLocation はエンティティの位置を設定する。排他制御を保証する。
// 既存の位置コンポーネントをすべて削除してから、新しい位置を設定する。
// 内部用関数なので直接呼び出さず、MoveToBackpack, MoveToField等を使用すること
func setLocation(world w.World, entity ecs.Entity, data gc.Location) {
	// 移動元のOwnerにWeightDirtyマーカーを付与する
	if entity.HasComponent(world.Components.LocationInBackpack) {
		loc := world.Components.LocationInBackpack.Get(entity).(*gc.LocationInBackpack)
		loc.Owner.AddComponent(world.Components.WeightDirty, &gc.WeightDirty{})
	}
	if entity.HasComponent(world.Components.LocationEquipped) {
		loc := world.Components.LocationEquipped.Get(entity).(*gc.LocationEquipped)
		loc.Owner.AddComponent(world.Components.WeightDirty, &gc.WeightDirty{})
	}
	if entity.HasComponent(world.Components.LocationInStorage) {
		loc := world.Components.LocationInStorage.Get(entity).(*gc.LocationInStorage)
		loc.Owner.AddComponent(world.Components.WeightDirty, &gc.WeightDirty{})
	}

	// すべての位置コンポーネントを削除（排他制御）
	entity.RemoveComponent(world.Components.LocationInBackpack)
	entity.RemoveComponent(world.Components.LocationEquipped)
	entity.RemoveComponent(world.Components.LocationOnField)
	entity.RemoveComponent(world.Components.LocationInStorage)

	// dataの型に応じて位置コンポーネントを追加
	switch v := data.(type) {
	case *gc.LocationInBackpack:
		entity.AddComponent(world.Components.LocationInBackpack, v)
	case *gc.LocationEquipped:
		entity.AddComponent(world.Components.LocationEquipped, v)
	case *gc.LocationOnField:
		entity.AddComponent(world.Components.LocationOnField, v)
	case *gc.LocationInStorage:
		entity.AddComponent(world.Components.LocationInStorage, v)
	}

	// フィールド以外に移動する場合はグリッド座標を除去する
	if _, ok := data.(*gc.LocationOnField); !ok {
		entity.RemoveComponent(world.Components.GridElement)
	}
}

type mergeLocation int

const (
	mergeInBackpack mergeLocation = iota
	mergeInStorage
)

// mergeStackableItems は指定ロケーション内の同一Owner配下にある同名Stackableアイテムを1つに統合する
func mergeStackableItems(world w.World, itemName string, loc mergeLocation, owner ecs.Entity) error {
	var locationComp ecs.DataComponent
	switch loc {
	case mergeInBackpack:
		locationComp = world.Components.LocationInBackpack
	case mergeInStorage:
		locationComp = world.Components.LocationInStorage
	default:
		return fmt.Errorf("未対応のmergeLocation: %d", loc)
	}

	var stackableItems []ecs.Entity
	world.Manager.Join(
		world.Components.Stackable,
		locationComp,
		world.Components.Name,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := world.Components.Name.Get(entity).(*gc.Name)
		if name.Name != itemName {
			return
		}
		switch l := locationComp.Get(entity).(type) {
		case *gc.LocationInBackpack:
			if l.Owner == owner {
				stackableItems = append(stackableItems, entity)
			}
		case *gc.LocationInStorage:
			if l.Owner == owner {
				stackableItems = append(stackableItems, entity)
			}
		}
	}))

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

		world.Manager.DeleteEntity(itemToMerge)
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

	world.Manager.Join(
		world.Components.Player,
		world.Components.GridElement,
		world.Components.SpriteRender,
		world.Components.Camera,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if !found {
			playerEntity = entity
			found = true
		}
	}))
	if !found {
		return errors.New("必須コンポーネントを持つプレイヤーエンティティが見つかりません")
	}

	// プレイヤーの位置を更新する
	gridElement := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)
	gridElement.X = consts.Tile(tileX)
	gridElement.Y = consts.Tile(tileY)

	// カメラ位置も同期する
	camera := world.Components.Camera.Get(playerEntity).(*gc.Camera)
	tileSize := float64(consts.TileSize)
	camera.X = float64(tileX)*tileSize + tileSize/2
	camera.Y = float64(tileY)*tileSize + tileSize/2
	camera.TargetX = camera.X
	camera.TargetY = camera.Y

	// Active隊員をプレイヤーの隣接タイルに配置する
	exclude := map[gc.GridElement]bool{}
	for _, member := range query.SquadMembers(world) {
		memberGrid := world.Components.GridElement.Get(member).(*gc.GridElement)
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
