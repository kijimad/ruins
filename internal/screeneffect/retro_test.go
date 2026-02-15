package screeneffect

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRetroFilter(t *testing.T) {
	t.Parallel()

	filter, err := NewRetroFilter()
	require.NoError(t, err, "フィルタの作成に失敗")
	require.NotNil(t, filter, "フィルタがnilではないこと")
}

func TestRetroFilter_Apply(t *testing.T) {
	t.Parallel()

	filter, err := NewRetroFilter()
	require.NoError(t, err)

	src := ebiten.NewImage(100, 100)
	dst := ebiten.NewImage(100, 100)

	// パニックしないことを確認
	assert.NotPanics(t, func() {
		filter.Apply(dst, src)
	}, "Applyでパニックが発生しないこと")
}

func TestPipeline_BeginEnd(t *testing.T) {
	t.Parallel()

	filter, err := NewRetroFilter()
	require.NoError(t, err)

	pipeline := NewPipeline(filter)

	// Beginでオフスクリーンバッファを取得
	offscreen := pipeline.Begin(100, 100)
	require.NotNil(t, offscreen, "オフスクリーンバッファがnilではないこと")

	bounds := offscreen.Bounds()
	assert.Equal(t, 100, bounds.Dx(), "幅が正しいこと")
	assert.Equal(t, 100, bounds.Dy(), "高さが正しいこと")

	// Endで画面に描画
	screen := ebiten.NewImage(100, 100)
	pipeline.End(screen)
}

func TestPipeline_ResizeBuffer(t *testing.T) {
	t.Parallel()

	filter, err := NewRetroFilter()
	require.NoError(t, err)

	pipeline := NewPipeline(filter)

	// 最初のサイズ
	offscreen1 := pipeline.Begin(100, 100)
	require.NotNil(t, offscreen1)

	// サイズ変更
	offscreen2 := pipeline.Begin(200, 150)
	require.NotNil(t, offscreen2)

	bounds := offscreen2.Bounds()
	assert.Equal(t, 200, bounds.Dx(), "新しい幅が正しいこと")
	assert.Equal(t, 150, bounds.Dy(), "新しい高さが正しいこと")
}
