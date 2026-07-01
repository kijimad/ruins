package activity

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
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
//   - 静的障害物: BlockPassコンポーネントを持つProp（壁・ドア）は常に通行不可
//   - キャラクター: 関係性で判定する。味方同士は位置交換可能、敵はブロック
//   - activity.CanMoveTo() で判定
//   - 用途: プレイヤー・AI移動時の衝突チェック
//
// ## システム間の一貫性
//
// マップ生成時にタイルからエンティティへの変換が行われ、一貫性が保たれます：
//
//   - TileWall → BlockPass付きエンティティ (通行不可)
//   - TileFloor → 通行可能エンティティ
//
// CanMoveTo は指定位置に移動可能かチェックする。
// fromX, fromY は移動元の座標で、斜め移動時の壁すり抜け防止に使用する
func CanMoveTo(world w.World, tileX, tileY, fromX, fromY int, movingEntity ecs.Entity) bool {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return false
	}

	if tileX < 0 || tileY < 0 || tileX >= si.MapWidth || tileY >= si.MapHeight {
		return false
	}

	// 斜め移動の場合、隣接する直交2方向が両方ブロックされていれば通行不可
	dx := tileX - fromX
	dy := tileY - fromY
	if dx != 0 && dy != 0 {
		if si.IsBlockPass(fromX+dx, fromY) && si.IsBlockPass(fromX, fromY+dy) {
			return false
		}
	}

	if si.IsBlockPass(tileX, tileY) {
		return false
	}

	// キャラクターがいる場合は関係性で判定する
	if target, ok := si.CharacterAt(tileX, tileY); ok {
		return CanPassThrough(world, movingEntity, target)
	}

	return true
}

// CanPassThrough はmoverがtargetのタイルを通過できるかを関係性で判定する。
// 味方同士は位置交換可能。プレイヤーは隊員に押しのけられない
func CanPassThrough(world w.World, mover, target ecs.Entity) bool {
	if mover == target {
		return true
	}
	// プレイヤーは隊員と位置交換できる
	if mover.HasComponent(world.Components.Player) {
		return target.HasComponent(world.Components.SquadMember)
	}
	// 隊員は他の隊員と位置交換できる。プレイヤーは押しのけられない
	if mover.HasComponent(world.Components.SquadMember) {
		return target.HasComponent(world.Components.SquadMember)
	}
	return false
}
