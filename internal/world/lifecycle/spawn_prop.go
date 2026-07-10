package lifecycle

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/engine/entities"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
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
	locked := 0
	world.Manager.Join(world.Components.Door).Visit(ecs.Visit(func(doorEntity ecs.Entity) {
		doorComp := world.Components.Door.Get(doorEntity)
		if doorComp.Locked {
			return
		}
		if doorComp.IsOpen {
			_ = CloseDoor(world, doorEntity)
		}
		doorComp.Locked = true
		locked++
	}))
	return locked
}

// UnlockAllDoors は全扉をアンロックして開く。開かれた扉の数を返す
func UnlockAllDoors(world w.World) int {
	opened := 0
	world.Manager.Join(world.Components.Door).Visit(ecs.Visit(func(doorEntity ecs.Entity) {
		doorComp := world.Components.Door.Get(doorEntity)
		doorComp.Locked = false
		if !doorComp.IsOpen {
			_ = OpenDoor(world, doorEntity)
			opened++
		}
	}))
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
		return consts.InvalidEntity, err
	}

	entitySpec.GridElement = &gc.GridElement{X: x, Y: y}
	loc := gc.LocationTypeOnField
	entitySpec.LocationType = &loc

	componentList := entities.ComponentList[gc.EntitySpec]{}
	componentList.Entities = append(componentList.Entities, entitySpec)
	ents, err := entities.AddEntities(world, componentList)
	if err != nil {
		return consts.InvalidEntity, err
	}
	return ents[len(ents)-1], nil
}

// SpawnDoor は扉を生成する
func SpawnDoor(world w.World, x consts.Tile, y consts.Tile, orientation gc.DoorOrientation) (ecs.Entity, error) {
	var spriteKey string
	if orientation == gc.DoorOrientationHorizontal {
		spriteKey = "door_horizontal_closed"
	} else {
		spriteKey = "door_vertical_closed"
	}

	loc := gc.LocationTypeOnField
	entitySpec := gc.EntitySpec{
		Name:        &gc.Name{Name: "扉"},
		Description: &gc.Description{Description: "開閉できる扉"},
		GridElement: &gc.GridElement{X: x, Y: y},
		SpriteRender: &gc.SpriteRender{
			SpriteSheetName: "field",
			SpriteKey:       spriteKey,
			Depth:           gc.DepthNumTaller,
		},
		Prop:         &gc.Prop{},
		BlockPass:    &gc.BlockPass{},
		BlockView:    &gc.BlockView{},
		LocationType: &loc,
		Door: &gc.Door{
			IsOpen:      false,
			Orientation: orientation,
		},
		Interactable: &gc.Interactable{Interactions: []gc.InteractionData{gc.DoorInteraction{}}},
	}

	componentList := entities.ComponentList[gc.EntitySpec]{}
	componentList.Entities = append(componentList.Entities, entitySpec)
	ents, err := entities.AddEntities(world, componentList)
	if err != nil {
		return consts.InvalidEntity, err
	}
	if len(ents) == 0 {
		return consts.InvalidEntity, fmt.Errorf("エンティティが生成されませんでした")
	}
	return ents[len(ents)-1], nil
}

// DeleteDoorLockTriggers はDoorLockInteractionを持つエンティティを全削除する
func DeleteDoorLockTriggers(world w.World) {
	var toDelete []ecs.Entity
	world.Manager.Join(world.Components.Interactable).Visit(ecs.Visit(func(triggerEntity ecs.Entity) {
		interactable := world.Components.Interactable.Get(triggerEntity)
		for _, interaction := range interactable.Interactions {
			if _, ok := interaction.(gc.DoorLockInteraction); ok {
				toDelete = append(toDelete, triggerEntity)
				return
			}
		}
	}))
	for _, entity := range toDelete {
		world.World.RemoveEntity(entity)
	}
}
