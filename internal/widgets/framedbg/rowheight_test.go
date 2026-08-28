package framedbg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
)

// TestInfoPanelLineHFitsFace は情報パネルの行送りが本文フェイスの字面を切らないかを検査する。
// 行送りとフェイスは独立に変えられるので、フォントを大きくしたときに気づけるよう下限を固定する。
func TestInfoPanelLineHFitsFace(t *testing.T) {
	t.Parallel()
	fonts, err := loader.LoadFonts()
	require.NoError(t, err)
	res, err := loader.LoadUIResources(fonts)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, LineH, ui.LineHeight(res.Text.BodyFace),
		"行送りがフェイスの字面より低い。行が重なる")
}
