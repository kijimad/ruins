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
		if !isInVision(world, playerX, playerY, enemyX, enemyY) {
			return
		}

		enemies = append(enemies, entity)
	}))

	return enemies, nil
}

// isInVision はプレイヤーから指定座標が見えるかをチェックする
func isInVision(world w.World, playerX, playerY, targetX, targetY int) bool {
	// 距離チェック（視界範囲外は見えない）
	dx := targetX - playerX
	dy := targetY - playerY
	distanceInPixels := math.Sqrt(float64(dx*dx+dy*dy)) * float64(consts.TileSize)
	visionRadius := consts.VisionRadiusTiles * float64(consts.TileSize)

	if distanceInPixels > visionRadius {
		return false
	}

	// 探索済みタイルかチェック（探索済みなら見える）
	gridElement := gc.GridElement{X: gc.Tile(targetX), Y: gc.Tile(targetY)}
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
		if !isInVision(world, playerX, playerY, itemX, itemY) {
			return
		}

		items = append(items, entity)
	}))

	return items, nil
}

// CalculateDistance は2つのエンティティ間の距離をタイル単位で計算する
func CalculateDistance(world w.World, entity1, entity2 ecs.Entity) int {
	if !entity1.HasComponent(world.Components.GridElement) || !entity2.HasComponent(world.Components.GridElement) {
		return 0
	}

	grid1 := world.Components.GridElement.Get(entity1).(*gc.GridElement)
	grid2 := world.Components.GridElement.Get(entity2).(*gc.GridElement)

	dx := int(grid1.X) - int(grid2.X)
	dy := int(grid1.Y) - int(grid2.Y)

	return int(math.Sqrt(float64(dx*dx + dy*dy)))
}
