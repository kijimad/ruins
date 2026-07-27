package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// midBuilding は中型民家テンプレを直接叩くときの建物サイズ。横型・縦型のどちらも退化しない本番相当の寸法。
var midBuilding = Rect{X: 0, Y: 0, W: 18, H: 18}

// TestPlanHouseMid_同じseedで完全一致する は中型間取りの決定性を固定する。再訪一致と serde の前提。
func TestPlanHouseMid_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	first := PlanHouseMid(midBuilding, 1)
	for range 5 {
		require.Equal(t, first, PlanHouseMid(midBuilding, 1), "同じ seed なら部屋も戸口も完全一致する")
	}
}

// housePlanners は横廊下と縦廊下の両中型プランナ。廊下の向きが違っても不変条件は共通なので、構造テストは
// 両方に対して回す。
var housePlanners = []struct {
	name string
	plan func(Rect, uint64) []PlannedRoom
}{
	{"horizontal", PlanHouseMid},
	{"vertical", PlanHouseMidV},
}

// TestPlanHouse_全室が玄関から戸口で連結する は最重要の不変条件を固定する。玄関・廊下の奥や水回りまで、
// どの部屋にも戸口を辿って到達できること。連結が壊れると入れない部屋ができる。縦型は浴室・トイレを寝室の
// 奥に nest するので、兄弟経由の到達もここで守る。
func TestPlanHouse_全室が玄関から戸口で連結する(t *testing.T) {
	t.Parallel()

	for _, p := range housePlanners {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			for seed := range uint64(30) {
				rooms := houseRooms(p.plan(midBuilding, seed))
				assert.Truef(t, allRoomsConnected(rooms), "seed=%d で全室が連結する", seed)
			}
		})
	}
}

// TestPlanHouse_玄関と廊下と水回りの小部屋を持つ は中型が要求する間取りの階層を固定する。玄関・廊下・
// 浴室・トイレの役割がそろい、玄関と水回りは居間より小さいこと。純 BSP では保証できない構造なので、これが
// 崩れると「いきなり広い部屋」に退行する。
func TestPlanHouse_玄関と廊下と水回りの小部屋を持つ(t *testing.T) {
	t.Parallel()

	plan := PlanHouseMid(midBuilding, 1)

	area := map[roleName]int{}
	for _, hr := range plan {
		area[hr.Role] = hr.Room.Rect.W * hr.Room.Rect.H
	}
	for _, role := range []roleName{"genkan", "corridor", "living", "kitchen", "bedroom", "bath", "toilet"} {
		assert.Containsf(t, area, role, "役割 %s の部屋がある", role)
	}
	assert.Less(t, area["genkan"], area["living"], "玄関は居間より狭い前室")
	assert.Less(t, area["bath"], area["living"], "浴室は居間より狭い小部屋")
	assert.Less(t, area["toilet"], area["living"], "トイレは居間より狭い小部屋")
}

// TestPlanHouse_部屋がfootprint内に収まり重ならない は間取りの健全性を固定する。部屋は建物をはみ出さず、
// 内側床どうしが重複しない。
func TestPlanHouse_部屋がfootprint内に収まり重ならない(t *testing.T) {
	t.Parallel()

	for _, p := range housePlanners {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			seen := make(map[Vec]bool)
			for _, hr := range p.plan(midBuilding, 1) {
				r := hr.Room.Rect
				require.GreaterOrEqual(t, r.X, midBuilding.X, "左端が建物内")
				require.GreaterOrEqual(t, r.Y, midBuilding.Y, "上端が建物内")
				require.LessOrEqual(t, r.X+r.W, midBuilding.X+midBuilding.W, "右端が建物内")
				require.LessOrEqual(t, r.Y+r.H, midBuilding.Y+midBuilding.H, "下端が建物内")
				for _, v := range r.interiorTiles() {
					require.Falsef(t, seen[v], "内側床 %v が複数部屋に重複しない", v)
					seen[v] = true
				}
			}
		})
	}
}

// TestPlanHouseAny_seedで型が選ばれ両方出る は間取り型の抽選を固定する。同じ seed は同じ型で決定的、seed を
// 振ると横廊下と縦廊下の両方が出て、生成した家はいずれも連結する。高さ16以上の建物で縦型も選ばれる。
func TestPlanHouseAny_seedで型が選ばれ両方出る(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 20, H: 18}
	first := PlanHouseAny(footprint, 3)
	require.Equal(t, first, PlanHouseAny(footprint, 3), "同じ seed なら同じ型で完全一致する")

	seenWide, seenTall := false, false
	for seed := range uint64(30) {
		plan := PlanHouseAny(footprint, seed)
		assert.Truef(t, allRoomsConnected(houseRooms(plan)), "seed=%d の家は連結する", seed)
		for _, hr := range plan {
			if hr.Role != roleCorridor {
				continue
			}
			if hr.Room.Rect.W > hr.Room.Rect.H {
				seenWide = true // 横廊下
			} else {
				seenTall = true // 縦廊下
			}
		}
	}
	assert.True(t, seenWide, "seed を振ると横廊下の家が出る")
	assert.True(t, seenTall, "seed を振ると縦廊下の家が出る")
}

// houseRooms は PlannedRoom 列から Room 列を取り出す。連結性検査など幾何だけを見る補助で使う。
func houseRooms(plan []PlannedRoom) []Room {
	rooms := make([]Room, len(plan))
	for i, hr := range plan {
		rooms[i] = hr.Room
	}
	return rooms
}
