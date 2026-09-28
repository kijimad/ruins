package menuframe

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// resolveColWidths は非公開関数なので、この internal test だけがモード別の分岐を直接検証できる。

func TestResolveColWidths_Icon列は正方の固定幅(t *testing.T) {
	t.Parallel()

	cols := []styled.Col{styled.Icon()}
	widths := resolveColWidths(cols, nil, nil, nil)

	assert.Equal(t, []int{theme.MenuIconW}, widths, "Icon列は内容に関わらずMenuIconWの固定幅になる")
}

func TestResolveColWidths_Grow列は0のままにする(t *testing.T) {
	t.Parallel()

	cols := []styled.Col{styled.Name()}
	rows := []Row{{Cells: styled.TextCells("とても長い項目名がここに入る想定の文字列")}}
	widths := resolveColWidths(cols, nil, rows, nil)

	assert.Equal(t, []int{0}, widths, "Grow列は行のflexが余り幅を割り当てるのでここでは0のまま")
}

func TestResolveColWidths_Fit列のアイコンは画像の実幅で測る(t *testing.T) {
	t.Parallel()

	icon := ebiten.NewImage(30, 12)
	cols := []styled.Col{styled.Fit()}
	rows := []Row{{Cells: []styled.Cell{styled.IconCell(icon)}}}
	widths := resolveColWidths(cols, nil, rows, nil)

	assert.Equal(t, []int{30 + theme.Space3}, widths, "Fit列のアイコンセルは正方に丸めず横長のまま画像の実幅で測る")
}

func TestResolveColWidths_列数より少ないセルの行は読み飛ばす(t *testing.T) {
	t.Parallel()

	icon := ebiten.NewImage(25, 25)
	cols := []styled.Col{styled.Name(), styled.Fit()}
	rows := []Row{
		// 1列目のセルしか持たない行。2列目の実測から外れる
		{Cells: styled.TextCells("項目のみ")},
		{Cells: []styled.Cell{styled.TextCell("項目"), styled.IconCell(icon)}},
	}
	widths := resolveColWidths(cols, nil, rows, nil)

	assert.Equal(t, []int{0, 25 + theme.Space3}, widths, "セルを持たない行はpanicせず読み飛ばし、2列目の幅は残る行のアイコンから決まる")
}

func TestResolveColWidths_見出しもFit列の実測に加える(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())
	face := world.Resources.UIResources.Text.BodyFace

	icon := ebiten.NewImage(5, 5)
	cols := []styled.Col{styled.Fit()}
	headerRow := []string{"非常に長い見出し文字列テスト"}
	rows := []Row{{Cells: []styled.Cell{styled.IconCell(icon)}}}

	widths := resolveColWidths(cols, headerRow, rows, face)

	wantHeaderW := uicore.MeasureTextWidth(headerRow[0], face) + theme.Space3
	assert.Equal(t, []int{wantHeaderW}, widths, "見出しがアイコンより広ければ見出しの実測幅が列幅を決める")
	assert.Greater(t, widths[0], 5+theme.Space3, "見出しがアイコンより広いことの前提")
}
