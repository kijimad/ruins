package styled_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/stretchr/testify/assert"
)

func TestTextCell_文字列だけを持つセルを返す(t *testing.T) {
	t.Parallel()
	got := styled.TextCell("HP")
	assert.Equal(t, styled.Cell{Text: "HP", Icon: nil}, got)
}

func TestIconCell(t *testing.T) {
	t.Parallel()

	t.Run("画像を持つセルを返す", func(t *testing.T) {
		t.Parallel()
		img := ebiten.NewImage(4, 4)
		got := styled.IconCell(img)
		assert.Equal(t, styled.Cell{Text: "", Icon: img}, got)
	})

	t.Run("nilを渡すと透明セルになり文字列は空のまま", func(t *testing.T) {
		t.Parallel()
		got := styled.IconCell(nil)
		assert.Equal(t, styled.Cell{Text: "", Icon: nil}, got)
	})
}

func TestTextCells(t *testing.T) {
	t.Parallel()

	t.Run("文字列を順にTextCellへ変換する", func(t *testing.T) {
		t.Parallel()
		got := styled.TextCells("剣", "12", "重い")
		assert.Equal(t, []styled.Cell{
			{Text: "剣"},
			{Text: "12"},
			{Text: "重い"},
		}, got)
	})

	t.Run("空で呼ぶと空スライスを返す", func(t *testing.T) {
		t.Parallel()
		got := styled.TextCells()
		assert.Empty(t, got)
	})
}
