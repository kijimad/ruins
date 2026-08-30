package messagewindow

import (
	"testing"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// TestRowHeightsFitTheirFace は窓の各行の高さが、その行へ載せるフェイスの字面を切らないかを検査する。
// 行高とフェイスは独立に変えられるので、フォントを大きくしたときに気づけるよう下限を固定する。
func TestRowHeightsFitTheirFace(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
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
	// 下位テストに割らず1つの中で回す。フェイスは world ごとに独立するが、この world の中では
	// 共有で、text/v2 のフェイスは内部に可変キャッシュを持つため並行に測ると競合する
	for _, tt := range tests {
		assert.GreaterOrEqual(t, tt.rowH, uicore.LineHeight(tt.face),
			"%s の行高がフェイスの字面より低い。文字の上下が切れる", tt.name)
	}
}
