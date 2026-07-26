package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSplitBuilding_同じseedで完全一致する は分割文法の決定性を固定する。再訪一致と serde の前提で、
// 同じ footprint と seed からは同じ部屋群と戸口が出る。
func TestSplitBuilding_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	first := SplitBuilding(footprint, 7)
	for range 5 {
		require.Equal(t, first, SplitBuilding(footprint, 7), "同じ seed なら部屋も戸口も完全一致する")
	}
}

// TestSplitBuilding_全部屋が戸口で連結する は最重要の不変条件を固定する。どの部屋にも建物入口から
// 戸口を辿って到達できること。実装は union-find の全域木で保証するので、テストは戸口グラフ側から BFS で
// 独立に検算し、実装とテストで経路を分ける。連結が壊れると入れない部屋ができ、手動プレイまで気づけない。
func TestSplitBuilding_全部屋が戸口で連結する(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	for seed := range uint64(50) {
		rooms := SplitBuilding(footprint, seed)
		require.GreaterOrEqualf(t, len(rooms), 2, "seed=%d で複数部屋に割れる", seed)
		assert.Truef(t, allRoomsConnected(rooms), "seed=%d で全部屋が戸口で連結する", seed)
	}
}

// TestSplitBuilding_部屋がfootprint内に収まり重ならない は分割の健全性を固定する。部屋は footprint を
// はみ出さず、内側床どうしが重複しない。BSP の共有壁は両隣の周壁なので、内側床だけを見れば排他になる。
func TestSplitBuilding_部屋がfootprint内に収まり重ならない(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	rooms := SplitBuilding(footprint, 3)
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
		rooms := SplitBuilding(footprint, seed)
		depths := roomDepths(footprint, rooms)

		zeros, maxDepth := 0, 0
		for _, d := range depths {
			require.GreaterOrEqualf(t, d, 0, "seed=%d では全室が入口から到達でき距離が非負", seed)
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
