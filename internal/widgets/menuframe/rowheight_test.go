package menuframe

import (
	"testing"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/internal/uicore"
	"github.com/kijimaD/ruins/internal/widgets/theme"
)

// TestRowHeightsFitTheirFace は行高の定数が、その行へ載せるフェイスの字面を切らないかを検査する。
//
// 行高はフェイスの大きさから導いておらず、どちらも独立に変えられる。フォントを大きくすると
// 行が字面より低くなり、文字の上下が切れる。見た目の破綻は golden を撮り直すまで気づけないので、
// 下限だけをここで固定する。上限は置かない。行をゆったり取るのは意匠の判断で、破綻ではない。
func TestRowHeightsFitTheirFace(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	tx := world.Resources.UIResources.Text

	tests := []struct {
		name string
		rowH int
		face text.Face
	}{
		{"タブ画面の密な行", theme.MenuTabRowH, tx.BodyFace},
		{"パネル画面のコマンド行", theme.MenuPanelRowH, tx.BodyFace},
		{"入力画面の見出し", formTitleH, tx.TitleFontFace},
		{"入力画面の入力欄", formInputH, tx.BodyFace},
		{"補助フェイスの1行", noteRowH, tx.SmallFace},
	}
	// 下位テストに割らず1つの中で回す。フェイスは world ごとに独立するが、この world の中では
	// 共有で、text/v2 のフェイスは内部に可変キャッシュを持つため並行に測ると競合する
	for _, tt := range tests {
		assert.GreaterOrEqual(t, tt.rowH, uicore.LineHeight(tt.face),
			"%s の行高がフェイスの字面より低い。文字の上下が切れる", tt.name)
	}
}
