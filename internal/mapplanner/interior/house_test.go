package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlanHouse_同じseedで完全一致する は廊下型間取りの決定性を固定する。再訪一致と serde の前提。
func TestPlanHouse_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 28, H: 20}
	first := PlanHouse(footprint, 1)
	for range 5 {
		require.Equal(t, first, PlanHouse(footprint, 1), "同じ seed なら部屋も戸口も完全一致する")
	}
}

// TestPlanHouse_全室が玄関から戸口で連結する は最重要の不変条件を固定する。廊下の奥や水回りまで、
// どの部屋にも入口の玄関から戸口を辿って到達できること。連結が壊れると入れない部屋ができる。
func TestPlanHouse_全室が玄関から戸口で連結する(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 28, H: 20}
	for seed := range uint64(20) {
		rooms := houseRooms(PlanHouse(footprint, seed))
		assert.Truef(t, allRoomsConnected(rooms), "seed=%d で全室が玄関から連結する", seed)
	}
}

// TestPlanHouse_玄関と廊下と水回りの小部屋を持つ は廊下型が要求する間取りの階層を固定する。玄関・
// 廊下・浴室・脱衣所・トイレの役割がそろい、玄関と水回りは居間より小さいこと。純 BSP では保証できない
// 構造なので、これが崩れると「いきなり広い部屋」に退行する。
func TestPlanHouse_玄関と廊下と水回りの小部屋を持つ(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 28, H: 20}
	plan := PlanHouse(footprint, 1)

	area := map[string]int{}
	for _, hr := range plan {
		area[hr.Role] = hr.Room.Rect.W * hr.Room.Rect.H
	}
	for _, role := range []string{"genkan", "corridor", "living", "kitchen", "bedroom", "dressing", "bath", "toilet"} {
		assert.Containsf(t, area, role, "役割 %s の部屋がある", role)
	}
	assert.Less(t, area["genkan"], area["living"], "玄関は居間より狭い前室")
	assert.Less(t, area["bath"], area["living"], "浴室は居間より狭い小部屋")
	assert.Less(t, area["toilet"], area["kitchen"], "トイレは台所より狭い小部屋")
}

// TestPlanHouse_部屋がfootprint内に収まり重ならない は間取りの健全性を固定する。部屋は footprint を
// はみ出さず、内側床どうしが重複しない。
func TestPlanHouse_部屋がfootprint内に収まり重ならない(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 28, H: 20}
	seen := make(map[Vec]bool)
	for _, hr := range PlanHouse(footprint, 1) {
		r := hr.Room.Rect
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

// houseRooms は HouseRoom 列から Room 列を取り出す。連結性検査など幾何だけを見る補助で使う。
func houseRooms(plan []HouseRoom) []Room {
	rooms := make([]Room, len(plan))
	for i, hr := range plan {
		rooms[i] = hr.Room
	}
	return rooms
}
