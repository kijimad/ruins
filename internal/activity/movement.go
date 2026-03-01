package activity

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// CanMoveTo は指定位置に移動可能かチェックする
//
// 移動判定システムの設計:
//
// このゲームでは二重の通行可否システムが採用されています：
//
// ## 1. タイルレベルでの通行可否判定 (マップ生成時)
//
// マップ生成フェーズで使用される論理的な通行可否判定です。
//
//   - TileFloor: 通行可能
//   - TileWall: 通行不可
//   - mapplanner.PathFinder.IsWalkable() で判定
//   - 用途: 接続性検証、部屋配置、コリドー生成
//
// ## 2. エンティティレベルでの通行可否判定 (実行時)
//
// ゲーム実行時に使用される動的な通行可否判定です。
//
//   - BlockPassコンポーネントを持つエンティティ: 通行不可
//   - activity.CanMoveTo() で判定
//   - 用途: プレイヤー・AI移動時の衝突チェック
//
// ## システム間の一貫性
//
// マップ生成時にタイルからエンティティへの変換が行われ、一貫性が保たれます：
//
//   - TileWall → BlockPass付きエンティティ (通行不可)
//   - TileFloor → 通行可能エンティティ
func CanMoveTo(world w.World, tileX, tileY int, movingEntity ecs.Entity) bool {
	// 基本的な境界チェック（実際のマップサイズを使用）
	mapWidth := int(world.Resources.Dungeon.Level.TileWidth)
	mapHeight := int(world.Resources.Dungeon.Level.TileHeight)
	if tileX < 0 || tileY < 0 || tileX >= mapWidth || tileY >= mapHeight {
		return false
	}

	// 他のエンティティとの衝突チェック
	canMove := true

	// 壁やブロックとの衝突チェック
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.BlockPass,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		// 自分自身は除外
		if entity == movingEntity {
			return
		}

		// 死亡しているエンティティは除外
		if entity.HasComponent(world.Components.Dead) {
			return
		}

		gridElementComponent := world.Components.GridElement.Get(entity)
		if gridElementComponent == nil {
			return
		}
		gridElement := gridElementComponent.(*gc.GridElement)
		if int(gridElement.X) == tileX && int(gridElement.Y) == tileY {
			canMove = false
		}
	}))

	// キャラクター同士の衝突チェック（プレイヤー、敵）
	if canMove {
		world.Manager.Join(
			world.Components.GridElement,
		).Visit(ecs.Visit(func(entity ecs.Entity) {
			// 自分自身は除外
			if entity == movingEntity {
				return
			}

			// 死亡しているエンティティは除外
			if entity.HasComponent(world.Components.Dead) {
				return
			}

			// キャラクターエンティティのみチェック（プレイヤーまたは敵AI）
			isCharacter := entity.HasComponent(world.Components.Player) || entity.HasComponent(world.Components.AIMoveFSM)
			if !isCharacter {
				return
			}

			gridElementComponent := world.Components.GridElement.Get(entity)
			if gridElementComponent == nil {
				return
			}
			gridElement := gridElementComponent.(*gc.GridElement)
			if int(gridElement.X) == tileX && int(gridElement.Y) == tileY {
				canMove = false
			}
		}))
	}

	return canMove
}
