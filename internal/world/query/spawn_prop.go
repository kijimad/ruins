package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// OpenDoor は扉を開く
func OpenDoor(world w.World, doorEntity ecs.Entity) error {
	if !doorEntity.HasComponent(world.Components.Door) {
		return fmt.Errorf("エンティティは扉ではありません")
	}

	doorComp := world.Components.Door.Get(doorEntity).(*gc.Door)
	return updateDoorState(world, doorEntity, doorComp.Orientation, true)
}

// CloseDoor は扉を閉じる
func CloseDoor(world w.World, doorEntity ecs.Entity) error {
	if !doorEntity.HasComponent(world.Components.Door) {
		return fmt.Errorf("エンティティは扉ではありません")
	}

	doorComp := world.Components.Door.Get(doorEntity).(*gc.Door)
	return updateDoorState(world, doorEntity, doorComp.Orientation, false)
}

// LockAllDoors は全扉を閉じてロックする。ロックされた扉の数を返す
func LockAllDoors(world w.World) int {
	locked := 0
	world.Manager.Join(world.Components.Door).Visit(ecs.Visit(func(doorEntity ecs.Entity) {
		doorComp := world.Components.Door.Get(doorEntity).(*gc.Door)
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
		doorComp := world.Components.Door.Get(doorEntity).(*gc.Door)
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
	doorComp := world.Components.Door.Get(doorEntity).(*gc.Door)
	doorComp.Orientation = orientation
	doorComp.IsOpen = isOpen

	// スプライトキーを更新
	if doorEntity.HasComponent(world.Components.SpriteRender) {
		spriteRender := world.Components.SpriteRender.Get(doorEntity).(*gc.SpriteRender)

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
		if doorEntity.HasComponent(world.Components.BlockPass) {
			doorEntity.RemoveComponent(world.Components.BlockPass)
		}
		if doorEntity.HasComponent(world.Components.BlockView) {
			doorEntity.RemoveComponent(world.Components.BlockView)
		}
	} else {
		if !doorEntity.HasComponent(world.Components.BlockPass) {
			doorEntity.AddComponent(world.Components.BlockPass, &gc.BlockPass{})
		}
		if !doorEntity.HasComponent(world.Components.BlockView) {
			doorEntity.AddComponent(world.Components.BlockView, &gc.BlockView{})
		}
	}

	return nil
}
