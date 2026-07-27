package interior

// 到達性は有界インスタンスでは必須のガード。塞がった部屋はその seed で永久に塞がる。配置後に
// 戸口から歩行可能な床が全て到達できるかを flood-fill で検査する。ここでは検査だけを提供し、失敗時の修復は後続 Stage で足す。

// containsInterior は v が外周の壁を除いた内側タイルかを返す。
func (r Rect) containsInterior(v Vec) bool {
	return v.X > r.X && v.X < r.X+r.W-1 && v.Y > r.Y && v.Y < r.Y+r.H-1
}

// blockingTiles は歩行を阻む配置のタイル集合を返す。家具は占有して通行を阻む。戦利品や装飾は阻まない。
func blockingTiles(placed []Placed) map[Vec]bool {
	blocked := make(map[Vec]bool)
	for _, p := range placed {
		if p.Kind == KindFurniture {
			blocked[p.Pos] = true
		}
	}
	return blocked
}

// reachableFloor は戸口から歩行可能な内側タイルを4近傍 BFS で塗って返す。blocked のタイルは通れない。
// 戸口が無い部屋は空を返す。map は集合の membership 判定にのみ使い、抽選には使わない。
func reachableFloor(room Room, blocked map[Vec]bool) map[Vec]bool {
	seen := make(map[Vec]bool)
	queue := make([]Vec, 0, room.Rect.W*room.Rect.H)
	enqueue := func(t Vec) {
		if room.Rect.containsInterior(t) && !blocked[t] && !seen[t] {
			seen[t] = true
			queue = append(queue, t)
		}
	}
	// 各戸口の内側隣接タイルを起点にする
	for _, d := range room.Doorways {
		for _, n := range neighbors4(Vec(d)) {
			enqueue(n)
		}
	}
	for len(queue) > 0 {
		t := queue[0]
		queue = queue[1:]
		for _, n := range neighbors4(t) {
			enqueue(n)
		}
	}
	return seen
}

// repairReachability は歩行可能な床が戸口から全て到達できるよう、通路を塞ぐ家具を最小限撤回する。
// 到達領域と未到達床の境目に立つ家具を固定順で1つ外して開通させ、全床が繋がるまで繰り返す。
// 各回で必ず到達領域が広がるので有限回で止まる。「blocker を固定順で撤回」の最小実装。
func repairReachability(room Room, placed []Placed) []Placed {
	for {
		blocked := blockingTiles(placed)
		reached := reachableFloor(room, blocked)
		gate := gateIndex(room, placed, blocked, reached)
		if gate < 0 {
			return placed
		}
		placed = append(placed[:gate:gate], placed[gate+1:]...)
	}
}

// gateIndex は到達領域と未到達の床の境目に立つ家具の添字を返す。外せば未到達領域が繋がる。
// 無ければ -1。戸口の内側は配置対象外なので戸口が塞がることはなく、到達領域は常に非空になる。
func gateIndex(room Room, placed []Placed, blocked, reached map[Vec]bool) int {
	for i, p := range placed {
		if p.Kind != KindFurniture {
			continue
		}
		nearReached, nearUnreachedFloor := false, false
		for _, n := range neighbors4(p.Pos) {
			if !room.Rect.containsInterior(n) {
				continue
			}
			if reached[n] {
				nearReached = true
			} else if !blocked[n] {
				nearUnreachedFloor = true
			}
		}
		if nearReached && nearUnreachedFloor {
			return i
		}
	}
	return -1
}

// neighbors4 は上下左右の4近傍。
func neighbors4(v Vec) [4]Vec {
	return [4]Vec{
		{X: v.X, Y: v.Y - 1},
		{X: v.X, Y: v.Y + 1},
		{X: v.X - 1, Y: v.Y},
		{X: v.X + 1, Y: v.Y},
	}
}
