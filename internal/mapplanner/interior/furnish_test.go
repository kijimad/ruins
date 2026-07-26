package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFurnishBuilding_大きい建物は多部屋になる は分割配線の要点を固定する。割れる大きさの footprint は
// 内部間仕切りを持ち、家具が置かれる。単室では倉庫のように間延びする大きな建物へ構造を与える。
func TestFurnishBuilding_大きい建物は多部屋になる(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	door := Vec{X: 13, Y: 0} // 北壁の入口
	site, placed := FurnishBuilding(1, footprint, door, "store")
	require.NotEmpty(t, site.Walls(), "割れる大きさの建物は内部間仕切りを持つ")
	require.NotEmpty(t, placed, "家具が置かれる")
}

// TestFurnishBuilding_同じseedで完全一致する は多部屋生成の決定性を固定する。再訪一致と serde の前提。
func TestFurnishBuilding_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	door := Vec{X: 13, Y: 0}
	s1, p1 := FurnishBuilding(1, footprint, door, "store")
	for range 5 {
		s2, p2 := FurnishBuilding(1, footprint, door, "store")
		require.Equal(t, s1.Walls(), s2.Walls(), "間仕切りが完全一致する")
		require.Equal(t, p1, p2, "配置が完全一致する")
	}
}

// TestFurnishBuilding_入口が部屋に繋がる は、外殻の入口が建物内の部屋へ戸口として繋がる不変条件を固定する。
// 前庭ぶん建物を内寄せしポーチで入口を下げても、入口はいずれかの部屋の戸口になり、建物へ入れる。
func TestFurnishBuilding_入口が部屋に繋がる(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	for _, door := range []Vec{{X: 13, Y: 0}, {X: 0, Y: 9}} { // 北壁・西壁
		site, _ := FurnishBuilding(1, footprint, door, "store")
		connected := false
		for _, hr := range site.Rooms {
			for _, d := range hr.Room.Doorways {
				if Vec(d) == site.Door {
					connected = true
				}
			}
		}
		assert.Truef(t, connected, "入口 %v が部屋の戸口に繋がる", door)
	}
}

// TestSubdivideBuilding_全部屋が連結する は最重要の不変条件を固定する。外殻の扉から入った部屋から全室へ
// 戸口を辿って到達できるよう、分割した部屋が相互に連結すること。
func TestSubdivideBuilding_全部屋が連結する(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	for seed := range uint64(20) {
		rooms := SubdivideBuilding(footprint, seed)
		assert.Truef(t, allRoomsConnected(rooms), "seed=%d で全部屋が戸口で連結する", seed)
	}
}

// TestFurnishBuilding_施設テンプレが多seedで奥室を役割へ分化する は施設固有テンプレが実経路で使われ、
// BSP フォールバックへ落ちないことを多 seed で固定する。前庭ぶん建物が縮んでテンプレ下限を割り、奥室が
// 全部 back の樽物置になった退行を検知する。in-game の footprint でテンプレは BSP の "back" を出さず、
// 施設固有の奥室役割を必ず出す。ゴールデンの目視より前に役割の分化崩れをここで止める。
func TestFurnishBuilding_施設テンプレが多seedで奥室を役割へ分化する(t *testing.T) {
	t.Parallel()

	cases := []struct {
		facility  string
		footprint Rect
		door      Vec
		backRoles []string // このどれかが必ず出る施設固有の奥室役割
	}{
		{"store", Rect{X: 0, Y: 0, W: 26, H: 18}, Vec{X: 13, Y: 0}, []string{"storeroom", "office", "restroom", "coldroom"}},
		{"clinic", Rect{X: 0, Y: 0, W: 26, H: 18}, Vec{X: 13, Y: 0}, []string{"exam", "pharmacy", "restroom", "office"}},
		{"house", Rect{X: 0, Y: 0, W: 28, H: 20}, Vec{X: 14, Y: 0}, []string{"kitchen", "bedroom", "bath", "toilet"}},
	}
	for _, c := range cases {
		for seed := range uint64(30) {
			site, _ := FurnishBuilding(seed, c.footprint, c.door, c.facility)
			roles := map[string]int{}
			for _, r := range site.Rooms {
				roles[r.Role]++
			}
			assert.NotContainsf(t, roles, "back", "%s seed=%d はテンプレを使い BSP フォールバックの back を出さない", c.facility, seed)
			has := false
			for _, r := range c.backRoles {
				if roles[r] > 0 {
					has = true
					break
				}
			}
			assert.Truef(t, has, "%s seed=%d は施設固有の奥室役割 %v のどれかを出す (roles=%v)", c.facility, seed, c.backRoles, roles)
		}
	}
}

// TestFurnishBuilding_部屋が退化しない は生成される部屋が内側床を必ず持つことを多 seed で固定する。前庭で
// 建物が縮むとテンプレの比率割りで薄い部屋が H<3 の内側床0に潰れ、床が描かれずラベルだけ浮く退行が出た。
// 全施設の全 seed で、庭でない部屋は内側床が1タイル以上あり、ラベルの下に必ず部屋があることを守る。
func TestFurnishBuilding_部屋が退化しない(t *testing.T) {
	t.Parallel()

	cases := []struct {
		facility  string
		footprint Rect
		door      Vec
	}{
		{"house", Rect{X: 0, Y: 0, W: 28, H: 20}, Vec{X: 14, Y: 0}},
		{"store", Rect{X: 0, Y: 0, W: 26, H: 18}, Vec{X: 13, Y: 0}},
		{"clinic", Rect{X: 0, Y: 0, W: 26, H: 18}, Vec{X: 13, Y: 0}},
	}
	for _, c := range cases {
		for seed := range uint64(50) {
			site, _ := FurnishBuilding(seed, c.footprint, c.door, c.facility)
			for _, hr := range site.Rooms {
				if hr.Role == "garden" {
					continue // 坪庭は内側が庭タイルで床を持たない
				}
				assert.NotEmptyf(t, hr.Room.Rect.interiorTiles(), "%s seed=%d の部屋 %s %+v が内側床を持つ", c.facility, seed, hr.Role, hr.Room.Rect)
			}
		}
	}
}
