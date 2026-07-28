package interior

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/require"
)

// TestFurnishBuilding_同じタイルに複数のpropを置かない は、加工パイプラインが1タイルへ2つの prop を重ねない
// ことを多 seed で固定する。hero の目玉を主室中央へ据えるとき、そこに既にある食卓などの什器を退けずに足すと
// スプライトが重なる退行があった。施設と seed を広くなめ、最終配置に座標の重複が無いことを守る。
func TestFurnishBuilding_同じタイルに複数のpropを置かない(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 20, H: 18}
	door := Vec{X: 10, Y: 0}
	for _, fac := range []FacilityKind{"house", "store", "clinic", "office", "depot", "antique", "lab", ""} {
		for seed := range uint64(300) {
			_, placed := FurnishBuilding(seed, footprint, door, fac)
			seen := map[Vec]string{}
			for _, p := range placed {
				prev, dup := seen[p.Pos]
				require.Falsef(t, dup, "施設 %q seed %d: タイル %v に %q と %q が重なる", fac, seed, p.Pos, prev, p.Ref)
				seen[p.Pos] = p.Ref
			}
		}
	}
}

// TestFrontSide_内寄せが無い建物は入口の辺を前面にする は、insetBuilding が前庭を作れず建物と footprint が
// 一致するとき、外皮 FacadePass と lot pass が入口のある辺を前面に選ぶことを固定する。既定で一辺へ倒すと
// 入口と逆の壁へ窓や塀を付け、玄関の無い正面ができる退行を止める。公開 API の Furnish 経路で、入口軸が短く
// 内寄せが破綻する footprint を通して検査する。
func TestFrontSide_内寄せが無い建物は入口の辺を前面にする(t *testing.T) {
	t.Parallel()

	// 入口軸の高さが frontYard を差し引くと inset の下限を割るので、insetBuilding は footprint をそのまま返す
	full := Rect{X: 0, Y: 0, W: 20, H: 6}
	cases := []struct {
		name string
		door Vec
		want side
	}{
		{"北の入口", Vec{X: 10, Y: 0}, sideNorth},
		{"南の入口", Vec{X: 10, Y: 5}, sideSouth},
		{"西の入口", Vec{X: 0, Y: 3}, sideWest},
		{"東の入口", Vec{X: 19, Y: 3}, sideEast},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := Site{Building: full, Footprint: full, Door: tc.door}
			require.Equal(t, tc.want, frontSide(s), "入口の辺を前面にする")
		})
	}
}

// TestFurnishBuilding_内寄せ破綻の小footprintでも完走する は、footprint が小さく insetBuilding が前庭を作れず
// building==footprint になったうえで、ポーチが入口を1マス内側へ下げる seed でも生成が panic しないことを
// 固定する。frontSide の fallback が辺上に無い入口を doorSide へ渡すと panic する退行があった。本番の
// footprint は 17〜20 だが、公開 API の Furnish 経路は呼び出し側の寸法に依らずクラッシュしてはならない。
func TestFurnishBuilding_内寄せ破綻の小footprintでも完走する(t *testing.T) {
	t.Parallel()

	for _, fac := range []FacilityKind{"house", "store", "clinic", "office", "depot", "antique", "lab", ""} {
		for fp := consts.Tile(5); fp <= 9; fp++ {
			for seed := range uint64(100) {
				require.NotPanicsf(t, func() {
					FurnishBuilding(seed, Rect{X: 0, Y: 0, W: fp, H: fp}, Vec{X: fp / 2, Y: 0}, fac)
				}, "施設 %q fp=%d seed=%d の生成が panic した", fac, fp, seed)
			}
		}
	}
}
