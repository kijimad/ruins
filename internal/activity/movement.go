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
//
// CanMoveTo は指定位置に移動可能かチェックする。
// fromX, fromY は移動元の座標で、斜め移動時の壁すり抜け防止に使用する
func CanMoveTo(world w.World, tileX, tileY, fromX, fromY int, movingEntity ecs.Entity) bool {
	si := query.GetSpatialIndex(world)

	if tileX < 0 || tileY < 0 || tileX >= si.MapWidth || tileY >= si.MapHeight {
		return false
	}

	// TODO: BlockPassは壁やProp専用にして、キャラクターからは外す。
	// キャラクターは条件によって通行を許可する場合があり、BlockPassでの一律ブロックが合わない。
	// 隊員・敵の通行判定はIsCharacterAt/IsSquadMemberAtに統合する
	//
	// プレイヤーが自分の隊員のいるタイルに移動する場合は位置入れ替えで許可する。
	// 隊員はBlockPassを持つため、IsBlockPassより先に判定する。
	// SpatialIndex.Charactersには隊員は含まれないが、BlockPassマップには含まれる
	if movingEntity.HasComponent(world.Components.Player) {
		if _, ok := query.SquadMemberAt(world, movingEntity, tileX, tileY); ok {
			return true
		}
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

	return !si.IsCharacterAt(tileX, tileY, movingEntity)
}
