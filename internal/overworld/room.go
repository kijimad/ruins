package overworld

import (
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
)

// rrect は部屋の床範囲。すべて閉区間で、壁はこの範囲の外側にある。
type rrect struct{ x0, y0, x1, y1 consts.Tile }

func (r rrect) width() consts.Tile  { return r.x1 - r.x0 + 1 }
func (r rrect) height() consts.Tile { return r.y1 - r.y0 + 1 }

// contains は座標が部屋の床範囲に入るかを返す。
func (r rrect) contains(x, y consts.Tile) bool {
	return x >= r.x0 && x <= r.x1 && y >= r.y0 && y <= r.y1
}

// wallSeg は部屋を仕切る内壁。縦または横の直線で、1マスだけドアの開口を持つ。
type wallSeg struct {
	x0, y0, x1, y1 consts.Tile
	doorX, doorY   consts.Tile
}

func (s wallSeg) vertical() bool { return s.x0 == s.x1 }

// subdivideBuilding は部屋 r を depth 段まで再帰分割し、部屋一覧と、ドアを開けた仕切り壁一覧を返す。
// ドアは分割時でなく後処理で「両側が床になる位置」に開けるので、親の壁のドアが子の壁と
// 交差して部屋が孤立する不具合を避ける。全部屋が必ず連結する。最小の一辺は cityMinRoom。
func subdivideBuilding(rng *rand.Rand, r rrect) ([]rrect, []wallSeg) {
	rooms, walls := splitRooms(rng, r, cityRoomDepth)
	placeDoors(rng, walls)
	return rooms, walls
}

// splitRooms は部屋 r を再帰的に2分割し、部屋の床範囲一覧と仕切り壁(ドア未設定)を返す。
// 長い辺を割る。両側に cityMinRoom 以上を残せないときは割らない。BSP の部屋割り。
func splitRooms(rng *rand.Rand, r rrect, depth int) ([]rrect, []wallSeg) {
	w, h := r.width(), r.height()
	// 壁が1列(行)を消費するので、両側に cityMinRoom 残すには 2*cityMinRoom+1 以上が要る
	canX := w >= 2*cityMinRoom+1
	canY := h >= 2*cityMinRoom+1
	if depth <= 0 || (!canX && !canY) {
		return []rrect{r}, nil
	}

	if canX && (!canY || w >= h) {
		cx := r.x0 + cityMinRoom + consts.Tile(rng.IntN(int(w-2*cityMinRoom)))
		leftRooms, leftWalls := splitRooms(rng, rrect{r.x0, r.y0, cx - 1, r.y1}, depth-1)
		rightRooms, rightWalls := splitRooms(rng, rrect{cx + 1, r.y0, r.x1, r.y1}, depth-1)
		wall := wallSeg{x0: cx, y0: r.y0, x1: cx, y1: r.y1}
		return append(leftRooms, rightRooms...), append(append([]wallSeg{wall}, leftWalls...), rightWalls...)
	}

	cy := r.y0 + cityMinRoom + consts.Tile(rng.IntN(int(h-2*cityMinRoom)))
	topRooms, topWalls := splitRooms(rng, rrect{r.x0, r.y0, r.x1, cy - 1}, depth-1)
	botRooms, botWalls := splitRooms(rng, rrect{r.x0, cy + 1, r.x1, r.y1}, depth-1)
	wall := wallSeg{x0: r.x0, y0: cy, x1: r.x1, y1: cy}
	return append(topRooms, botRooms...), append(append([]wallSeg{wall}, topWalls...), botWalls...)
}

// placeDoors は各仕切り壁に、垂直な両隣がともに床(壁でない)になる位置へドアを開ける。
// これで隣接する2部屋が確実につながり、親子の壁の交差でドアが塞がれる事故を防ぐ。
func placeDoors(rng *rand.Rand, walls []wallSeg) {
	wallSet := map[consts.Coord[consts.Tile]]bool{}
	for _, s := range walls {
		for x := s.x0; x <= s.x1; x++ {
			for y := s.y0; y <= s.y1; y++ {
				wallSet[consts.Coord[consts.Tile]{X: x, Y: y}] = true
			}
		}
	}
	isWall := func(x, y consts.Tile) bool { return wallSet[consts.Coord[consts.Tile]{X: x, Y: y}] }

	for i := range walls {
		s := &walls[i]
		var cands []consts.Coord[consts.Tile]
		if s.vertical() {
			for y := s.y0; y <= s.y1; y++ {
				if !isWall(s.x0-1, y) && !isWall(s.x0+1, y) {
					cands = append(cands, consts.Coord[consts.Tile]{X: s.x0, Y: y})
				}
			}
		} else {
			for x := s.x0; x <= s.x1; x++ {
				if !isWall(x, s.y0-1) && !isWall(x, s.y0+1) {
					cands = append(cands, consts.Coord[consts.Tile]{X: x, Y: s.y0})
				}
			}
		}
		if len(cands) == 0 {
			// 退避。両側床の位置が無い退化ケースは壁の中央に開ける
			cands = []consts.Coord[consts.Tile]{{X: (s.x0 + s.x1) / 2, Y: (s.y0 + s.y1) / 2}}
		}
		d := cands[rng.IntN(len(cands))]
		s.doorX, s.doorY = d.X, d.Y
	}
}

// roomsConnected は全部屋がドアの開口を通って連結しているかを、床マスの塗りつぶしで検証する。
// テストで部屋分割が孤立部屋を作らないことを固定する。
func roomsConnected(rooms []rrect, walls []wallSeg) bool {
	if len(rooms) <= 1 {
		return true
	}
	floor := map[consts.Coord[consts.Tile]]bool{}
	for _, r := range rooms {
		for x := r.x0; x <= r.x1; x++ {
			for y := r.y0; y <= r.y1; y++ {
				floor[consts.Coord[consts.Tile]{X: x, Y: y}] = true
			}
		}
	}
	for _, s := range walls {
		floor[consts.Coord[consts.Tile]{X: s.doorX, Y: s.doorY}] = true // ドアの開口も床
	}

	start := consts.Coord[consts.Tile]{X: rooms[0].x0, Y: rooms[0].y0}
	seen := map[consts.Coord[consts.Tile]]bool{start: true}
	stack := []consts.Coord[consts.Tile]{start}
	for len(stack) > 0 {
		c := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, n := range []consts.Coord[consts.Tile]{{X: c.X - 1, Y: c.Y}, {X: c.X + 1, Y: c.Y}, {X: c.X, Y: c.Y - 1}, {X: c.X, Y: c.Y + 1}} {
			if floor[n] && !seen[n] {
				seen[n] = true
				stack = append(stack, n)
			}
		}
	}
	for _, r := range rooms {
		if !seen[consts.Coord[consts.Tile]{X: r.x0, Y: r.y0}] {
			return false
		}
	}
	return true
}
