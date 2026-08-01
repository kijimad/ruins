package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestScatterArea_同じseedは同じ結果 は ScatterArea が決定的な純関数であることを固定する。
// 再訪一致と serde 安全の土台なので、同一入力で必ず同じ並びを返す。
func TestScatterArea_同じseedは同じ結果(t *testing.T) {
	t.Parallel()
	area := Rect{X: 0, Y: 0, W: 12, H: 12}
	all := func(Vec) bool { return true }

	a := ScatterArea(area, all, 42, 6)
	b := ScatterArea(area, all, 42, 6)
	assert.Equal(t, a, b, "同じ seed と入力なら同じ結果")

	c := ScatterArea(area, all, 43, 6)
	assert.NotEqual(t, a, c, "seed が違えば選ばれるタイルも変わる")
}

// TestScatterArea_acceptで候補を絞る は、置ける条件を accept 述語に委ねられることを固定する。
func TestScatterArea_acceptで候補を絞る(t *testing.T) {
	t.Parallel()
	area := Rect{X: 0, Y: 0, W: 12, H: 12}

	got := ScatterArea(area, func(v Vec) bool { return v.X%2 == 0 }, 7, 100)
	assert.NotEmpty(t, got)
	for _, v := range got {
		assert.Zero(t, v.X%2, "accept を満たすタイルだけが選ばれる: %v", v)
	}
}

// TestScatterArea_countで上限 は返す枚数が count で頭打ちになり、0 以下で空になることを固定する。
func TestScatterArea_countで上限(t *testing.T) {
	t.Parallel()
	area := Rect{X: 0, Y: 0, W: 12, H: 12}
	all := func(Vec) bool { return true }

	assert.Len(t, ScatterArea(area, all, 1, 3), 3, "count 個だけ返す")
	assert.Empty(t, ScatterArea(area, all, 1, 0), "count 0 は空")
}

// TestScatterArea_外周は選ばない は interiorTiles が外周1タイルを除くので、チャンクの継ぎ目や
// 部屋の壁際には置かれないことを固定する。
func TestScatterArea_外周は選ばない(t *testing.T) {
	t.Parallel()
	area := Rect{X: 0, Y: 0, W: 5, H: 5}

	got := ScatterArea(area, func(Vec) bool { return true }, 1, 100)
	assert.NotEmpty(t, got)
	for _, v := range got {
		assert.True(t, v.X > 0 && v.X < 4 && v.Y > 0 && v.Y < 4, "外周1タイルは選ばない: %v", v)
	}
}
