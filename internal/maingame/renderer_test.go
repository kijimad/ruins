package maingame

import (
	"testing"

	"github.com/kijimaD/ruins/internal/screeneffect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRenderer_パイプラインを保持しフレーム未生成で始まる は、
// newRenderer が渡されたパイプラインをそのまま保持し、フレームをまだ確保していないことを確認する。
func TestNewRenderer_パイプラインを保持しフレーム未生成で始まる(t *testing.T) {
	t.Parallel()

	pipeline := screeneffect.NewPipeline()
	r := newRenderer(pipeline)

	assert.Same(t, pipeline, r.pipeline)
	assert.Nil(t, r.frame)
}

// TestEnsureFrame は、フレームバッファがサイズ変更時にだけ作り直されることを確認する。
func TestEnsureFrame(t *testing.T) {
	t.Parallel()

	t.Run("未確保なら新規に確保する", func(t *testing.T) {
		t.Parallel()

		r := newRenderer(screeneffect.NewPipeline())
		r.ensureFrame(100, 50)

		require.NotNil(t, r.frame)
		assert.Equal(t, 100, r.frameW)
		assert.Equal(t, 50, r.frameH)
	})

	t.Run("同じサイズなら作り直さない", func(t *testing.T) {
		t.Parallel()

		r := newRenderer(screeneffect.NewPipeline())
		r.ensureFrame(100, 50)
		frame := r.frame

		r.ensureFrame(100, 50)

		assert.Same(t, frame, r.frame)
	})

	t.Run("サイズが変われば作り直す", func(t *testing.T) {
		t.Parallel()

		r := newRenderer(screeneffect.NewPipeline())
		r.ensureFrame(100, 50)
		frame := r.frame

		r.ensureFrame(200, 80)

		assert.NotSame(t, frame, r.frame)
		assert.Equal(t, 200, r.frameW)
		assert.Equal(t, 80, r.frameH)
	})
}
