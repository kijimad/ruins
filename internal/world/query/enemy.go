package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// GetVisibleEnemies は視界内の敵エンティティをすべて取得して返す
func GetVisibleEnemies(world w.World) ([]ecs.Entity, error) {
	playerEntity, err := GetPlayerEntity(world)
	if err != nil {
		return nil, err
	}

	if !playerEntity.HasComponent(world.Components.GridElement) {
		return nil, fmt.Errorf("プレイヤーがGridElementを持っていません")
	}

	playerGrid := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)
	playerX := int(playerGrid.X)
	playerY := int(playerGrid.Y)

	var enemies []ecs.Entity

	world.Manager.Join(
		world.Components.GridElement,
		world.Components.FactionEnemy,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
		enemyX := int(gridElement.X)
		enemyY := int(gridElement.Y)

		if !IsInVision(world, playerX, playerY, enemyX, enemyY) {
			return
		}

		enemies = append(enemies, entity)
	}))

	return enemies, nil
}

// IsInVision はプレイヤーから指定座標が現在見えるかをチェックする。
// リアルタイムの可視性データを使用し、暗闇のタイルは見えないと判定する
func IsInVision(world w.World, playerX, playerY, targetX, targetY int) bool {
	distanceInPixels := geometry.Distance(float64(playerX), float64(playerY), float64(targetX), float64(targetY)) * float64(consts.TileSize)
	visionRadius := float64(consts.VisionRadiusTiles) * float64(consts.TileSize)

	if distanceInPixels > visionRadius {
		return false
	}

	dungeon := GetDungeon(world)
	if dungeon.VisibleTiles == nil {
		return false
	}

	gridElement := gc.GridElement{X: consts.Tile(targetX), Y: consts.Tile(targetY)}
	return dungeon.VisibleTiles[gridElement]
}

// GetVisibleItems は視界内のアイテムエンティティをすべて取得して返す
func GetVisibleItems(world w.World) ([]ecs.Entity, error) {
	playerEntity, err := GetPlayerEntity(world)
	if err != nil {
		return nil, err
	}

	if !playerEntity.HasComponent(world.Components.GridElement) {
		return nil, fmt.Errorf("プレイヤーがGridElementを持っていません")
	}

	playerGrid := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)
	playerX := int(playerGrid.X)
	playerY := int(playerGrid.Y)

	var items []ecs.Entity

	world.Manager.Join(
		world.Components.GridElement,
		world.Components.LocationOnField,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
		itemX := int(gridElement.X)
		itemY := int(gridElement.Y)

		if !IsInVision(world, playerX, playerY, itemX, itemY) {
			return
		}

		items = append(items, entity)
	}))

	return items, nil
}

// GetEntityName はエンティティの名前を取得する
func GetEntityName(entity ecs.Entity, world w.World) string {
	name := world.Components.Name.Get(entity)
	if name != nil {
		return name.(*gc.Name).Name
	}
	return "Unknown"
}

// AppendNameWithColor はエンティティの種類に応じて色付きで名前を追加する
func AppendNameWithColor(logger *gamelog.Logger, entity ecs.Entity, name string, world w.World) {
	if entity.HasComponent(world.Components.Player) {
		logger.PlayerName(name)
	} else if entity.HasComponent(world.Components.AI) {
		logger.NPCName(name)
	} else {
		logger.Append(name)
	}
}
