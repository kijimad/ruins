package activity

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

type bfsGrid struct {
	world w.World
	mover ecs.Entity
	si    *gc.SpatialIndex
}

func (g *bfsGrid) isPassable(pos consts.Coord[consts.Tile]) bool {
	if pos.X < 0 || pos.Y < 0 || pos.X >= g.si.MapWidth || pos.Y >= g.si.MapHeight {
		return false
	}
	key := gc.GridElement{Coord: pos}
	if g.si.BlockPass[key] {
		return false
	}
	if target, ok := g.si.Characters[key]; ok {
		return CanSwapPosition(g.world, g.mover, target)
	}
	return true
}

// canPassDiagonal は斜め移動時の壁すり抜けを禁じる。隣接する直交2方向の
// どちらかが通行可能なら斜めに進める。d は移動方向で各成分は -1/0/1
func (g *bfsGrid) canPassDiagonal(cur, d consts.Coord[consts.Tile]) bool {
	if d.X == 0 || d.Y == 0 {
		return true
	}
	return g.isPassable(cur.Add(consts.Coord[consts.Tile]{X: d.X})) ||
		g.isPassable(cur.Add(consts.Coord[consts.Tile]{Y: d.Y}))
}

// FindNextStep はBFSで最短経路を求め、次の1歩の座標を返す。
// 経路が見つからない場合はfalseを返す。
// ゴールが通行不能でも到達を認識する。ゴールはキューに入れないので通り抜ける経路は生まれない。
// キャラクターの通行可否はmoverとの関係性で決まる
func FindNextStep(world w.World, mover ecs.Entity, from, goal consts.Coord[consts.Tile]) (consts.Coord[consts.Tile], bool) {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return consts.Coord[consts.Tile]{}, false
	}

	if from == goal {
		return consts.Coord[consts.Tile]{}, false
	}

	inBounds := func(p consts.Coord[consts.Tile]) bool {
		return p.X >= 0 && p.Y >= 0 && p.X < si.MapWidth && p.Y < si.MapHeight
	}
	if !inBounds(from) || !inBounds(goal) {
		return consts.Coord[consts.Tile]{}, false
	}

	g := &bfsGrid{world: world, mover: mover, si: si}

	// visited は探索済みタイル、firstStep はそのタイルへ到達する最初の1歩を記録する
	visited := map[consts.Coord[consts.Tile]]bool{from: true}
	firstStep := map[consts.Coord[consts.Tile]]consts.Coord[consts.Tile]{}

	queue := []consts.Coord[consts.Tile]{from}

	dirs := []consts.Coord[consts.Tile]{
		{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0},
		{X: -1, Y: -1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: 1, Y: 1},
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, d := range dirs {
			next := cur.Add(d)

			isGoal := next == goal

			if !isGoal && !g.isPassable(next) {
				continue
			}
			if !g.canPassDiagonal(cur, d) {
				continue
			}
			if visited[next] {
				continue
			}
			visited[next] = true

			if cur == from {
				firstStep[next] = next
			} else {
				firstStep[next] = firstStep[cur]
			}

			if isGoal {
				return firstStep[next], true
			}

			queue = append(queue, next)
		}
	}

	return consts.Coord[consts.Tile]{}, false
}
