package hud

import (
	"image/color"
	"testing"

	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/stretchr/testify/assert"
)

// TestBottomRightLineY は右下スタックの積み方を固定する。行が1つ上がるごとに1行の高さと行間ぶん
// 上へ積むこと、最下段の基準位置を検証する。具体ピクセルでなく積み方の性質で固定する。
func TestBottomRightLineY(t *testing.T) {
	t.Parallel()

	data := GameInfoData{
		MessageAreaHeight: 40,
		ScreenDimensions:  ScreenDimensions{Width: 800, Height: 600},
	}
	const textHeight = 16

	// 段が1つ上がるごとに1行の高さ + 行間ぶん Y が小さくなる
	prev := bottomRightLineY(data, textHeight, 0)
	for row := 1; row <= bottomRightRowNorthDepth; row++ {
		cur := bottomRightLineY(data, textHeight, row)
		assert.Less(t, cur, prev, "上の段ほど Y が小さい")
		assert.InDelta(t, float64(textHeight)+theme.Space2F, prev-cur, 0.001, "段の間隔は1行の高さ+行間")
		prev = cur
	}

	// 最下段 row=0 はメッセージエリアと下マージンから1行ぶん上
	want := float64(data.ScreenDimensions.Height) - float64(data.MessageAreaHeight) - theme.Space4F - float64(textHeight)
	assert.InDelta(t, want, bottomRightLineY(data, textHeight, 0), 0.001, "最下段の基準位置")
}

func TestLerpColor(t *testing.T) {
	t.Parallel()

	a := color.RGBA{R: 0, G: 0, B: 0, A: 100}
	b := color.RGBA{R: 100, G: 200, B: 50, A: 50}

	t.Run("tが0ならaをそのまま返すが不透明度は255になる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 0, G: 0, B: 0, A: 255}, lerpColor(a, b, 0))
	})

	t.Run("tが1ならbをそのまま返すが不透明度は255になる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 100, G: 200, B: 50, A: 255}, lerpColor(a, b, 1))
	})

	t.Run("tが中間なら各チャンネルを線形補間する", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 50, G: 100, B: 25, A: 255}, lerpColor(a, b, 0.5))
	})
}
