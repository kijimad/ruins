package hud

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
)

// TestLogLineHeightFitsFace はログ領域の行送りが本文フェイスの字面を切らないかを検査する。
// 行送りとフェイスは独立に変えられるので、フォントを大きくしたときに気づけるよう下限を固定する。
func TestLogLineHeightFitsFace(t *testing.T) {
	t.Parallel()
	res, err := loader.LoadUIResources()
	require.NoError(t, err)

	assert.GreaterOrEqual(t, DefaultMessageAreaConfig.LineHeight, ui.LineHeight(res.Text.BodyFace),
		"行送りがフェイスの字面より低い。ログの行が重なる")
}
