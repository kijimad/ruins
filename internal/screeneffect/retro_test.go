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

func TestRetroFilter_initShader_既に設定済みなら読み込み直さない(t *testing.T) {
	t.Parallel()

	filter, err := NewRetroFilter()
	require.NoError(t, err)
	shader := filter.shader

	err = filter.initShader()

	require.NoError(t, err)
	assert.Same(t, shader, filter.shader, "既存のシェーダーを上書きしないこと")
}

func TestRetroFilter_Apply_シェーダー未設定ならそのまま描画する(t *testing.T) {
	t.Parallel()

	filter := &RetroFilter{}
	src := ebiten.NewImage(2, 2)
	dst := ebiten.NewImage(2, 2)

	assert.NotPanics(t, func() {
		filter.Apply(dst, src)
	}, "シェーダーが無いときは DrawImage による素通し描画になり、パニックしないこと")
}

func TestPipeline_ensureScratch_サイズが同じなら作り直さない(t *testing.T) {
	t.Parallel()

	f1, err := NewRetroFilter()
	require.NoError(t, err)
	f2, err := NewRetroFilter()
	require.NoError(t, err)

	pipeline := NewPipeline(f1, f2)
	src := ebiten.NewImage(64, 48)
	screen := ebiten.NewImage(64, 48)

	pipeline.Apply(screen, src)
	scratch0 := pipeline.scratch[0]
	scratch1 := pipeline.scratch[1]
	require.NotNil(t, scratch0)
	require.NotNil(t, scratch1)

	pipeline.Apply(screen, src)

	assert.Same(t, scratch0, pipeline.scratch[0], "同じサイズでの再適用はバッファを作り直さないこと")
	assert.Same(t, scratch1, pipeline.scratch[1], "同じサイズでの再適用はバッファを作り直さないこと")
}

func TestPipeline_ensureScratch_サイズが変わったら作り直す(t *testing.T) {
	t.Parallel()

	f1, err := NewRetroFilter()
	require.NoError(t, err)
	f2, err := NewRetroFilter()
	require.NoError(t, err)

	pipeline := NewPipeline(f1, f2)
	pipeline.Apply(ebiten.NewImage(64, 48), ebiten.NewImage(64, 48))
	scratch0 := pipeline.scratch[0]

	pipeline.Apply(ebiten.NewImage(32, 32), ebiten.NewImage(32, 32))

	assert.NotSame(t, scratch0, pipeline.scratch[0], "サイズが変わったらバッファを作り直すこと")
}
