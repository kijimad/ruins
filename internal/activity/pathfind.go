package activity

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
)

// FindNextStep はBFSで最短経路を求め、次の1歩の座標を返す。
// 経路が見つからない場合はfalseを返す。
// 隊員のいるタイルは通行可能として扱う
func FindNextStep(world w.World, fromX, fromY, goalX, goalY int) (int, int, bool) {
	si := query.GetSpatialIndex(world)
	if !si.Built || si.MapWidth == 0 || si.MapHeight == 0 {
		return 0, 0, false
	}

	if fromX == goalX && fromY == goalY {
		return 0, 0, false
	}

	width, height := si.MapWidth, si.MapHeight

	type coord struct{ x, y int }

	visited := make([]bool, width*height)
	// 各セルの「1歩目の移動先」を記録する
	firstStep := make([]coord, width*height)

	idx := func(x, y int) int { return y*width + x }

	visited[idx(fromX, fromY)] = true

	queue := []coord{{fromX, fromY}}

	dirs := [8][2]int{
		{0, -1}, {0, 1}, {-1, 0}, {1, 0},
		{-1, -1}, {1, -1}, {-1, 1}, {1, 1},
	}

	isPassable := func(x, y int) bool {
		if x < 0 || y < 0 || x >= width || y >= height {
			return false
		}
		key := gc.GridElement{X: consts.Tile(x), Y: consts.Tile(y)}
		if si.BlockPass[key] {
			// 隊員のタイルは通行可能にする
			if _, ok := si.SquadMembers[key]; ok {
				return true
			}
			return false
		}
		return true
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, d := range dirs {
			nx, ny := cur.x+d[0], cur.y+d[1]

			if !isPassable(nx, ny) {
				continue
			}

			// 斜め移動のすり抜け防止
			if d[0] != 0 && d[1] != 0 {
				if !isPassable(cur.x+d[0], cur.y) && !isPassable(cur.x, cur.y+d[1]) {
					continue
				}
			}

			ni := idx(nx, ny)
			if visited[ni] {
				continue
			}
			visited[ni] = true

			if cur.x == fromX && cur.y == fromY {
				firstStep[ni] = coord{nx, ny}
			} else {
				firstStep[ni] = firstStep[idx(cur.x, cur.y)]
			}

			if nx == goalX && ny == goalY {
				step := firstStep[ni]
				return step.x, step.y, true
			}

			queue = append(queue, coord{nx, ny})
		}
	}

	return 0, 0, false
}
