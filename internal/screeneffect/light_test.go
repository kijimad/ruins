package screeneffect

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLightFilter(t *testing.T) {
	t.Parallel()

	filter, err := NewLightFilter()
	require.NoError(t, err, "フィルタの作成に失敗")
	require.NotNil(t, filter, "フィルタがnilではないこと")
}

func TestLightFilter_Apply(t *testing.T) {
	t.Parallel()

	filter, err := NewLightFilter()
	require.NoError(t, err)

	src := ebiten.NewImage(120, 90)
	dst := ebiten.NewImage(120, 90)

	assert.NotPanics(t, func() {
		filter.Apply(dst, src)
	}, "Applyでパニックが発生しないこと")
}

func TestLightFilter_Apply_nilは無処理(t *testing.T) {
	t.Parallel()

	var filter *LightFilter
	src := ebiten.NewImage(120, 90)
	dst := ebiten.NewImage(120, 90)

	assert.NotPanics(t, func() {
		filter.Apply(dst, src)
	}, "nilレシーバのApplyはパニックしない")
}
