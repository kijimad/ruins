package interior

import (
	"testing"

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

// allRoomsConnected は戸口を共有する部屋を隣接とみなし、0 番から BFS で全部屋に届くかを確かめる。
func allRoomsConnected(rooms []Room) bool {
	doorRooms := make(map[Vec][]int)
	for i, r := range rooms {
		for _, d := range r.Doorways {
			v := d
			doorRooms[v] = append(doorRooms[v], i)
		}
	}
	seen := map[int]bool{0: true}
	stack := []int{0}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, d := range rooms[cur].Doorways {
			for _, nb := range doorRooms[d] {
				if !seen[nb] {
					seen[nb] = true
					stack = append(stack, nb)
				}
			}
		}
	}
	return len(seen) == len(rooms)
}
