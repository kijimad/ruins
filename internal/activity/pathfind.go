package activity

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// FindNextStep はBFSで最短経路を求め、次の1歩の座標を返す。
// 経路が見つからない場合はfalseを返す。
// ゴールが通行不能でも到達を認識する。ゴールはキューに入れないので通り抜ける経路は生まれない。
// キャラクターの通行可否はmoverとの関係性で決まる
func FindNextStep(world w.World, mover ecs.Entity, fromX, fromY, goalX, goalY int) (int, int, bool) {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return 0, 0, false
	}

	if fromX == goalX && fromY == goalY {
		return 0, 0, false
	}

	width, height := si.MapWidth, si.MapHeight

	type coord struct{ x, y int }

	visited := make([]bool, width*height)
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
			return false
		}
		if target, ok := si.Characters[key]; ok {
			return CanPassThrough(world, mover, target)
		}
		return true
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, d := range dirs {
			nx, ny := cur.x+d[0], cur.y+d[1]

			isGoal := nx == goalX && ny == goalY

			if !isGoal && !isPassable(nx, ny) {
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

			if isGoal {
				step := firstStep[ni]
				return step.x, step.y, true
			}

			queue = append(queue, coord{nx, ny})
		}
	}

	return 0, 0, false
}
