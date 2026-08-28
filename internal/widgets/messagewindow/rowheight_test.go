package messagewindow

import (
	"testing"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
)

// TestRowHeightsFitTheirFace は窓の各行の高さが、その行へ載せるフェイスの字面を切らないかを検査する。
// 行高とフェイスは独立に変えられるので、フォントを大きくしたときに気づけるよう下限を固定する。
func TestRowHeightsFitTheirFace(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	tx := world.Resources.UIResources.Text

	tests := []struct {
		name string
		rowH int
		face text.Face
	}{
		{"選択肢の行", choiceRowH, tx.BodyFace},
		{"ページ表示の行", pageRowH, tx.SmallFace},
		{"タイトルバー", titleBarHeight, tx.SmallFace},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.GreaterOrEqual(t, tt.rowH, ui.LineHeight(tt.face),
				"行高がフェイスの字面より低い。文字の上下が切れる")
		})
	}
}
