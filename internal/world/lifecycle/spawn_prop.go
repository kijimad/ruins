package lifecycle

import (
	"fmt"
	"slices"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// OpenDoor は扉を開く
func OpenDoor(world w.World, doorEntity ecs.Entity) error {
	if !world.Components.Door.Has(doorEntity) {
		return fmt.Errorf("エンティティは扉ではありません")
	}

	doorComp := world.Components.Door.Get(doorEntity)
	return updateDoorState(world, doorEntity, doorComp.Orientation, true)
}

// CloseDoor は扉を閉じる
func CloseDoor(world w.World, doorEntity ecs.Entity) error {
	if !world.Components.Door.Has(doorEntity) {
		return fmt.Errorf("エンティティは扉ではありません")
	}

	doorComp := world.Components.Door.Get(doorEntity)
	return updateDoorState(world, doorEntity, doorComp.Orientation, false)
}

// LockAllDoors は全扉を閉じてロックする。ロックされた扉の数を返す
func LockAllDoors(world w.World) int {
	var doors []ecs.Entity
	// 退避中ステージの扉は操作しない。現ステージのみ対象にする
	doorQuery := query.ActiveFilter1[gc.Door](world).Query()
	for doorQuery.Next() {
		doors = append(doors, doorQuery.Entity())
	}
	locked := 0
	for _, doorEntity := range doors {
		if world.Components.Door.Get(doorEntity).Locked {
			continue
		}
		if world.Components.Door.Get(doorEntity).IsOpen {
			_ = CloseDoor(world, doorEntity)
		}
		// CloseDoorがarchetypeを変えGetポインタを失効させるため、取り直してから書き込む
		world.Components.Door.Get(doorEntity).Locked = true
		locked++
	}
	if locked > 0 {
		// BlockView が変化したので視界を再計算させる
		query.GetVisionState(world).NeedsForceUpdate = true
	}
	return locked
}

// UnlockAllDoors は全扉をアンロックして開く。開かれた扉の数を返す
func UnlockAllDoors(world w.World) int {
	var doors []ecs.Entity
	// 退避中ステージの扉は操作しない。現ステージのみ対象にする
	doorQuery := query.ActiveFilter1[gc.Door](world).Query()
	for doorQuery.Next() {
		doors = append(doors, doorQuery.Entity())
	}
	opened := 0
	for _, doorEntity := range doors {
		doorComp := world.Components.Door.Get(doorEntity)
		doorComp.Locked = false
		if !doorComp.IsOpen {
			_ = OpenDoor(world, doorEntity)
			opened++
		}
	}
	if opened > 0 {
		// BlockView が変化したので視界を再計算させる
		query.GetVisionState(world).NeedsForceUpdate = true
	}
	return opened
}

// updateDoorState は扉の向きと開閉状態に応じて、状態を更新する
func updateDoorState(world w.World, doorEntity ecs.Entity, orientation gc.DoorOrientation, isOpen bool) error {
	doorComp := world.Components.Door.Get(doorEntity)
	doorComp.Orientation = orientation
	doorComp.IsOpen = isOpen

	// スプライトキーを更新
	if world.Components.SpriteRender.Has(doorEntity) {
		spriteRender := world.Components.SpriteRender.Get(doorEntity)

		if isOpen {
			if orientation == gc.DoorOrientationHorizontal {
				spriteRender.SpriteKey = "door_horizontal_open"
			} else {
				spriteRender.SpriteKey = "door_vertical_open"
			}
		} else {
			if orientation == gc.DoorOrientationHorizontal {
				spriteRender.SpriteKey = "door_horizontal_closed"
			} else {
				spriteRender.SpriteKey = "door_vertical_closed"
			}
		}
	}

	// BlockPass / BlockView を更新
	if isOpen {
		if world.Components.BlockPass.Has(doorEntity) {
			world.Components.BlockPass.Remove(doorEntity)
		}
		if world.Components.BlockView.Has(doorEntity) {
			world.Components.BlockView.Remove(doorEntity)
		}
	} else {
		if !world.Components.BlockPass.Has(doorEntity) {
			world.Components.BlockPass.Add(doorEntity, &gc.BlockPass{})
		}
		if !world.Components.BlockView.Has(doorEntity) {
			world.Components.BlockView.Add(doorEntity, &gc.BlockView{})
		}
	}

	return nil
}

// SpawnProp は置物を生成する
func SpawnProp(world w.World, propName string, x consts.Tile, y consts.Tile) (ecs.Entity, error) {
	entitySpec, err := raw.NewPropSpec(world.Resources.RawMaster, propName)
	if err != nil {
		return gc.InvalidEntity, err
	}

	entitySpec.GridElement = &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}}
	entitySpec.LocationOnField = &gc.LocationOnField{}

	return world.Components.AddEntity(world.ECS, &entitySpec), nil
}

// SpawnDungeonEntrance は遺跡入口プロップを生成する。触れて Enter で definitionName の遺跡へ入る。
// オーバーワールドはコードで入口を配置するため、raw でなく EntitySpec を直接組む。
// 入口はオーバーワールドの地物なので StageBound{overworld} を直接持たせ、遺跡進入時に帯と共に
// 退避されるようにする。swapTo の遅延 Bind に頼らず、明示束縛でリファクタリング耐性を上げる。
func SpawnDungeonEntrance(world w.World, x consts.Tile, y consts.Tile, definitionName string) (ecs.Entity, error) {
	// ダンジョン内の階段ポータルと同じ raw プロップ warp_next を流用し、回転アニメを揃える。
	// warp_next は次階ポータル用なので、相互作用を遺跡進入へ差し替え、入口固有のコンポーネントを
	// 足す。オーバーワールドの地物として帯へ明示束縛し、遺跡進入時に帯と共に退避されるようにする。
	entitySpec, err := raw.NewPropSpec(world.Resources.RawMaster, "warp_next")
	if err != nil {
		return gc.InvalidEntity, err
	}
	entitySpec.Name = &gc.Name{Name: "遺跡入口"}
	entitySpec.Description = &gc.Description{Description: "遺跡へ通じる入口"}
	entitySpec.GridElement = &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}}
	entitySpec.LocationOnField = &gc.LocationOnField{}
	entitySpec.Interactable = &gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionDungeonEnter}}
	entitySpec.DungeonEntrance = &gc.DungeonEntrance{DefinitionName: definitionName}
	entitySpec.StageBound = &gc.StageBound{Key: gc.NewOverworldStage()}
	// warp_next は暗いダンジョン用に光源を持つが、明るいオーバーワールドでは効かないうえ
	// 入口に不要なので落とす。流用するのはスプライトとアニメフレームだけでよい。
	entitySpec.LightSource = nil

	return world.Components.AddEntity(world.ECS, &entitySpec), nil
}

// SpawnDoor は扉を生成する
func SpawnDoor(world w.World, x consts.Tile, y consts.Tile, orientation gc.DoorOrientation) (ecs.Entity, error) {
	var spriteKey string
	if orientation == gc.DoorOrientationHorizontal {
		spriteKey = "door_horizontal_closed"
	} else {
		spriteKey = "door_vertical_closed"
	}

	entitySpec := gc.EntitySpec{
		Name:        &gc.Name{Name: "扉"},
		Description: &gc.Description{Description: "開閉できる扉"},
		GridElement: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}},
		SpriteRender: &gc.SpriteRender{
			SpriteSheetName: fieldSpriteSheet,
			SpriteKey:       spriteKey,
			Depth:           gc.DepthNumTaller,
		},
		Prop:            &gc.Prop{},
		BlockPass:       &gc.BlockPass{},
		BlockView:       &gc.BlockView{},
		LocationOnField: &gc.LocationOnField{},
		Door: &gc.Door{
			IsOpen:      false,
			Orientation: orientation,
		},
		Interactable: &gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionDoor}},
	}

	return world.Components.AddEntity(world.ECS, &entitySpec), nil
}

// DeleteDoorLockTriggers はDoorLockInteractionを持つエンティティを全削除する
func DeleteDoorLockTriggers(world w.World) {
	var toDelete []ecs.Entity
	// 退避中ステージの鍵トリガは消さない。現ステージのみ対象にする
	interactableQuery := query.ActiveFilter1[gc.Interactable](world).Query()
	for interactableQuery.Next() {
		triggerEntity := interactableQuery.Entity()
		interactable := world.Components.Interactable.Get(triggerEntity)
		if slices.Contains(interactable.Interactions, gc.InteractionDoorLock) {
			toDelete = append(toDelete, triggerEntity)
		}
	}
	for _, entity := range toDelete {
		world.ECS.RemoveEntity(entity)
	}
}
