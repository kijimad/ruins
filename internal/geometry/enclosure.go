package geometry

// EnclosedRegion は幅 width 高さ height の矩形グリッド内で、seed から blocked セルを
// 越えずに4方向で繋がるセル集合と、その領域が矩形の外周セルへ達したかを返す。
// 囲われの判定に使う。外周へ達する領域は囲いの外へ抜けられるので屋外、
// 達しない領域は壁に閉ざされているので屋内とみなせる。
// セルのインデックスは y*width+x。seed が範囲外または blocked のときは空集合を返す。
func EnclosedRegion(width, height int, blocked func(x, y int) bool, seedX, seedY int) (cells []int, touchesEdge bool) {
	if seedX < 0 || seedX >= width || seedY < 0 || seedY >= height || blocked(seedX, seedY) {
		return nil, false
	}

	visited := make([]bool, width*height)
	queue := []int{seedY*width + seedX}
	visited[queue[0]] = true

	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]
		cells = append(cells, idx)

		x, y := idx%width, idx/width
		if x == 0 || x == width-1 || y == 0 || y == height-1 {
			touchesEdge = true
		}

		for _, d := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || nx >= width || ny < 0 || ny >= height {
				continue
			}
			nidx := ny*width + nx
			if visited[nidx] || blocked(nx, ny) {
				continue
			}
			visited[nidx] = true
			queue = append(queue, nidx)
		}
	}

	return cells, touchesEdge
}
