package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFurnish_施設種別ごとに決定的に内装を返す は公開入口の決定性を固定する。どの施設種別でも何かを置き、
// 同じ引数なら完全一致する。overworld の建物furnishが再訪で一致する前提。
func TestFurnish_施設種別ごとに決定的に内装を返す(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 16, H: 12}
	door := Vec{X: 8, Y: 11}
	for _, fac := range []string{"house", "store", "clinic", "office", "depot", "antique", "lab", "unknown"} {
		first := Furnish(3, footprint, door, fac)
		require.NotEmptyf(t, first, "%s は何か配置する", fac)
		require.Equalf(t, first, Furnish(3, footprint, door, fac), "%s は同じ引数で完全一致する", fac)
	}
}

// TestFurnish_密度と経年が建物ごとに変わる は密度プロファイルと経年 condition の直交軸を固定する。seed を
// 振ると家具の数がばらつき、経年した建物と手つかずの建物の両方が出る。同じ施設でも建物ごとに違って見える。
func TestFurnish_密度と経年が建物ごとに変わる(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 18, H: 14}
	door := Vec{X: 9, Y: 13}
	counts := make(map[int]bool)
	aged, pristine := false, false
	for seed := range uint64(30) {
		if buildingAged(seed) {
			aged = true
		} else {
			pristine = true
		}
		n := 0
		for _, p := range Furnish(seed, footprint, door, "store") {
			if p.Kind == KindFurniture {
				n++
			}
		}
		counts[n] = true
	}
	assert.GreaterOrEqual(t, len(counts), 3, "密度と抽選で家具数が建物ごとにばらつく")
	assert.True(t, aged, "経年した建物が出る")
	assert.True(t, pristine, "手つかずの建物も出る")
}

// TestFacilityContent_seedで店の変種が変わる は content 変種の抽選を固定する。seed を振ると同じ店でも
// コンビニ・薬局・食料品店の別々の内装が出て、いずれも店として分類される。データを足さず配合を変える
// 変種で、同じ施設種別に多様性が出ることを守る。
func TestFacilityContent_seedで店の変種が変わる(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 16, H: 12}
	door := Vec{X: 8, Y: 11}
	ids := make(map[string]bool)
	for seed := range uint64(30) {
		ids[facilityContent("store", seed).ID] = true
		assert.Equalf(t, "store", classifyRoom(Furnish(seed, footprint, door, "store")), "seed=%d のどの変種も店に分類される", seed)
	}
	assert.GreaterOrEqual(t, len(ids), 2, "seed を振ると店の変種が複数出る")
}

// TestFurnish_家具は施設種別どおりに分類される は Furnish と classifyRoom を突き合わせる。生成した内装を
// 逆推定した役割が施設種別と一致し、店は店・診療所は診療所・民家は寝室系に見える。
func TestFurnish_家具は施設種別どおりに分類される(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 16, H: 12}
	door := Vec{X: 8, Y: 11}
	cases := []struct{ facility, role string }{
		{"store", "store"},
		{"clinic", "clinic"},
		{"house", "bedroom"},
		{"depot", "storage"},
	}
	for _, c := range cases {
		placed := Furnish(3, footprint, door, c.facility)
		assert.Equalf(t, c.role, classifyRoom(placed), "%s は %s に分類される", c.facility, c.role)
	}
}
