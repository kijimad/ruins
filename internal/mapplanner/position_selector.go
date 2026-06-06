package mapplanner

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// positionSelector は配置位置を選択する関数型。
type positionSelector func(planData *MetaPlan, world w.World) (consts.Tile, consts.Tile, bool)

// findPosition はセレクタを順に試し、最初に成功した結果を返す。
// 全セレクタが失敗した場合はエラーを返す
func findPosition(planData *MetaPlan, world w.World, selectors ...positionSelector) (consts.Tile, consts.Tile, error) {
	for _, sel := range selectors {
		if x, y, ok := sel(planData, world); ok {
			return x, y, nil
		}
	}
	return 0, 0, fmt.Errorf("配置位置が見つかりません")
}

// inRoomSelector は指定された部屋内からランダムに配置位置を選択する
func inRoomSelector(room gc.Rect, maxAttempts int) positionSelector {
	return func(planData *MetaPlan, world w.World) (consts.Tile, consts.Tile, bool) {
		return planData.randomPositionInRoom(room, world, maxAttempts)
	}
}

// onMapSelector はマップ全体からランダムに配置位置を選択する
func onMapSelector(maxAttempts int) positionSelector {
	return func(planData *MetaPlan, world w.World) (consts.Tile, consts.Tile, bool) {
		for i := 0; i < maxAttempts; i++ {
			x := consts.Tile(planData.RNG.IntN(int(planData.Level.TileWidth)))
			y := consts.Tile(planData.RNG.IntN(int(planData.Level.TileHeight)))
			if planData.IsSpawnableTile(world, x, y) {
				return x, y, true
			}
		}
		return 0, 0, false
	}
}

// nearSelector は指定座標の周辺かつ部屋内から配置位置を選択する
func nearSelector(centerX, centerY consts.Tile, radius int, room gc.Rect, maxAttempts int) positionSelector {
	return func(planData *MetaPlan, world w.World) (consts.Tile, consts.Tile, bool) {
		return planData.randomPositionNear(centerX, centerY, radius, room, world, maxAttempts)
	}
}

// reachableSelector はマップ全体からランダムに選び、プレイヤーから到達可能な位置を返す
func reachableSelector(pf *PathFinder, playerPos consts.Coord[int], maxAttempts int) positionSelector {
	return func(planData *MetaPlan, world w.World) (consts.Tile, consts.Tile, bool) {
		for i := 0; i < maxAttempts; i++ {
			x := consts.Tile(planData.RNG.IntN(int(planData.Level.TileWidth)))
			y := consts.Tile(planData.RNG.IntN(int(planData.Level.TileHeight)))
			if planData.IsSpawnableTile(world, x, y) && pf.IsReachable(playerPos.X, playerPos.Y, int(x), int(y)) {
				return x, y, true
			}
		}
		return 0, 0, false
	}
}
