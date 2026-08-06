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

func TestPipeline_Apply(t *testing.T) {
	t.Parallel()

	filter, err := NewRetroFilter()
	require.NoError(t, err)

	pipeline := NewPipeline(filter)
	src := ebiten.NewImage(100, 100)
	screen := ebiten.NewImage(100, 100)

	assert.NotPanics(t, func() {
		pipeline.Apply(screen, src)
	}, "Filter を1枚かける Apply がパニックしないこと")
}

func TestPipeline_Apply_多段チェーン(t *testing.T) {
	t.Parallel()

	f1, err := NewRetroFilter()
	require.NoError(t, err)
	f2, err := NewRetroFilter()
	require.NoError(t, err)

	// 多段は中間バッファを ping-pong する。パニックせず適用できること。
	pipeline := NewPipeline(f1, f2)
	src := ebiten.NewImage(120, 80)
	screen := ebiten.NewImage(120, 80)

	assert.NotPanics(t, func() {
		pipeline.Apply(screen, src)
	})
}

func TestPipeline_Apply_フィルタなしは素通し(t *testing.T) {
	t.Parallel()

	src := ebiten.NewImage(100, 100)
	screen := ebiten.NewImage(100, 100)

	// Filter を渡さない、および nil のみの場合はどちらも素通しでパニックしない。
	assert.NotPanics(t, func() {
		NewPipeline().Apply(screen, src)
	})
	assert.NotPanics(t, func() {
		NewPipeline(nil).Apply(screen, src)
	})
}

func TestPipeline_Apply_nilレシーバは無処理(t *testing.T) {
	t.Parallel()

	var pipeline *Pipeline
	screen := ebiten.NewImage(100, 100)
	src := ebiten.NewImage(100, 100)

	assert.NotPanics(t, func() {
		pipeline.Apply(screen, src)
	}, "nil パイプラインの Apply はパニックしない")
}
