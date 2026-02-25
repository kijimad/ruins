package worldhelper

import (
	"fmt"
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// GetVisibleEnemies は視界内の敵エンティティをすべて取得して返す
func GetVisibleEnemies(world w.World) ([]ecs.Entity, error) {
	// プレイヤー位置を取得
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

	// 視界内の敵を収集
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.FactionEnemy,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
		enemyX := int(gridElement.X)
		enemyY := int(gridElement.Y)

		// 視界チェック（プレイヤーから見えるかどうか）
		if !IsInVision(world, playerX, playerY, enemyX, enemyY) {
			return
		}

		enemies = append(enemies, entity)
	}))

	return enemies, nil
}

// IsInVision はプレイヤーから指定座標が見えるかをチェックする
func IsInVision(world w.World, playerX, playerY, targetX, targetY int) bool {
	// 距離チェック（視界範囲外は見えない）
	dx := targetX - playerX
	dy := targetY - playerY
	distanceInPixels := math.Sqrt(float64(dx*dx+dy*dy)) * float64(consts.TileSize)
	visionRadius := consts.VisionRadiusTiles * float64(consts.TileSize)

	if distanceInPixels > visionRadius {
		return false
	}

	// 探索済みタイルかチェック（探索済みなら見える）
	gridElement := gc.GridElement{X: consts.Tile(targetX), Y: consts.Tile(targetY)}
	return world.Resources.Dungeon.ExploredTiles[gridElement]
}

// GetVisibleItems は視界内のアイテムエンティティをすべて取得して返す
func GetVisibleItems(world w.World) ([]ecs.Entity, error) {
	// プレイヤー位置を取得
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

	// 視界内のアイテムを収集
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.Item,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
		itemX := int(gridElement.X)
		itemY := int(gridElement.Y)

		// 視界チェック（プレイヤーから見えるかどうか）
		if !IsInVision(world, playerX, playerY, itemX, itemY) {
			return
		}

		items = append(items, entity)
	}))

	return items, nil
}
