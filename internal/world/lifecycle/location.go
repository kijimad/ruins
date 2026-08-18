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
// 1個1エンティティなので統合はしない。同一スタックの束ねは表示側が stackKey で行う。
func MoveToBackpack(world w.World, entity ecs.Entity, owner ecs.Entity) error {
	clearLocation(world, entity)
	world.Components.LocationInBackpack.Add(entity, &gc.LocationInBackpack{Owner: owner})
	ensureRemoved(world.Components.GridElement, entity)
	ensureMarker(world, world.Components.StatsChanged, owner, &gc.StatsChanged{})
	ensureMarker(world, world.Components.WeightDirty, owner, &gc.WeightDirty{})
	return nil
}

// TransferUnits は item が属するスタックのうち count 個を recipient のバックパックへ移す。
// count が0以下、または在庫数以上ならスタックを丸ごと移す。1個1エンティティなので、
// 同一スタックのエンティティを count 個選んで移すだけでよい。生成や分割は不要。
func TransferUnits(world w.World, item ecs.Entity, recipient ecs.Entity, count int) error {
	members := query.StackMembers(world, item)
	move := count
	if count <= 0 || count >= len(members) {
		move = len(members)
	}
	if _, err := MoveMembersToBackpack(world, members[:move], recipient); err != nil {
		return fmt.Errorf("failed to transfer unit: %w", err)
	}
	return nil
}

// MoveMembersToBackpack は members を全てバックパックへ移す。移せた個数を返す。
// スタックを丸ごと動かす操作はこの1関数を通し、一部だけ動かす取りこぼしを防ぐ。
// 対象は呼び出し側が query.StackMembers 等で束ねて渡す。移動先の詳細だけをここに集約する。
func MoveMembersToBackpack(world w.World, members []ecs.Entity, owner ecs.Entity) (int, error) {
	moved := 0
	var errs []error
	for _, member := range members {
		if err := MoveToBackpack(world, member, owner); err != nil {
			errs = append(errs, err)
			continue
		}
		moved++
	}
	return moved, errors.Join(errs...)
}

// MoveMembersToStorage は members を全て収納へ移す。移せた個数を返す。容量判定は呼び出し側が事前に行う。
func MoveMembersToStorage(world w.World, members []ecs.Entity, storage ecs.Entity) (int, error) {
	moved := 0
	var errs []error
	for _, member := range members {
		if err := MoveToStorage(world, member, storage); err != nil {
			errs = append(errs, err)
			continue
		}
		moved++
	}
	return moved, errors.Join(errs...)
}

// MoveMembersToField は members を全て指定タイルへ落とす。各個体に GridElement を付け、床の同一スタックへ束ねる。
// GridElement 付与を移動と対にしてここへ集約し、片方だけ忘れて床に出ないバグを防ぐ。移した個数を返す。
func MoveMembersToField(world w.World, members []ecs.Entity, coord consts.Coord[consts.Tile], previousOwner ecs.Entity) int {
	for _, member := range members {
		MoveToField(world, member, &previousOwner)
		world.Components.GridElement.Add(member, &gc.GridElement{Coord: coord})
	}
	return len(members)
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
// 1個1エンティティなので統合はしない。同一スタックの束ねは表示側が stackKey で行う。
func MoveToStorage(world w.World, entity ecs.Entity, storage ecs.Entity) error {
	clearLocation(world, entity)
	world.Components.LocationInStorage.Add(entity, &gc.LocationInStorage{Owner: storage})
	ensureRemoved(world.Components.GridElement, entity)
	ensureMarker(world, world.Components.WeightDirty, storage, &gc.WeightDirty{})
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
