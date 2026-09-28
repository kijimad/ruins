package menuframe_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

func TestSplitList_一覧を単一ページの縦積みで返す(t *testing.T) {
	t.Parallel()

	res := resources.UIResources{Text: &resources.TextResources{}}
	rows := []menuframe.Row{
		{Cells: styled.TextCells("項目A")},
		{Cells: styled.TextCells("項目B")},
	}

	list := menuframe.SplitList(-1, rows, res)

	assert.Equal(t, []string{"項目A", "項目B"}, uicore.CollectLabels(list))
}

func TestSplitScreen_見出し_左右_説明_ヒントの順に並べる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())
	res := world.Resources.UIResources
	left := menuframe.SplitList(-1, []menuframe.Row{{Cells: styled.TextCells("左項目")}}, res)
	right := uicore.NewText("詳細本文", res.Text.BodyFace, theme.TextPrimary)

	root := menuframe.SplitScreen(world, res, "見出し", left, right, "説明文", "ヒント文")

	assert.Equal(t, []string{"見出し", "左項目", "詳細本文", "説明文", "ヒント文"}, uicore.CollectLabels(root))
}
