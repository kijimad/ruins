package activity

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

type bfsGrid struct {
	world  w.World
	mover  ecs.Entity
	si     *gc.SpatialIndex
	width  int
	height int
}

func (g *bfsGrid) isPassable(x, y int) bool {
	if x < 0 || y < 0 || x >= g.width || y >= g.height {
		return false
	}
	key := gc.GridElement{X: consts.Tile(x), Y: consts.Tile(y)}
	if g.si.BlockPass[key] {
		return false
	}
	if target, ok := g.si.Characters[key]; ok {
		return CanSwapPosition(g.world, g.mover, target)
	}
	return true
}

func (g *bfsGrid) canPassDiagonal(cx, cy, dx, dy int) bool {
	if dx == 0 || dy == 0 {
		return true
	}
	return g.isPassable(cx+dx, cy) || g.isPassable(cx, cy+dy)
}

// FindNextStep はBFSで最短経路を求め、次の1歩の座標を返す。
// 経路が見つからない場合はfalseを返す。
// ゴールが通行不能でも到達を認識する。ゴールはキューに入れないので通り抜ける経路は生まれない。
// キャラクターの通行可否はmoverとの関係性で決まる
func FindNextStep(world w.World, mover ecs.Entity, from, goal consts.Coord[consts.Tile]) (consts.Coord[consts.Tile], bool) {
	// BFS 内部はタイルインデックスの int 演算なので、境界でだけ int へ展開する
	fromX, fromY, goalX, goalY := int(from.X), int(from.Y), int(goal.X), int(goal.Y)
	si := query.GetSpatialIndex(world)
	if si == nil {
		return consts.Coord[consts.Tile]{}, false
	}

	if fromX == goalX && fromY == goalY {
		return consts.Coord[consts.Tile]{}, false
	}

	width, height := si.MapWidth, si.MapHeight

	if goalX < 0 || goalY < 0 || goalX >= width || goalY >= height {
		return consts.Coord[consts.Tile]{}, false
	}
	if fromX < 0 || fromY < 0 || fromX >= width || fromY >= height {
		return consts.Coord[consts.Tile]{}, false
	}

	g := &bfsGrid{world: world, mover: mover, si: si, width: width, height: height}

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

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, d := range dirs {
			nx, ny := cur.x+d[0], cur.y+d[1]

			isGoal := nx == goalX && ny == goalY

			if !isGoal && !g.isPassable(nx, ny) {
				continue
			}

			if !g.canPassDiagonal(cur.x, cur.y, d[0], d[1]) {
				continue
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
				return consts.Coord[consts.Tile]{X: consts.Tile(step.x), Y: consts.Tile(step.y)}, true
			}

			queue = append(queue, coord{nx, ny})
		}
	}

	return consts.Coord[consts.Tile]{}, false
}
