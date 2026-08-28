package entityspec

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
)

// TestSpecPanelRowHFitsFace は spec パネルの行高が補助フェイスの字面を切らないかを検査する。
// 行高とフェイスは独立に変えられるので、フォントを大きくしたときに気づけるよう下限を固定する。
func TestSpecPanelRowHFitsFace(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	assert.GreaterOrEqual(t, specPanelRowH, ui.LineHeight(world.Resources.UIResources.Text.SmallFace),
		"行高がフェイスの字面より低い。文字の上下が切れる")
}
