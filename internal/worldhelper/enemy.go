package worldhelper

import (
	"math"
	"sort"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// EnemyInfo は視界内の敵情報
type EnemyInfo struct {
	Entity   ecs.Entity
	Name     string
	HP       int
	MaxHP    int
	Distance int // タイル単位の距離
	GridX    int
	GridY    int
}

// ItemInfo は視界内のアイテム情報
type ItemInfo struct {
	Entity      ecs.Entity
	Name        string
	Description string
	Distance    int // タイル単位の距離
	GridX       int
	GridY       int
}

// GetVisibleEnemies は視界内の敵をすべて取得し、距離順にソートして返す
func GetVisibleEnemies(world w.World) []EnemyInfo {
	// プレイヤー位置を取得
	playerEntity, err := GetPlayerEntity(world)
	if err != nil {
		return nil
	}

	if !playerEntity.HasComponent(world.Components.GridElement) {
		return nil
	}

	playerGrid := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)
	playerX := int(playerGrid.X)
	playerY := int(playerGrid.Y)

	var enemies []EnemyInfo

	// 視界内の敵を収集
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.FactionEnemy,
		world.Components.Pools,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
		enemyX := int(gridElement.X)
		enemyY := int(gridElement.Y)

		// 視界チェック（プレイヤーから見えるかどうか）
		if !isInVision(world, playerX, playerY, enemyX, enemyY) {
			return
		}

		pools := world.Components.Pools.Get(entity).(*gc.Pools)

		// 名前を取得
		name := "敵"
		if entity.HasComponent(world.Components.Name) {
			nameComp := world.Components.Name.Get(entity).(*gc.Name)
			name = nameComp.Name
		}

		// 距離を計算（タイル単位）
		dx := enemyX - playerX
		dy := enemyY - playerY
		distance := int(math.Sqrt(float64(dx*dx + dy*dy)))

		enemies = append(enemies, EnemyInfo{
			Entity:   entity,
			Name:     name,
			HP:       pools.HP.Current,
			MaxHP:    pools.HP.Max,
			Distance: distance,
			GridX:    enemyX,
			GridY:    enemyY,
		})
	}))

	// 距離順にソート（近い順）
	sort.Slice(enemies, func(i, j int) bool {
		return enemies[i].Distance < enemies[j].Distance
	})

	return enemies
}

// isInVision はプレイヤーから指定座標が見えるかをチェックする
func isInVision(world w.World, playerX, playerY, targetX, targetY int) bool {
	// 距離チェック（視界範囲外は見えない）
	dx := targetX - playerX
	dy := targetY - playerY
	distanceInPixels := math.Sqrt(float64(dx*dx+dy*dy)) * float64(consts.TileSize)
	visionRadius := 16 * float64(consts.TileSize) // VisionSystemと同じ視界半径

	if distanceInPixels > visionRadius {
		return false
	}

	// 探索済みタイルかチェック（探索済みなら見える）
	gridElement := gc.GridElement{X: gc.Tile(targetX), Y: gc.Tile(targetY)}
	return world.Resources.Dungeon.ExploredTiles[gridElement]
}

// GetVisibleItems は視界内のアイテムをすべて取得し、距離順にソートして返す
func GetVisibleItems(world w.World) []ItemInfo {
	// プレイヤー位置を取得
	playerEntity, err := GetPlayerEntity(world)
	if err != nil {
		return nil
	}

	if !playerEntity.HasComponent(world.Components.GridElement) {
		return nil
	}

	playerGrid := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)
	playerX := int(playerGrid.X)
	playerY := int(playerGrid.Y)

	var items []ItemInfo

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

		// 名前を取得
		name := "アイテム"
		if entity.HasComponent(world.Components.Name) {
			nameComp := world.Components.Name.Get(entity).(*gc.Name)
			name = nameComp.Name
		}

		// 説明を取得
		description := ""
		if entity.HasComponent(world.Components.Description) {
			descComp := world.Components.Description.Get(entity).(*gc.Description)
			description = descComp.Description
		}

		// 距離を計算（タイル単位）
		dx := itemX - playerX
		dy := itemY - playerY
		distance := int(math.Sqrt(float64(dx*dx + dy*dy)))

		items = append(items, ItemInfo{
			Entity:      entity,
			Name:        name,
			Description: description,
			Distance:    distance,
			GridX:       itemX,
			GridY:       itemY,
		})
	}))

	// 距離順にソート（近い順）
	sort.Slice(items, func(i, j int) bool {
		return items[i].Distance < items[j].Distance
	})

	return items
}
