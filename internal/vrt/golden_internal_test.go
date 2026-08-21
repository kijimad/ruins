package vrt

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToleranceForSize はトレランスが noiseScale / √面積 であることと、面積が増えるほど
// 比率が下がることを固定する。noiseScale の実数値には依存させない。係数の調整はノイズ実測に
// 応じて起きるので、値を焼き込むとその都度テストを書き換えることになる
func TestToleranceForSize(t *testing.T) {
	t.Parallel()

	t.Run("noiseScaleを面積の平方根で割った値を返す", func(t *testing.T) {
		t.Parallel()
		got := toleranceForSize(960, 720)
		assert.InDelta(t, noiseScale/math.Sqrt(960*720), got, 1e-12)
	})

	t.Run("面積が増えるほど比率が下がる", func(t *testing.T) {
		t.Parallel()
		sizes := [][2]int{{300, 30}, {400, 120}, {640, 480}, {960, 720}}

		prev := math.Inf(1)
		for _, s := range sizes {
			got := toleranceForSize(s[0], s[1])
			assert.Less(t, got, prev, "%dx%d は1つ小さい画像より低い比率になる", s[0], s[1])
			prev = got
		}
	})

	t.Run("ゼロサイズは0を返す", func(t *testing.T) {
		t.Parallel()
		assert.Zero(t, toleranceForSize(0, 0))
	})

	t.Run("フルスクリーンでもメニュー1行規模の変化を検出できる比率になっている", func(t *testing.T) {
		t.Parallel()
		// ラベル2行の書き換えが 0.0825% だった実測に対し、余裕を持って下回っていること。
		// 係数を緩めすぎると実変化が静かに素通りするので、上限を仕込んで気づけるようにする
		got := toleranceForSize(960, 720)
		require.Less(t, got, 0.0825/100/2, "メニュー行の変化を見逃さない比率であること")
	})
}
