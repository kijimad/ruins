package menuframe

import (
	"testing"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/theme"
)

// TestRowHeightsFitTheirFace は行高の定数が、その行へ載せるフェイスの字面を切らないかを検査する。
//
// 行高はフェイスの大きさから導いておらず、どちらも独立に変えられる。フォントを大きくすると
// 行が字面より低くなり、文字の上下が切れる。見た目の破綻は golden を撮り直すまで気づけないので、
// 下限だけをここで固定する。上限は置かない。行をゆったり取るのは意匠の判断で、破綻ではない。
func TestRowHeightsFitTheirFace(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
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
		{"入力画面の補足行", formLineH, tx.SmallFace},
		{"画面下部の補助行", noteRowH, tx.SmallFace},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.GreaterOrEqual(t, tt.rowH, ui.LineHeight(tt.face),
				"行高がフェイスの字面より低い。文字の上下が切れる")
		})
	}
}
