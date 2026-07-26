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
