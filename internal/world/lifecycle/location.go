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
		id := world.Components.RawID.Get(entity).ID
		if err := mergeStackableItems(world, id, mergeInBackpack, owner); err != nil {
			return fmt.Errorf("failed to merge items in backpack: %w", err)
		}
	}
	return nil
}

// TransferUnits は item のうち count 個だけ recipient のバックパックへ移す。
// count が0以下、または在庫数以上なら item を丸ごと移す。在庫数より少なければ、
// 元スタックを count 個減らし、同名の count 個を生成して recipient のバックパックへ統合する。
func TransferUnits(world w.World, item ecs.Entity, recipient ecs.Entity, count int) error {
	available := query.GetEntityCount(world, item)
	if count <= 0 || count >= available {
		return MoveToBackpack(world, item, recipient)
	}

	id := world.Components.RawID.Get(item).ID
	if err := ChangeItemCount(world, item, -count); err != nil {
		return fmt.Errorf("failed to decrement source stack: %w", err)
	}
	moved, err := spawnItemBase(world, id, count)
	if err != nil {
		return fmt.Errorf("failed to generate %d units to transfer: %w", count, err)
	}
	return MoveToBackpack(world, moved, recipient)
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
		// 所有者からフィールドへ移す実行時の移送は現ステージに属す。すぐ現ステージへ束縛し、
		// 置いた物が総重量など現ステージのクエリに即座に乗るようにする。次の swap を待つ遅延束縛
		// では、内部で置いた物が退場するまで総重量に現れない。
		// 生成時のフィールド生成は previousOwner が nil で来るのでここを通らず、生成後の
		// stage.Bind に束縛を委ねる。生成中は CurrentStage がまだ旧ステージなので誤束縛を避ける。
		if d := query.GetDungeon(world); d != nil {
			// entity は直前に LocationOnField を付けたので生存が保証され、Upsert は失敗しない
			_ = gc.Upsert(world.ECS, world.Components.StageBound, entity, &gc.StageBound{Key: d.CurrentStage})
		}
	}
}

// SpillStorageItems は収納の中身をすべて指定タイルのフィールドへ落とす。
// 収納エンティティを取り壊す前に呼び、中身が孤児化するのを防ぐ
func SpillStorageItems(world w.World, storage ecs.Entity, x consts.Tile, y consts.Tile) {
	// クエリ走査中の構造変更を避けるため、先に集めてから移動する
	var items []ecs.Entity
	q := ecs.NewFilter1[gc.LocationInStorage](world.ECS).Query()
	for q.Next() {
		entity := q.Entity()
		if world.Components.LocationInStorage.Get(entity).Owner == storage {
			items = append(items, entity)
		}
	}
	for _, item := range items {
		world.Components.GridElement.Add(item, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}})
		MoveToField(world, item, &storage)
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
		id := world.Components.RawID.Get(entity).ID
		if err := mergeStackableItems(world, id, mergeInStorage, storage); err != nil {
			return fmt.Errorf("failed to merge items in storage: %w", err)
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

// mergeStackableItems は指定ロケーション内の同一Owner配下にある同一idのStackableアイテムを1つに統合する
func mergeStackableItems(world w.World, itemID string, loc mergeLocation, owner ecs.Entity) error {
	// Ark のフィルタは静的な型引数を要求するため、ロケーション種別ごとに分岐する
	var stackableItems []ecs.Entity
	switch loc {
	case mergeInBackpack:
		q := ecs.NewFilter3[gc.Stackable, gc.LocationInBackpack, gc.RawID](world.ECS).Query()
		for q.Next() {
			entity := q.Entity()
			if world.Components.RawID.Get(entity).ID != itemID {
				continue
			}
			if world.Components.LocationInBackpack.Get(entity).Owner == owner {
				stackableItems = append(stackableItems, entity)
			}
		}
	case mergeInStorage:
		q := ecs.NewFilter3[gc.Stackable, gc.LocationInStorage, gc.RawID](world.ECS).Query()
		for q.Next() {
			entity := q.Entity()
			if world.Components.RawID.Get(entity).ID != itemID {
				continue
			}
			if world.Components.LocationInStorage.Get(entity).Owner == owner {
				stackableItems = append(stackableItems, entity)
			}
		}
	default:
		return fmt.Errorf("unsupported mergeLocation: %d", loc)
	}

	if len(stackableItems) <= 1 {
		return nil
	}

	targetEntity := stackableItems[0]
	for i := 1; i < len(stackableItems); i++ {
		itemToMerge := stackableItems[i]
		mergeCount := query.GetEntityCount(world, itemToMerge)

		if err := ChangeItemCount(world, targetEntity, mergeCount); err != nil {
			return fmt.Errorf("failed to merge counts: %w", err)
		}

		world.ECS.RemoveEntity(itemToMerge)
	}

	return nil
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
		return errors.New("no player entity with required components found")
	}

	// プレイヤーの位置を更新する
	gridElement := world.Components.GridElement.Get(playerEntity)
	gridElement.X = pos.X
	gridElement.Y = pos.Y

	// カメラ位置も同期する
	camera := world.Components.Camera.Get(playerEntity)
	camera.Pos = consts.TileCenterToWorld(pos)
	camera.Target = camera.Pos

	return nil
}
