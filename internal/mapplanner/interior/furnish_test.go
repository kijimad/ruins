package interior

import (
	"slices"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoleContent_施設に無い役割は自施設の奥室既定へ落ちる は roleContent のフォールバックを直接固定する。
// カタログに無い役割は、他施設が同名役割を持っていても自施設の fallback content へ落ち、施設間の依存を持たない。
// end-to-end の FurnishBuilding では覆えない中間経路で、ここが壊れると特定役割の部屋が静かに変わる。
func TestRoleContent_施設に無い役割は自施設の奥室既定へ落ちる(t *testing.T) {
	t.Parallel()

	raws := oapi.Raws{
		InteriorContents: &[]oapi.InteriorContent{{Id: "store_fallback"}, {Id: "house_corridor"}},
		FacilityRooms: &[]oapi.FacilityRooms{
			{Facility: "store", Rooms: &[]oapi.RoomContent{}, Fallback: "store_fallback"},
			{Facility: "house", Rooms: &[]oapi.RoomContent{{Role: "corridor", Content: "house_corridor"}}, Fallback: "house_corridor"},
		},
	}

	// store は corridor を持たない。民家が corridor を持っていても store 自身の fallback へ落ちる
	c, err := roleContent(raws, FacilitySpec{ID: "store"}, roleCorridor, 0)
	require.NoError(t, err)
	require.Equal(t, "store_fallback", c.ID)
}

// TestFurnishBuilding_大きい建物は多部屋になる は分割配線の要点を固定する。割れる大きさの footprint は
// 内部間仕切りを持ち、家具が置かれる。単室では倉庫のように間延びする大きな建物へ構造を与える。
func TestFurnishBuilding_大きい建物は多部屋になる(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	door := Vec{X: 13, Y: 0} // 北壁の入口
	site, placed, err := FurnishBuilding(testRaws(), 1, footprint, door, facSpec("store"))
	require.NoError(t, err)
	require.NotEmpty(t, site.Walls(), "割れる大きさの建物は内部間仕切りを持つ")
	require.NotEmpty(t, placed, "家具が置かれる")
}

// TestFurnishBuilding_同じseedで完全一致する は多部屋生成の決定性を固定する。再訪一致と serde の前提。
func TestFurnishBuilding_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	door := Vec{X: 13, Y: 0}
	s1, p1, err := FurnishBuilding(testRaws(), 1, footprint, door, facSpec("store"))
	require.NoError(t, err)
	for range 5 {
		s2, p2, err := FurnishBuilding(testRaws(), 1, footprint, door, facSpec("store"))
		require.NoError(t, err)
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
		site, _, err := FurnishBuilding(testRaws(), 1, footprint, door, facSpec("store"))
		require.NoError(t, err)
		connected := false
		for _, hr := range site.Rooms {
			for _, d := range hr.Room.Doorways {
				if d == site.Door {
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

// TestFurnishBuilding_施設テンプレが本番サイズで奥室を役割へ分化する は施設固有テンプレが in-game の建物
// サイズで発火し、BSP フォールバックへ落ちないことを固定する。本番の市街地チャンク(24x24)が生む footprint
// 17〜20 では建物がテンプレ下限を満たし、店・診療所・民家(コンパクト)の役割ごとに部屋が分化する。奥室が
// 全部 back の樽物置に落ちる退行を、ゴールデンの目視より前にここで止める。
func TestFurnishBuilding_施設テンプレが本番サイズで奥室を役割へ分化する(t *testing.T) {
	t.Parallel()

	cases := []struct {
		facility string
		roles    []roleName // このどれかが必ず出る施設固有の役割
	}{
		{"store", []roleName{"storeroom", "office", "restroom", "coldroom"}},
		{"clinic", []roleName{"examination_room", "pharmacy", "restroom", "office"}},
		{"house", []roleName{"kitchen", "bedroom", "bath"}},
	}
	for _, c := range cases {
		for fp := consts.Tile(17); fp <= 20; fp++ { // 本番でテンプレが発火する footprint 範囲
			for seed := range uint64(20) {
				footprint := Rect{X: 0, Y: 0, W: fp, H: fp}
				site, _, err := FurnishBuilding(testRaws(), seed, footprint, Vec{X: fp / 2, Y: 0}, facSpec(c.facility))
				require.NoError(t, err)
				roles := map[roleName]int{}
				for _, r := range site.Rooms {
					roles[r.Role]++
				}
				assert.NotContainsf(t, roles, roleName("back"), "%s fp=%d seed=%d はテンプレを使い BSP の back を出さない", c.facility, fp, seed)
				has := false
				for _, r := range c.roles {
					if roles[r] > 0 {
						has = true
						break
					}
				}
				assert.Truef(t, has, "%s fp=%d seed=%d は施設固有の役割 %v のどれかを出す (roles=%v)", c.facility, fp, seed, c.roles, roles)
			}
		}
	}
}

// TestFurnishBuilding_部屋が退化しない は生成される部屋が内側床を必ず持つことを多 seed で固定する。前庭で
// 建物が縮むとテンプレの比率割りで薄い部屋が H<3 の内側床0に潰れ、床が描かれずラベルだけ浮く退行が出た。
// 本番の footprint 17〜20 の全 seed で全室が内側床を1タイル以上持つことを守る。玄関は街路のある北・西の
// 両辺を舐める。西玄関は前庭ぶん建物幅が縮み建物14幅になるので、狭い側の退化も捕まえる。
func TestFurnishBuilding_部屋が退化しない(t *testing.T) {
	t.Parallel()

	for _, fac := range []string{"house", "store", "clinic"} {
		for fp := consts.Tile(17); fp <= 20; fp++ {
			doors := map[string]Vec{"北": {X: fp / 2, Y: 0}, "西": {X: 0, Y: fp / 2}}
			for dside, door := range doors {
				for seed := range uint64(30) {
					footprint := Rect{X: 0, Y: 0, W: fp, H: fp}
					site, _, err := FurnishBuilding(testRaws(), seed, footprint, door, facSpec(fac))
					require.NoError(t, err)
					for _, hr := range site.Rooms {
						assert.NotEmptyf(t, hr.Room.Rect.interiorTiles(), "%s fp=%d 玄関=%s seed=%d の部屋 %s %+v が内側床を持つ", fac, fp, dside, seed, hr.Role, hr.Room.Rect)
					}
				}
			}
		}
	}
}

// TestFurnishBuilding_民家の入口は玄関に開く は建物の入口が必ず玄関の部屋へ開くことを本番サイズの多 seed で
// 固定する。玄関を建物の奥に置き入口が居室へ開いていた退行を止める。玄関は街路のある北・西の両辺を舐める。
func TestFurnishBuilding_民家の入口は玄関に開く(t *testing.T) {
	t.Parallel()

	for fp := consts.Tile(17); fp <= 20; fp++ {
		doors := map[string]Vec{"北": {X: fp / 2, Y: 0}, "西": {X: 0, Y: fp / 2}}
		for dside, door := range doors {
			for seed := range uint64(30) {
				site, _, err := FurnishBuilding(testRaws(), seed, Rect{X: 0, Y: 0, W: fp, H: fp}, door, facSpec("house"))
				require.NoError(t, err)
				var genkan *PlannedRoom
				for i := range site.Rooms {
					if site.Rooms[i].Role == "genkan" {
						genkan = &site.Rooms[i]
						break
					}
				}
				if !assert.NotNilf(t, genkan, "民家 fp=%d 玄関=%s seed=%d は玄関の部屋を持つ", fp, dside, seed) {
					continue
				}
				opensToGenkan := slices.Contains(genkan.Room.Doorways, site.Door)
				assert.Truef(t, opensToGenkan, "民家 fp=%d 玄関=%s seed=%d の入口 %+v は玄関 %+v へ開く", fp, dside, seed, site.Door, genkan.Room.Rect)
			}
		}
	}
}

// TestFurnishBuilding_民家は浴室とトイレを持ち居間より小さい は民家の間取りに浴室とトイレが必ず別室であり、
// かつ水回りが居間より小さく保たれることを本番サイズの多 seed で固定する。建物を広げた際に民家が田の字4室へ
// 落ちてトイレが消えた退行と、水回りが居室並みに広すぎた退行の両方を止める。玄関は街路のある北・西の両辺を舐める。
func TestFurnishBuilding_民家は浴室とトイレを持ち居間より小さい(t *testing.T) {
	t.Parallel()

	interiorArea := func(r Rect) int {
		w, h := r.W-2, r.H-2
		if w < 0 || h < 0 {
			return 0
		}
		return int(w * h)
	}
	const wcMaxInterior = 12 // 浴室・トイレの内側面積の上限。居室並みに広がる退行を止める
	for fp := consts.Tile(17); fp <= 20; fp++ {
		doors := map[string]Vec{"北": {X: fp / 2, Y: 0}, "西": {X: 0, Y: fp / 2}}
		for dside, door := range doors {
			for seed := range uint64(30) {
				footprint := Rect{X: 0, Y: 0, W: fp, H: fp}
				site, _, err := FurnishBuilding(testRaws(), seed, footprint, door, facSpec("house"))
				require.NoError(t, err)
				rect := map[roleName]Rect{}
				for _, hr := range site.Rooms {
					rect[hr.Role] = hr.Room.Rect
				}
				bath, hasBath := rect["bath"]
				toilet, hasToilet := rect["toilet"]
				living, hasLiving := rect["living"]
				assert.Truef(t, hasBath, "民家 fp=%d 玄関=%s seed=%d は浴室を持つ", fp, dside, seed)
				assert.Truef(t, hasToilet, "民家 fp=%d 玄関=%s seed=%d はトイレを持つ", fp, dside, seed)
				if !hasBath || !hasToilet || !hasLiving {
					continue
				}
				ab, at, al := interiorArea(bath), interiorArea(toilet), interiorArea(living)
				assert.LessOrEqualf(t, ab, wcMaxInterior, "民家 fp=%d 玄関=%s seed=%d の浴室 %+v が広すぎない", fp, dside, seed, bath)
				assert.LessOrEqualf(t, at, wcMaxInterior, "民家 fp=%d 玄関=%s seed=%d のトイレ %+v が広すぎない", fp, dside, seed, toilet)
				assert.Lessf(t, ab, al, "民家 fp=%d 玄関=%s seed=%d の浴室は居間より小さい", fp, dside, seed)
				assert.Lessf(t, at, al, "民家 fp=%d 玄関=%s seed=%d のトイレは居間より小さい", fp, dside, seed)
			}
		}
	}
}
