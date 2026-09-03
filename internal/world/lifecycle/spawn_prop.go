package lifecycle

import (
	"fmt"

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
		return fmt.Errorf("entity is not a door")
	}

	doorComp := world.Components.Door.Get(doorEntity)
	return updateDoorState(world, doorEntity, doorComp.Orientation, true)
}

// CloseDoor は扉を閉じる
func CloseDoor(world w.World, doorEntity ecs.Entity) error {
	if !world.Components.Door.Has(doorEntity) {
		return fmt.Errorf("entity is not a door")
	}

	doorComp := world.Components.Door.Get(doorEntity)
	return updateDoorState(world, doorEntity, doorComp.Orientation, false)
}

// updateDoorState は扉の向きと開閉状態に応じて、状態を更新する
func updateDoorState(world w.World, doorEntity ecs.Entity, orientation gc.DoorOrientation, isOpen bool) error {
	doorComp := world.Components.Door.Get(doorEntity)
	doorComp.Orientation = orientation
	doorComp.IsOpen = isOpen

	// スプライトキーと高さを更新する。
	// 開いた扉は平らな低い物として扱い、高さを下げる。高さが無いのでドロップシャドウを落とさず、
	// 背の高い物の後ろに沈む。閉じた扉は高さのある障壁に戻す。
	if world.Components.SpriteRender.Has(doorEntity) {
		spriteRender := world.Components.SpriteRender.Get(doorEntity)

		if isOpen {
			spriteRender.Depth = gc.DepthNumRug
			if orientation == gc.DoorOrientationHorizontal {
				spriteRender.SpriteKey = "door_horizontal_open"
			} else {
				spriteRender.SpriteKey = "door_vertical_open"
			}
		} else {
			spriteRender.Depth = gc.DepthNumTaller
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

	// 通行可否が変わったので周囲の囲われを焼き直す。扉を開けると外気が入り、閉じると屋内に戻る
	if world.Components.GridElement.Has(doorEntity) {
		coord := world.Components.GridElement.Get(doorEntity).Coord
		RecalcShelterAround(world, coord.X, coord.Y)
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
	entitySpec.Name = &gc.Name{Name: query.T(world, "Ruins Entrance")}
	entitySpec.Description = &gc.Description{Description: query.T(world, "An entrance leading to ruins")}
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
func SpawnDoor(world w.World, pos consts.Coord[consts.Tile], orientation gc.DoorOrientation) (ecs.Entity, error) {
	var spriteKey string
	if orientation == gc.DoorOrientationHorizontal {
		spriteKey = "door_horizontal_closed"
	} else {
		spriteKey = "door_vertical_closed"
	}

	return world.Components.AddEntity(world.ECS, &gc.EntitySpec{
		Name:        &gc.Name{Name: query.T(world, "Door")},
		Description: &gc.Description{Description: query.T(world, "An openable door")},
		GridElement: &gc.GridElement{Coord: pos},
		SpriteRender: &gc.SpriteRender{
			SpriteSheetName: fieldSpriteSheet,
			SpriteKey:       spriteKey,
			Depth:           gc.DepthNumTaller,
		},
		Fixed:           &gc.Fixed{},
		BlockPass:       &gc.BlockPass{},
		BlockView:       &gc.BlockView{},
		LocationOnField: &gc.LocationOnField{},
		Door: &gc.Door{
			IsOpen:      false,
			Orientation: orientation,
		},
		Interactable: &gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionDoor}},
	}), nil
}

// SpawnCube は押して動かせる移動拠点キューブをオーバーワールドに生成する。
// blue_cube のスプライトを流用した無地のキューブ。BlockPass だが Pushable なので
// 歩き込むと通行でなく押しになる。オーバーワールドの地物として帯へ明示束縛する。
func SpawnCube(world w.World, pos consts.Coord[consts.Tile]) (ecs.Entity, error) {
	return world.Components.AddEntity(world.ECS, &gc.EntitySpec{
		Name:        &gc.Name{Name: query.T(world, "Cube")},
		Description: &gc.Description{Description: query.T(world, "A mobile base you can push")},
		GridElement: &gc.GridElement{Coord: pos},
		SpriteRender: &gc.SpriteRender{
			SpriteSheetName: fieldSpriteSheet,
			SpriteKey:       "blue_cube",
			Depth:           gc.DepthNumTaller,
		},
		Fixed:           &gc.Fixed{},
		BlockPass:       &gc.BlockPass{},
		Pushable:        &gc.Pushable{},
		LocationOnField: &gc.LocationOnField{},
		StageBound:      &gc.StageBound{Key: gc.NewOverworldStage()},
		// 隣接して手動で内部へ入る、または引く。歩き込みは押し、明示的な入る/引くはメニューから
		Interactable: &gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionEnterCube, gc.InteractionPullCube}},
	}), nil
}
