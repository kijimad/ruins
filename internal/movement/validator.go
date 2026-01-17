// Package movement provides movement validation logic shared between player and AI systems.
//
// このパッケージは移動判定の責務を持つ：
//   - エンティティ衝突チェック
//   - マップ境界チェック
//   - 通行可否判定
package movement

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// CanMoveTo は指定位置に移動可能かチェックする
func CanMoveTo(world w.World, tileX, tileY int, movingEntity ecs.Entity) bool {
	// 基本的な境界チェック
	if tileX < 0 || tileY < 0 || tileX >= consts.MapTileWidth || tileY >= consts.MapTileHeight {
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
