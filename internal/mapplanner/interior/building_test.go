package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubdivideBuilding_同じseedで完全一致する は分割文法の決定性を固定する。再訪一致と serde の前提で、
// 同じ footprint と seed からは同じ部屋群と戸口が出る。
func TestSubdivideBuilding_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	first := SubdivideBuilding(footprint, 7)
	for range 5 {
		require.Equal(t, first, SubdivideBuilding(footprint, 7), "同じ seed なら部屋も戸口も完全一致する")
	}
}

// TestSubdivideBuilding_部屋がfootprint内に収まり重ならない は分割の健全性を固定する。部屋は footprint を
// はみ出さず、内側床どうしが重複しない。BSP の共有壁は両隣の周壁なので、内側床だけを見れば排他になる。
func TestSubdivideBuilding_部屋がfootprint内に収まり重ならない(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	rooms := SubdivideBuilding(footprint, 3)
	seen := make(map[Vec]bool)
	for _, rm := range rooms {
		r := rm.Rect
		require.GreaterOrEqual(t, r.X, footprint.X, "左端が footprint 内")
		require.GreaterOrEqual(t, r.Y, footprint.Y, "上端が footprint 内")
		require.LessOrEqual(t, r.X+r.W, footprint.X+footprint.W, "右端が footprint 内")
		require.LessOrEqual(t, r.Y+r.H, footprint.Y+footprint.H, "下端が footprint 内")
		for _, v := range r.interiorTiles() {
			require.Falsef(t, seen[v], "内側床 %v が複数部屋に重複しない", v)
			seen[v] = true
		}
	}
}

// TestRoomDepths_入口が距離0で全室が到達可能 はゾーン分類の基礎を固定する。入口の間はちょうど1室で
// 距離 0、全室が入口から到達できて距離が非負、入口より奥の部屋が必ず存在する。役割割り当てはこの距離
// 順に乗るので、ここが崩れると玄関ホールが奥に、寝室が入口に来るといった動線の破綻になる。
func TestRoomDepths_入口が距離0で全室が到達可能(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	for seed := range uint64(30) {
		rooms := SubdivideBuilding(footprint, seed)
		addEntrance(footprint, rooms) // roomDepths は入口を距離0の起点にするので建物入口を1つ開ける
		depths := roomDepths(rooms)

		zeros := 0
		maxDepth := roomDepth(0)
		for _, d := range depths {
			require.NotEqualf(t, depthUnreachable, d, "seed=%d では全室が入口から到達できる", seed)
			if d == 0 {
				zeros++
			}
			if d > maxDepth {
				maxDepth = d
			}
		}
		assert.Equalf(t, 1, zeros, "seed=%d では入口の間がちょうど1室", seed)
		assert.Positivef(t, maxDepth, "seed=%d では入口より奥の部屋が存在する", seed)
	}
}

// addEntrance は footprint 下辺の中央に近い部屋へ建物入口を開ける。外から屋内への戸口。production の入口は
// 敷地計画 Site が開けるのでこれは使わないが、roomDepths の起点にする建物入口を持つ部屋群をテストで作る。
func addEntrance(footprint Rect, rooms []Room) {
	bottom := footprint.Y + footprint.H - 1
	cx := footprint.X + footprint.W/2
	best := -1
	bestDist := 1 << 30
	for i, r := range rooms {
		if r.Rect.Y+r.Rect.H-1 != bottom {
			continue // 下辺に接する部屋だけ
		}
		if cx <= r.Rect.X || cx >= r.Rect.X+r.Rect.W-1 {
			continue // 中央列がその部屋の内側にある
		}
		d := abs(cx - (r.Rect.X + r.Rect.W/2))
		if d < bestDist {
			bestDist, best = d, i
		}
	}
	if best < 0 {
		best = 0 // 保険。どれかに開ける
	}
	rr := rooms[best].Rect
	ex := cx
	if ex <= rr.X || ex >= rr.X+rr.W-1 {
		ex = rr.X + rr.W/2
	}
	rooms[best].Doorways = append(rooms[best].Doorways, Doorway{X: ex, Y: bottom})
}

// allRoomsConnected は戸口を共有する部屋を隣接とみなし、0 番から BFS で全部屋に届くかを確かめる。
func allRoomsConnected(rooms []Room) bool {
	doorRooms := make(map[Vec][]int)
	for i, r := range rooms {
		for _, d := range r.Doorways {
			v := Vec(d)
			doorRooms[v] = append(doorRooms[v], i)
		}
	}
	seen := map[int]bool{0: true}
	stack := []int{0}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, d := range rooms[cur].Doorways {
			for _, nb := range doorRooms[Vec(d)] {
				if !seen[nb] {
					seen[nb] = true
					stack = append(stack, nb)
				}
			}
		}
	}
	return len(seen) == len(rooms)
}
